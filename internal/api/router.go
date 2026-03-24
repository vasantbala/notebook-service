package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/util"
)

func NewRouter(h *Handlers) http.Handler {
	r := chi.NewRouter()

	//TODO: Add middleware

	//Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		util.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "notebook-service",
		})
	})

	r.Route("/notebooks", func(r chi.Router) {
		r.Get("/", h.ListNotebooks)
		r.Post("/", h.CreateNotebook)
		r.Delete("/{notebookID}", h.DeleteNotebook)
		r.Get("/{notebookID}", h.GetNotebook)
	})

	return r
}
