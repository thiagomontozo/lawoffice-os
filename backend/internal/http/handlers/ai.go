package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/ai"
)

func (h *Handler) AskMatterAI(w http.ResponseWriter, r *http.Request) {
	matterID := chi.URLParam(r, "id")
	if !validID(matterID) {
		bad(w, r, "Invalid Matter ID")
		return
	}
	var input struct {
		Question    string   `json:"question"`
		DocumentIDs []string `json:"documentIds"`
	}
	if err := decode(w, r, &input); err != nil {
		bad(w, r, "Invalid AI query payload")
		return
	}
	currentUser := user(r)
	if h.AI == nil {
		h.fail(w, r, ai.ErrDisabled)
		return
	}
	answer, err := h.AI.Ask(r.Context(), ai.Query{
		FirmID: currentUser.FirmID, UserID: currentUser.ID, MatterID: matterID,
		Question: input.Question, DocumentIDs: input.DocumentIDs,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(input.Question)))
	ip, userAgent := auditContext(r)
	_ = h.Store.Audit(r.Context(), currentUser.FirmID, currentUser.ID, "ai.query", "matter", &matterID, map[string]any{
		"responseId": answer.ID, "questionHash": hex.EncodeToString(digest[:]),
		"questionCharacters": utf8.RuneCountInString(strings.TrimSpace(input.Question)),
		"citationCount":      len(answer.Citations), "model": answer.Model, "retrieval": answer.Retrieval,
	}, ip, userAgent)
	writeJSON(w, http.StatusOK, answer)
}

func (h *Handler) MatterAIFeedback(w http.ResponseWriter, r *http.Request) {
	matterID := chi.URLParam(r, "id")
	if !validID(matterID) {
		bad(w, r, "Invalid Matter ID")
		return
	}
	var input struct {
		ResponseID string  `json:"responseId"`
		Rating     string  `json:"rating"`
		Reason     *string `json:"reason"`
	}
	if err := decode(w, r, &input); err != nil {
		bad(w, r, "Invalid AI feedback payload")
		return
	}
	currentUser := user(r)
	if h.AI == nil {
		h.fail(w, r, ai.ErrDisabled)
		return
	}
	if err := h.AI.Feedback(r.Context(), currentUser.FirmID, currentUser.ID, matterID, input.ResponseID, input.Rating, input.Reason); err != nil {
		h.fail(w, r, err)
		return
	}
	ip, userAgent := auditContext(r)
	_ = h.Store.Audit(r.Context(), currentUser.FirmID, currentUser.ID, "ai.feedback", "matter", &matterID, map[string]any{
		"responseId": input.ResponseID, "rating": input.Rating,
	}, ip, userAgent)
	w.WriteHeader(http.StatusNoContent)
}
