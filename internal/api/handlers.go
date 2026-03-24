package api

import (
	"encoding/json"
	"net/http"

	"github.com/vasantbala/notebook-service/internal/service"
	"github.com/vasantbala/notebook-service/internal/util"
)

type Handlers struct {
	Notebooks service.NotebookService
}

func (h *Handlers) ListNotebooks(w http.ResponseWriter, r *http.Request) {

	//TODO: hardcode user id for now; Will substitute with JWT later
	userID := "user01"

	notebooks, err := h.Notebooks.ListNotebooks(r.Context(), userID)

	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	util.WriteJSON(w, http.StatusOK, notebooks)
}

func (h *Handlers) CreateNotebook(w http.ResponseWriter, r *http.Request) {
	//TODO: hardcode user id for now; Will substitute with JWT later
	userID := "user01"

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
