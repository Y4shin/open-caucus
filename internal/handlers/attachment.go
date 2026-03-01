package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// buildAttachmentItem converts a model AgendaAttachment and its blob to a template AttachmentItem.
func buildAttachmentItem(a *model.AgendaAttachment, blob *model.BinaryBlob) templates.AttachmentItem {
	item := templates.AttachmentItem{
		ID:           a.ID,
		IDString:     strconv.FormatInt(a.ID, 10),
		BlobID:       blob.ID,
		BlobIDString: strconv.FormatInt(blob.ID, 10),
		Filename:     blob.Filename,
	}
	if a.Label != nil {
		item.Label = *a.Label
	}
	return item
}

// loadAttachmentListPartial loads attachments for a single agenda point and returns the partial input.
func (h *Handler) loadAttachmentListPartial(ctx context.Context, slug, meetingIDStr string, ap *model.AgendaPoint) (*templates.AttachmentListPartialInput, error) {
	apIDStr := strconv.FormatInt(ap.ID, 10)

	attachments, err := h.Repository.ListAttachmentsForAgendaPoint(ctx, ap.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attachments for agenda point %d: %w", ap.ID, err)
	}

	items := make([]templates.AttachmentItem, 0, len(attachments))
	for _, a := range attachments {
		blob, err := h.Repository.GetBlobByID(ctx, a.BlobID)
		if err != nil {
			return nil, fmt.Errorf("failed to load blob for attachment %d: %w", a.ID, err)
		}
		items = append(items, buildAttachmentItem(a, blob))
	}

	return &templates.AttachmentListPartialInput{
		CommitteeSlug:       slug,
		MeetingIDString:     meetingIDStr,
		AgendaPointIDStr:    apIDStr,
		AgendaPointTitle:    ap.Title,
		Attachments:         items,
		CurrentAttachmentID: ap.CurrentAttachmentID,
	}, nil
}

// ManageAttachmentCreate handles file upload and creates a new attachment for an agenda point.
func (h *Handler) ManageAttachmentCreate(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttachmentListPartialInput, *routes.ResponseMeta, error) {
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	ap, err := h.Repository.GetAgendaPointByID(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda point: %w", err)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		partial, loadErr := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, ap)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = "File is required."
		return partial, nil, nil
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	storagePath, sizeBytes, err := h.Storage.Store(header.Filename, contentType, file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to store file: %w", err)
	}

	blob, err := h.Repository.CreateBlob(ctx, header.Filename, contentType, sizeBytes, storagePath)
	if err != nil {
		_ = h.Storage.Delete(storagePath)
		return nil, nil, fmt.Errorf("failed to create blob record: %w", err)
	}

	var label *string
	if raw := r.FormValue("label"); raw != "" {
		label = &raw
	}

	if _, err := h.Repository.CreateAttachment(ctx, apID, blob.ID, label); err != nil {
		_ = h.Repository.DeleteBlob(ctx, blob.ID)
		_ = h.Storage.Delete(storagePath)
		return nil, nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	partial, err := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, ap)
	return partial, nil, err
}

// ManageAttachmentDelete removes an attachment and its associated blob and stored file.
func (h *Handler) ManageAttachmentDelete(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttachmentListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}

	attachmentID, err := strconv.ParseInt(params.AttachmentId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attachment ID")
	}

	attachment, err := h.Repository.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attachment: %w", err)
	}

	blob, err := h.Repository.GetBlobByID(ctx, attachment.BlobID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load blob: %w", err)
	}

	if err := h.Repository.DeleteAttachment(ctx, attachmentID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete attachment: %w", err)
	}

	if err := h.Repository.DeleteBlob(ctx, blob.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete blob record: %w", err)
	}

	_ = h.Storage.Delete(blob.StoragePath)
	h.publishCurrentDocumentChanged(meetingID)

	ap, err := h.Repository.GetAgendaPointByID(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda point: %w", err)
	}

	partial, err := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, ap)
	return partial, nil, err
}

// ServeBlobDownload streams an attachment blob by blob ID.
func (h *Handler) ServeBlobDownload(w http.ResponseWriter, r *http.Request, params routes.RouteParams) error {
	blobID, err := strconv.ParseInt(params.BlobId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid blob ID")
	}

	blob, err := h.Repository.GetBlobByID(r.Context(), blobID)
	if err != nil {
		return fmt.Errorf("failed to load blob: %w", err)
	}

	reader, err := h.Storage.Open(blob.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to open blob: %w", err)
	}
	defer reader.Close()

	w.Header().Set("Content-Type", blob.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", blob.Filename))
	w.Header().Set("Content-Length", strconv.FormatInt(blob.SizeBytes, 10))
	if _, err := io.Copy(w, reader); err != nil {
		return fmt.Errorf("failed to stream blob: %w", err)
	}
	return nil
}
