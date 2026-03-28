package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/model"
	"github.com/vasantbala/notebook-service/internal/util"
)

func (h *Handlers) ListConversations(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	convs, err := h.Conversations.ListConversations(r.Context(), notebookID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, convs)
}

func (h *Handlers) GetConversation(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	convID := chi.URLParam(r, "conversationID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	conv, err := h.Conversations.GetConversation(r.Context(), convID, notebookID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, conv)
}

func (h *Handlers) CreateConversation(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	conv, err := h.Conversations.CreateConversation(r.Context(), notebookID, userID, req.Title)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusCreated, conv)
}

func (h *Handlers) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	convID := chi.URLParam(r, "conversationID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	if err := h.Conversations.DeleteConversation(r.Context(), convID, notebookID, userID); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ListMessages(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conversationID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	msgs, err := h.Conversations.ListMessages(r.Context(), convID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, msgs)
}

// handleServiceError maps sentinel errors to HTTP status codes.
// Defined here so all handler files in the api package can use it.
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, model.ErrForbidden):
		util.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, model.ErrConflict):
		util.WriteJSON(w, http.StatusConflict, map[string]string{"error": "conflict"})
	default:
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}
