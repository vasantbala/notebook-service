package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/util"
)

func (h *Handlers) ListNotebooks(w http.ResponseWriter, r *http.Request) {

	userID, _ := r.Context().Value(UserIDKey).(string)

	notebooks, err := h.Notebooks.ListNotebooks(r.Context(), userID)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	util.WriteJSON(w, http.StatusOK, notebooks)
}

func (h *Handlers) CreateNotebook(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	nb, err := h.Notebooks.CreateNotebook(r.Context(), userID, req.Title, req.Description)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	util.WriteJSON(w, http.StatusCreated, nb)
}

func (h *Handlers) DeleteNotebook(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)

	notebookId := chi.URLParam(r, "notebookID")

	err := h.Notebooks.DeleteNotebook(r.Context(), notebookId, userID)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	util.WriteJSON(w, http.StatusOK, nil)
}

func (h *Handlers) GetNotebook(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)

	notebookId := chi.URLParam(r, "notebookID")

	notebook, err := h.Notebooks.GetNotebook(r.Context(), notebookId, userID)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	util.WriteJSON(w, http.StatusOK, notebook)
}

func (h *Handlers) UpdateNotebook(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)
	notebookID := chi.URLParam(r, "notebookID")

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	nb, err := h.Notebooks.UpdateNotebook(r.Context(), notebookID, userID, req.Title, req.Description)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, nb)
}
