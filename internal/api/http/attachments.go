package apihttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/storage"
)

type attachmentUploadHandler struct {
	repo  repository.Repository
	store storage.Service
}

func NewAttachmentUploadHandler(repo repository.Repository, store storage.Service) http.Handler {
	return &attachmentUploadHandler{repo: repo, store: store}
}

func (h *attachmentUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := r.PathValue("slug")
	meetingID, err := strconv.ParseInt(r.PathValue("meetingId"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid meeting id")
		return
	}
	agendaPointID, err := strconv.ParseInt(r.PathValue("agendaPointId"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid agenda point id")
		return
	}

	if err := requireChairperson(r, h.repo, slug); err != nil {
		writeJSONError(w, apiHTTPStatus(err), err.Error())
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	ap, err := h.repo.GetAgendaPointByID(r.Context(), agendaPointID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "agenda point not found")
		return
	}
	if ap.MeetingID != meetingID {
		writeJSONError(w, http.StatusBadRequest, "agenda point does not belong to meeting")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	storagePath, sizeBytes, err := h.store.Store(header.Filename, contentType, file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	blob, err := h.repo.CreateBlob(r.Context(), header.Filename, contentType, sizeBytes, storagePath)
	if err != nil {
		_ = h.store.Delete(storagePath)
		writeJSONError(w, http.StatusInternalServerError, "failed to create blob record")
		return
	}

	var label *string
	if raw := strings.TrimSpace(r.FormValue("label")); raw != "" {
		label = &raw
	}

	attachment, err := h.repo.CreateAttachment(r.Context(), agendaPointID, blob.ID, label)
	if err != nil {
		_ = h.repo.DeleteBlob(r.Context(), blob.ID)
		_ = h.store.Delete(storagePath)
		writeJSONError(w, http.StatusInternalServerError, "failed to create attachment")
		return
	}

	labelValue := ""
	if label != nil {
		labelValue = *label
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"attachment_id": attachment.ID,
		"blob_id":       blob.ID,
		"filename":      blob.Filename,
		"label":         labelValue,
		"download_url":  fmt.Sprintf("/blobs/%d/download", blob.ID),
	})
}

type blobDownloadHandler struct {
	repo  repository.Repository
	store storage.Service
}

func NewBlobDownloadHandler(repo repository.Repository, store storage.Service) http.Handler {
	return &blobDownloadHandler{repo: repo, store: store}
}

func (h *blobDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blobID, err := strconv.ParseInt(r.PathValue("blobId"), 10, 64)
	if err != nil {
		http.Error(w, "invalid blob id", http.StatusBadRequest)
		return
	}

	blob, err := h.repo.GetBlobByID(r.Context(), blobID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	reader, err := h.store.Open(blob.StoragePath)
	if err != nil {
		http.Error(w, "failed to open blob", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", blob.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", blob.Filename))
	w.Header().Set("Content-Length", strconv.FormatInt(blob.SizeBytes, 10))
	if _, err := io.Copy(w, reader); err != nil {
		http.Error(w, "failed to stream blob", http.StatusInternalServerError)
		return
	}
}

func requireChairperson(r *http.Request, repo repository.Repository, committeeSlug string) error {
	sd, ok := session.GetSession(r.Context())
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}

	account, err := repo.GetAccountByID(r.Context(), *sd.AccountID)
	if err != nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account not found")
	}
	if account.IsAdmin {
		return nil
	}

	membership, err := repo.GetUserMembershipByAccountIDAndSlug(r.Context(), *sd.AccountID, committeeSlug)
	if err != nil || membership.Role != "chairperson" {
		return apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
	}
	return nil
}

func apiHTTPStatus(err error) int {
	return apierrors.HTTPStatus(err)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
