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

// buildMotionItem converts a model Motion and its associated BinaryBlob to a template MotionItem.
func buildMotionItem(m *model.Motion, blob *model.BinaryBlob) templates.MotionItem {
	item := templates.MotionItem{
		ID:           m.ID,
		IDString:     strconv.FormatInt(m.ID, 10),
		BlobID:       blob.ID,
		BlobIDString: strconv.FormatInt(blob.ID, 10),
		Title:        m.Title,
		Filename:     blob.Filename,
	}
	if m.VotesFor != nil {
		item.VotesFor = m.VotesFor
		item.VotesAgainst = m.VotesAgainst
		item.VotesAbstained = m.VotesAbstained
		item.VotesEligible = m.VotesEligible
	}
	return item
}

// loadMotionListPartial loads motions for a single agenda point and returns the partial input.
func (h *Handler) loadMotionListPartial(ctx context.Context, slug, meetingIDStr string, ap *model.AgendaPoint) (*templates.MotionListPartialInput, error) {
	apIDStr := strconv.FormatInt(ap.ID, 10)

	motions, err := h.Repository.ListMotionsForAgendaPoint(ctx, ap.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load motions for agenda point %d: %w", ap.ID, err)
	}

	motionItems := make([]templates.MotionItem, 0, len(motions))
	for _, m := range motions {
		blob, err := h.Repository.GetBlobByID(ctx, m.BlobID)
		if err != nil {
			return nil, fmt.Errorf("failed to load blob for motion %d: %w", m.ID, err)
		}
		motionItems = append(motionItems, buildMotionItem(m, blob))
	}

	return &templates.MotionListPartialInput{
		CommitteeSlug:    slug,
		MeetingIDString:  meetingIDStr,
		AgendaPointID:    ap.ID,
		AgendaPointIDStr: apIDStr,
		AgendaPointTitle: ap.Title,
		Motions:          motionItems,
	}, nil
}

// ManageMotionCreate handles file upload and creates a new motion for an agenda point.
func (h *Handler) ManageMotionCreate(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MotionListPartialInput, *routes.ResponseMeta, error) {
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	title := r.FormValue("title")

	ap, err := h.Repository.GetAgendaPointByID(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda point: %w", err)
	}

	if title == "" {
		partial, loadErr := h.loadMotionListPartial(ctx, params.Slug, params.MeetingId, ap)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = "Title is required."
		return partial, nil, nil
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		partial, loadErr := h.loadMotionListPartial(ctx, params.Slug, params.MeetingId, ap)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = "Document is required."
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

	if _, err := h.Repository.CreateMotion(ctx, apID, blob.ID, title); err != nil {
		_ = h.Repository.DeleteBlob(ctx, blob.ID)
		_ = h.Storage.Delete(storagePath)
		return nil, nil, fmt.Errorf("failed to create motion: %w", err)
	}

	partial, err := h.loadMotionListPartial(ctx, params.Slug, params.MeetingId, ap)
	return partial, nil, err
}

// ManageMotionDelete removes a motion and its associated blob and file.
func (h *Handler) ManageMotionDelete(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MotionListPartialInput, *routes.ResponseMeta, error) {
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}

	motionID, err := strconv.ParseInt(params.MotionId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid motion ID")
	}

	motion, err := h.Repository.GetMotionByID(ctx, motionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load motion: %w", err)
	}

	blob, err := h.Repository.GetBlobByID(ctx, motion.BlobID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load blob: %w", err)
	}

	if err := h.Repository.DeleteMotion(ctx, motionID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete motion: %w", err)
	}

	if err := h.Repository.DeleteBlob(ctx, blob.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete blob record: %w", err)
	}

	_ = h.Storage.Delete(blob.StoragePath)

	ap, err := h.Repository.GetAgendaPointByID(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda point: %w", err)
	}

	partial, err := h.loadMotionListPartial(ctx, params.Slug, params.MeetingId, ap)
	return partial, nil, err
}

// ManageMotionRecordVote records a vote tally for a motion and returns the updated motion item partial.
func (h *Handler) ManageMotionRecordVote(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MotionItemPartialInput, *routes.ResponseMeta, error) {
	motionID, err := strconv.ParseInt(params.MotionId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid motion ID")
	}

	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	votesFor, err := strconv.ParseInt(r.FormValue("votes_for"), 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid votes_for value")
	}
	votesAgainst, err := strconv.ParseInt(r.FormValue("votes_against"), 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid votes_against value")
	}
	votesAbstained, err := strconv.ParseInt(r.FormValue("votes_abstained"), 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid votes_abstained value")
	}
	votesEligible, err := strconv.ParseInt(r.FormValue("votes_eligible"), 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid votes_eligible value")
	}

	if err := h.Repository.SetMotionVotes(ctx, motionID, votesFor, votesAgainst, votesAbstained, votesEligible); err != nil {
		return nil, nil, fmt.Errorf("failed to record votes: %w", err)
	}

	motion, err := h.Repository.GetMotionByID(ctx, motionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to reload motion: %w", err)
	}

	blob, err := h.Repository.GetBlobByID(ctx, motion.BlobID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load blob: %w", err)
	}

	motionItem := buildMotionItem(motion, blob)
	return &templates.MotionItemPartialInput{
		CommitteeSlug:    params.Slug,
		MeetingIDString:  params.MeetingId,
		AgendaPointIDStr: params.AgendaPointId,
		Motion:           motionItem,
	}, nil, nil
}

// ServeBlobDownload streams a stored file to the client.
func (h *Handler) ServeBlobDownload(w http.ResponseWriter, r *http.Request, params routes.RouteParams) error {
	blobID, err := strconv.ParseInt(params.BlobId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid blob ID")
	}

	blob, err := h.Repository.GetBlobByID(r.Context(), blobID)
	if err != nil {
		return fmt.Errorf("failed to load blob: %w", err)
	}

	rc, err := h.Storage.Open(blob.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to open stored file: %w", err)
	}
	defer rc.Close()

	w.Header().Set("Content-Type", blob.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, blob.Filename))
	w.Header().Set("Content-Length", strconv.FormatInt(blob.SizeBytes, 10))

	if _, err := io.Copy(w, rc); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}
