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

func (h *Handler) resolveCurrentDocumentBlob(ctx context.Context, ap *model.AgendaPoint) (*model.BinaryBlob, error) {
	if ap.CurrentAttachmentID != nil {
		attachment, err := h.Repository.GetAttachmentByID(ctx, *ap.CurrentAttachmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to load current attachment: %w", err)
		}
		blob, err := h.Repository.GetBlobByID(ctx, attachment.BlobID)
		if err != nil {
			return nil, fmt.Errorf("failed to load attachment blob: %w", err)
		}
		return blob, nil
	}

	return nil, nil
}

func (h *Handler) loadCurrentDocInfoForAgendaPoint(ctx context.Context, ap *model.AgendaPoint) (*templates.LiveCurrentDocInfo, error) {
	blob, err := h.resolveCurrentDocumentBlob(ctx, ap)
	if err != nil {
		return nil, err
	}
	if blob == nil {
		return nil, nil
	}
	return &templates.LiveCurrentDocInfo{
		BlobID:      blob.ID,
		ContentType: blob.ContentType,
		Filename:    blob.Filename,
	}, nil
}

func (h *Handler) loadAgendaPointForMeeting(ctx context.Context, meetingID, agendaPointID int64) (*model.AgendaPoint, error) {
	ap, err := h.Repository.GetAgendaPointByID(ctx, agendaPointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load agenda point: %w", err)
	}
	if ap.MeetingID != meetingID {
		return nil, fmt.Errorf("agenda point does not belong to meeting")
	}
	return ap, nil
}

// ServeCurrentDocument streams the currently selected agenda-point document for the meeting.
func (h *Handler) ServeCurrentDocument(w http.ResponseWriter, r *http.Request, params routes.RouteParams) error {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid meeting ID")
	}

	meeting, err := h.Repository.GetMeetingByID(r.Context(), meetingID)
	if err != nil {
		return fmt.Errorf("failed to load meeting: %w", err)
	}
	if meeting.CurrentAgendaPointID == nil {
		http.NotFound(w, r)
		return nil
	}

	ap, err := h.Repository.GetAgendaPointByID(r.Context(), *meeting.CurrentAgendaPointID)
	if err != nil {
		return fmt.Errorf("failed to load agenda point: %w", err)
	}

	blob, err := h.resolveCurrentDocumentBlob(r.Context(), ap)
	if err != nil {
		return err
	}
	if blob == nil {
		http.NotFound(w, r)
		return nil
	}

	rc, err := h.Storage.Open(blob.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to open stored file: %w", err)
	}
	defer rc.Close()

	w.Header().Set("Content-Type", blob.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename=%q`, blob.Filename))
	w.Header().Set("Content-Length", strconv.FormatInt(blob.SizeBytes, 10))

	if _, err := io.Copy(w, rc); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}

// SetCurrentAttachment marks an attachment as the active live document for an agenda point.
func (h *Handler) SetCurrentAttachment(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttachmentListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	agendaPointID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}
	attachmentID, err := strconv.ParseInt(params.AttachmentId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attachment ID")
	}

	ap, err := h.loadAgendaPointForMeeting(ctx, meetingID, agendaPointID)
	if err != nil {
		return nil, nil, err
	}
	attachment, err := h.Repository.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attachment: %w", err)
	}
	if attachment.AgendaPointID != ap.ID {
		return nil, nil, fmt.Errorf("attachment does not belong to agenda point")
	}

	if err := h.Repository.SetCurrentAttachment(ctx, ap.ID, attachmentID); err != nil {
		return nil, nil, fmt.Errorf("failed to set current attachment: %w", err)
	}
	h.publishCurrentDocumentChanged(meetingID)

	ap, err = h.Repository.GetAgendaPointByID(ctx, ap.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to reload agenda point: %w", err)
	}

	partial, err := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, ap)
	return partial, nil, err
}

// ClearCurrentDocument clears the active live document and refreshes the attachment partial.
func (h *Handler) ClearCurrentDocument(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttachmentListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	agendaPointID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}

	ap, err := h.loadAgendaPointForMeeting(ctx, meetingID, agendaPointID)
	if err != nil {
		return nil, nil, err
	}

	if err := h.Repository.ClearCurrentDocument(ctx, ap.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to clear current document: %w", err)
	}
	h.publishCurrentDocumentChanged(meetingID)

	ap, err = h.Repository.GetAgendaPointByID(ctx, ap.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to reload agenda point: %w", err)
	}

	partial, err := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, ap)
	return partial, nil, err
}
