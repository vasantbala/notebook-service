package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/db"
	"github.com/vasantbala/notebook-service/internal/model"
	"github.com/vasantbala/notebook-service/internal/util"
)

// createConversationRequest is the JSON body for POST /notebooks/{notebookID}/conversations/.
type createConversationRequest struct {
	Title string `json:"title" example:"Discussion on chapter 3"`
}

// ListConversations godoc
//
// @Summary      List conversations
// @Description  Returns all conversations in a notebook.
// @Tags         conversations
// @Produce      json
// @Param        notebookID  path  string  true  "Notebook UUID"
// @Success      200  {array}   model.Conversation
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/conversations/ [get]
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

// GetConversation godoc
//
// @Summary      Get a conversation
// @Description  Returns a single conversation by ID.
// @Tags         conversations
// @Produce      json
// @Param        notebookID      path  string  true  "Notebook UUID"
// @Param        conversationID  path  string  true  "Conversation UUID"
// @Success      200  {object}  model.Conversation
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/conversations/{conversationID} [get]
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

// CreateConversation godoc
//
// @Summary      Create a conversation
// @Description  Creates a new conversation inside a notebook.
// @Tags         conversations
// @Consume      json
// @Produce      json
// @Param        notebookID  path  string                     true  "Notebook UUID"
// @Param        body        body  createConversationRequest  true  "Conversation title"
// @Success      201  {object}  model.Conversation
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/conversations/ [post]
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

// DeleteConversation godoc
//
// @Summary      Delete a conversation
// @Description  Permanently deletes a conversation and all its messages.
// @Tags         conversations
// @Produce      json
// @Param        notebookID      path  string  true  "Notebook UUID"
// @Param        conversationID  path  string  true  "Conversation UUID"
// @Success      204  "No Content"
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/conversations/{conversationID} [delete]
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

// ListMessages godoc
//
// @Summary      List messages
// @Description  Returns all messages in a conversation. Checks Redis cache first, falls back to DB.
// @Tags         conversations
// @Produce      json
// @Param        notebookID      path  string  true  "Notebook UUID"
// @Param        conversationID  path  string  true  "Conversation UUID"
// @Success      200  {array}   model.Message
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/conversations/{conversationID}/messages [get]
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

// updateConversationRequest is the JSON body for PATCH /conversations/{conversationID}.
type updateConversationRequest struct {
	Title        *string `json:"title"`
	RAGEnabled   *bool   `json:"rag_enabled"`
	UseReasoning *bool   `json:"use_reasoning"`
	Model        *string `json:"model"` // set to "" to clear the per-conversation override
}

// UpdateConversation godoc
//
// @Summary      Update a conversation
// @Description  Partially updates a conversation's title, RAG toggle, reasoning toggle, or model override.
// @Tags         conversations
// @Consume      json
// @Produce      json
// @Param        notebookID      path  string                     true  "Notebook UUID"
// @Param        conversationID  path  string                     true  "Conversation UUID"
// @Param        body            body  updateConversationRequest  true  "Fields to update"
// @Success      200  {object}  model.Conversation
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/conversations/{conversationID} [patch]
func (h *Handlers) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	convID := chi.URLParam(r, "conversationID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	var req updateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	patch := db.ConversationPatch{
		Title:        req.Title,
		RAGEnabled:   req.RAGEnabled,
		UseReasoning: req.UseReasoning,
		Model:        req.Model,
	}

	conv, err := h.Conversations.UpdateConversation(r.Context(), convID, notebookID, userID, patch)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, conv)
}
