package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/util"
)

// maxUploadSize is the maximum file size accepted for source uploads (50 MB).
const maxUploadSize = 50 << 20

func (h *Handlers) ListSources(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	sources, err := h.Sources.ListSources(r.Context(), notebookID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, sources)
}

func (h *Handlers) GetSource(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	sourceID := chi.URLParam(r, "sourceID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	src, err := h.Sources.GetSource(r.Context(), sourceID, notebookID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, src)
}

// UploadSource handles multipart/form-data uploads.
//
// Expected form fields:
//
//	file        — the file binary (required)
//	storage_key — object storage key the client pre-uploaded to (optional; defaults to a generated key)
//
// The handler forwards the raw bearer token to the service so it can be
// passed on to rag-anything's /ingest endpoint.
func (h *Handlers) UploadSource(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	userID, _ := r.Context().Value(UserIDKey).(string)
	bearerToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	// Limit total body to prevent memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		util.WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file too large (max 50 MB)"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer file.Close()

	filename := header.Filename
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// storage_key is the object key the client already wrote to S3/local storage.
	// If not provided, default to notebookID/filename so the service can look it up.
	storageKey := r.FormValue("storage_key")
	if storageKey == "" {
		storageKey = notebookID + "/" + filename
	}

	// io.Reader is consumed once; the service reads it to send to rag-anything.
	src, err := h.Sources.UploadSource(r.Context(), notebookID, userID, filename, storageKey, mimeType, bearerToken, io.Reader(file))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	util.WriteJSON(w, http.StatusAccepted, src) // 202 — ingest is async, status=pending
}

func (h *Handlers) DeleteSource(w http.ResponseWriter, r *http.Request) {
	notebookID := chi.URLParam(r, "notebookID")
	sourceID := chi.URLParam(r, "sourceID")
	userID, _ := r.Context().Value(UserIDKey).(string)

	if err := h.Sources.DeleteSource(r.Context(), sourceID, notebookID, userID); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
