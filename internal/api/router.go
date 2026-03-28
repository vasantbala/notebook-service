package api

import (
	"net/http"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/util"
)

func NewRouter(h *Handlers, jwks keyfunc.Keyfunc, jwtCache jwtCache) http.Handler {
	r := chi.NewRouter()

	//Middleware
	r.Use(LoggerMiddleware)
	//Adding here add auth for all endpoints
	//r.Use(AuthMiddleware(jwks))

	//Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		util.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "notebook-service",
		})
	})

	r.Route("/notebooks", func(r chi.Router) {
		r.Use(AuthMiddleware(jwks, jwtCache))
		r.Get("/", h.ListNotebooks)
		r.Post("/", h.CreateNotebook)
		r.Route("/{notebookID}", func(r chi.Router) {
			r.Delete("/", h.DeleteNotebook)
			r.Get("/", h.GetNotebook)
			//TODO: r.Patch("/", h.UpdateNotebook)
			// r.Route("/conversations", func(r chi.Router){
			// 	r.Get("/", h.ListConversations)
			// 	r.Post("/", h.CreateConversation)
			// 	r.Route("/{conversationID}", func(r chi.Router) {
			// 		r.Get("/", h.GetConversation)
			// 		r.Delete("/", h.DeleteConversation)
			// 		r.Get("/messages", h.ListMessages)
			// 		r.Get("/chat", h.ChatStream) // SSE — Phase 13
			// 	})
			// })
			// r.Route("/sources", func(r chi.Router) {
			// 	r.Get("/", h.ListSources)
			// 	r.Post("/", h.UploadSource)
			// 	r.Route("/{sourceID}", func(r chi.Router) {
			// 		r.Get("/", h.GetSource)
			// 		r.Delete("/", h.DeleteSource)
			// 	})
			// })
		})

	})

	return r
}
