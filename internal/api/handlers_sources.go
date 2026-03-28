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

// ListSources godoc
//
// @Summary      List sources
// @Description  Returns all source documents in a notebook.
// @Tags         sources
// @Produce      json
// @Param        notebookID  path  string  true  "Notebook UUID"
// @Success      200  {array}   model.Source
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/sources/ [get]
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

// GetSource godoc
//
// @Summary      Get a source
// @Description  Returns a single source document including its ingestion status.
// @Tags         sources
// @Produce      json
// @Param        notebookID  path  string  true  "Notebook UUID"
// @Param        sourceID    path  string  true  "Source UUID"
// @Success      200  {object}  model.Source
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/sources/{sourceID} [get]
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

// UploadSource godoc
//
// @Summary      Upload a source document
// @Description  Accepts a multipart upload, creates a source record, and enqueues async ingestion into rag-anything.
// @Tags         sources
// @Consume      mpfd
// @Produce      json
// @Param        notebookID   path      string  true   "Notebook UUID"
// @Param        file         formData  file    true   "Document file (PDF, DOCX, etc.)"
// @Param        storage_key  formData  string  false  "Pre-uploaded object storage key (defaults to notebookID/filename)"
// @Success      202  {object}  model.Source  "Accepted — ingestion is asynchronous"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      413  {object}  map[string]string  "File too large (max 50 MB)"
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/sources/ [post]
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

// DeleteSource godoc
//
// @Summary      Delete a source
// @Description  Permanently deletes a source document and removes it from rag-anything.
// @Tags         sources
// @Produce      json
// @Param        notebookID  path  string  true  "Notebook UUID"
// @Param        sourceID    path  string  true  "Source UUID"
// @Success      204  "No Content"
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/sources/{sourceID} [delete]
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
