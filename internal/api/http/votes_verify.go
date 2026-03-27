package apihttp

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Y4shin/conference-tool/internal/repository"
)

type verifyOpenVoteRequest struct {
	VoteID       int64  `json:"vote_id"`
	ReceiptToken string `json:"receipt_token"`
	AttendeeID   *int64 `json:"attendee_id,omitempty"`
}

type verifySecretVoteRequest struct {
	VoteID       int64  `json:"vote_id"`
	ReceiptToken string `json:"receipt_token"`
}

func NewVerifyOpenVoteReceiptHandler(repo repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verifyOpenVoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
		if req.VoteID <= 0 || strings.TrimSpace(req.ReceiptToken) == "" {
			writeJSONError(w, http.StatusBadRequest, "vote_id and receipt_token are required")
			return
		}

		verification, err := repo.VerifyOpenBallotByReceipt(r.Context(), req.VoteID, strings.TrimSpace(req.ReceiptToken))
		if err != nil {
			writeJSONError(w, mapVoteVerifyHTTPStatus(err), err.Error())
			return
		}
		if req.AttendeeID != nil && verification.AttendeeID != *req.AttendeeID {
			writeJSONError(w, http.StatusNotFound, "ballot not found for attendee")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"vote_id":           verification.VoteDefinitionID,
			"vote_name":         verification.VoteName,
			"attendee_id":       verification.AttendeeID,
			"attendee_number":   verification.AttendeeNumber,
			"receipt_token":     verification.ReceiptToken,
			"choice_labels":     verification.ChoiceLabels,
			"choice_option_ids": verification.ChoiceOptionIDs,
		})
	}
}

func NewVerifySecretVoteReceiptHandler(repo repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verifySecretVoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
		if req.VoteID <= 0 || strings.TrimSpace(req.ReceiptToken) == "" {
			writeJSONError(w, http.StatusBadRequest, "vote_id and receipt_token are required")
			return
		}

		verification, err := repo.VerifySecretBallotByReceipt(r.Context(), req.VoteID, strings.TrimSpace(req.ReceiptToken))
		if err != nil {
			writeJSONError(w, mapVoteVerifyHTTPStatus(err), err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"vote_id":                  verification.VoteDefinitionID,
			"vote_name":                verification.VoteName,
			"receipt_token":            verification.ReceiptToken,
			"encrypted_commitment_b64": base64.StdEncoding.EncodeToString(verification.EncryptedCommitment),
			"commitment_cipher":        verification.CommitmentCipher,
			"commitment_version":       verification.CommitmentVersion,
		})
	}
}

func mapVoteVerifyHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not found"):
		return http.StatusNotFound
	case strings.Contains(message, "counting"):
		return http.StatusConflict
	case strings.Contains(message, "invalid"):
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}
