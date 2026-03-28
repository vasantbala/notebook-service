package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/util"
)

// createNotebookRequest is the JSON body for POST /notebooks/.
type createNotebookRequest struct {
	Title       string `json:"title"       example:"My Research"`
	Description string `json:"description" example:"Notes on ML papers"`
}

// updateNotebookRequest is the JSON body for PATCH /notebooks/{notebookID}.
type updateNotebookRequest struct {
	Title       string `json:"title"       example:"Updated Title"`
	Description string `json:"description" example:"Updated description"`
}

// ListNotebooks godoc
//
// @Summary      List notebooks
// @Description  Returns all notebooks belonging to the authenticated user.
// @Tags         notebooks
// @Produce      json
// @Success      200  {array}   model.Notebook
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/ [get]
func (h *Handlers) ListNotebooks(w http.ResponseWriter, r *http.Request) {

	userID, _ := r.Context().Value(UserIDKey).(string)

	notebooks, err := h.Notebooks.ListNotebooks(r.Context(), userID)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	util.WriteJSON(w, http.StatusOK, notebooks)
}

// CreateNotebook godoc
//
// @Summary      Create a notebook
// @Description  Creates a new notebook for the authenticated user.
// @Tags         notebooks
// @Consume      json
// @Produce      json
// @Param        body  body      createNotebookRequest  true  "Notebook fields"
// @Success      201   {object}  model.Notebook
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/ [post]
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

// DeleteNotebook godoc
//
// @Summary      Delete a notebook
// @Description  Permanently deletes a notebook and all its conversations and sources.
// @Tags         notebooks
// @Produce      json
// @Param        notebookID  path  string  true  "Notebook UUID"
// @Success      200  {object}  nil
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID} [delete]
func (h *Handlers) DeleteNotebook(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)

	notebookId := chi.URLParam(r, "notebookID")

	err := h.Notebooks.DeleteNotebook(r.Context(), notebookId, userID)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	util.WriteJSON(w, http.StatusOK, nil)
}

// GetNotebook godoc
//
// @Summary      Get a notebook
// @Description  Returns a single notebook by ID. The caller must own it.
// @Tags         notebooks
// @Produce      json
// @Param        notebookID  path      string  true  "Notebook UUID"
// @Success      200  {object}  model.Notebook
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID} [get]
func (h *Handlers) GetNotebook(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)

	notebookId := chi.URLParam(r, "notebookID")

	notebook, err := h.Notebooks.GetNotebook(r.Context(), notebookId, userID)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	util.WriteJSON(w, http.StatusOK, notebook)
}

// UpdateNotebook godoc
//
// @Summary      Update a notebook
// @Description  Updates the title and/or description of a notebook.
// @Tags         notebooks
// @Consume      json
// @Produce      json
// @Param        notebookID  path  string               true  "Notebook UUID"
// @Param        body        body  updateNotebookRequest  true  "Fields to update"
// @Success      200  {object}  model.Notebook
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID} [patch]
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
