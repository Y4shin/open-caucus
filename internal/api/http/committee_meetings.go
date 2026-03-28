package apihttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/session"
)

// NewCommitteeMeetingCreateHandler creates a new meeting in a committee.
// Requires chairperson role.
// POST /committee/{slug}/meetings
func NewCommitteeMeetingCreateHandler(repo repository.Repository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		sd, ok := session.GetSession(r.Context())
		if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
			writeError(w, apierrors.New(apierrors.KindUnauthenticated, "account session required"))
			return
		}

		membership, err := repo.GetUserMembershipByAccountIDAndSlug(r.Context(), *sd.AccountID, slug)
		if err != nil {
			writeError(w, apierrors.New(apierrors.KindPermissionDenied, "not a member of this committee"))
			return
		}
		account, _ := repo.GetAccountByID(r.Context(), *sd.AccountID)
		if membership.Role != "chairperson" && (account == nil || !account.IsAdmin) {
			writeError(w, apierrors.New(apierrors.KindPermissionDenied, "chairperson role required"))
			return
		}

		committee, err := repo.GetCommitteeBySlug(r.Context(), slug)
		if err != nil {
			writeError(w, apierrors.New(apierrors.KindNotFound, "committee not found"))
			return
		}

		var name, description string
		if r.Header.Get("Content-Type") == "application/json" {
			var body struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, apierrors.New(apierrors.KindInvalidArgument, "invalid request body"))
				return
			}
			name = body.Name
			description = body.Description
		} else {
			if err := r.ParseForm(); err != nil {
				writeError(w, apierrors.New(apierrors.KindInvalidArgument, "invalid form data"))
				return
			}
			name = r.FormValue("name")
			description = r.FormValue("description")
		}

		if name == "" {
			writeError(w, apierrors.New(apierrors.KindInvalidArgument, "meeting name is required"))
			return
		}

		if err := repo.CreateMeeting(r.Context(), committee.ID, name, description, "", false); err != nil {
			writeError(w, apierrors.Wrap(apierrors.KindInternal, "failed to create meeting", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	})
}

// NewCommitteeMeetingDeleteHandler deletes a meeting from a committee.
// Requires chairperson role.
// DELETE /committee/{slug}/meetings/{meetingId}
func NewCommitteeMeetingDeleteHandler(repo repository.Repository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		meetingIDStr := r.PathValue("meetingId")

		sd, ok := session.GetSession(r.Context())
		if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
			writeError(w, apierrors.New(apierrors.KindUnauthenticated, "account session required"))
			return
		}

		membership, err := repo.GetUserMembershipByAccountIDAndSlug(r.Context(), *sd.AccountID, slug)
		if err != nil {
			writeError(w, apierrors.New(apierrors.KindPermissionDenied, "not a member of this committee"))
			return
		}
		account, _ := repo.GetAccountByID(r.Context(), *sd.AccountID)
		if membership.Role != "chairperson" && (account == nil || !account.IsAdmin) {
			writeError(w, apierrors.New(apierrors.KindPermissionDenied, "chairperson role required"))
			return
		}

		meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
		if err != nil {
			writeError(w, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id"))
			return
		}

		if err := repo.DeleteMeeting(r.Context(), meetingID); err != nil {
			writeError(w, apierrors.Wrap(apierrors.KindInternal, "failed to delete meeting", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	})
}

// NewCommitteeMeetingActivateHandler sets or clears the active meeting for a committee.
// Requires chairperson role.
// POST /committee/{slug}/meetings/{meetingId}/active  (toggle: activate if inactive, deactivate if active)
func NewCommitteeMeetingActivateHandler(repo repository.Repository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		meetingIDStr := r.PathValue("meetingId")

		sd, ok := session.GetSession(r.Context())
		if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
			writeError(w, apierrors.New(apierrors.KindUnauthenticated, "account session required"))
			return
		}

		membership, err := repo.GetUserMembershipByAccountIDAndSlug(r.Context(), *sd.AccountID, slug)
		if err != nil {
			writeError(w, apierrors.New(apierrors.KindPermissionDenied, "not a member of this committee"))
			return
		}
		account, _ := repo.GetAccountByID(r.Context(), *sd.AccountID)
		if membership.Role != "chairperson" && (account == nil || !account.IsAdmin) {
			writeError(w, apierrors.New(apierrors.KindPermissionDenied, "chairperson role required"))
			return
		}

		meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
		if err != nil {
			writeError(w, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id"))
			return
		}

		// Toggle: if this meeting is currently active, deactivate; otherwise activate.
		committee, err := repo.GetCommitteeBySlug(r.Context(), slug)
		if err != nil {
			writeError(w, apierrors.New(apierrors.KindNotFound, "committee not found"))
			return
		}

		var newActiveMeetingID *int64
		if committee.CurrentMeetingID == nil || *committee.CurrentMeetingID != meetingID {
			newActiveMeetingID = &meetingID
		}
		// else: already active → deactivate (nil)

		if err := repo.SetActiveMeeting(r.Context(), slug, newActiveMeetingID); err != nil {
			writeError(w, apierrors.Wrap(apierrors.KindInternal, "failed to set active meeting", err))
			return
		}

		isActive := newActiveMeetingID != nil
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"active": isActive})
	})
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	var apiErr *apierrors.Error
	if !errors.As(err, &apiErr) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(apierrors.HTTPStatus(err))
	json.NewEncoder(w).Encode(map[string]string{"error": apiErr.Message})
}
