package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vasantbala/notebook-service/internal/db"
	"github.com/vasantbala/notebook-service/internal/model"
	"github.com/vasantbala/notebook-service/internal/service"
)

// ---------------------------------------------------------------------------
// Stubs — satisfy the interfaces without Redis, Postgres, or external services
// ---------------------------------------------------------------------------

// noopJWTCache always reports a cache miss so AuthMiddleware falls through to
// JWT parsing. Used alongside the test router that skips real JWT validation.
type noopJWTCache struct{}

func (noopJWTCache) GetUserID(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}
func (noopJWTCache) SetUserID(_ context.Context, _ string, _ string, _ time.Duration) error {
	return nil
}

// noopRateLimiter always allows requests.
type noopRateLimiter struct{}

func (noopRateLimiter) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
	return true, nil
}

// inMemConversationService is a minimal in-memory ConversationService for tests.
type inMemConversationService struct {
	mu            sync.Mutex
	conversations map[string]*model.Conversation // key: conv ID
	messages      map[string][]model.Message     // key: conv ID
}

func newInMemConversationService() service.ConversationService {
	return &inMemConversationService{
		conversations: make(map[string]*model.Conversation),
		messages:      make(map[string][]model.Message),
	}
}

func (s *inMemConversationService) ListConversations(_ context.Context, notebookID, userID string) ([]model.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Conversation
	for _, c := range s.conversations {
		if c.NotebookID == notebookID && c.UserID == userID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (s *inMemConversationService) GetConversation(_ context.Context, id, notebookID, userID string) (*model.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conversations[id]
	if !ok || c.NotebookID != notebookID || c.UserID != userID {
		return nil, fmt.Errorf("get conversation: %w", model.ErrNotFound)
	}
	return c, nil
}

func (s *inMemConversationService) CreateConversation(_ context.Context, notebookID, userID, title string) (model.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	c := &model.Conversation{
		ID:         uuid.NewString(),
		NotebookID: notebookID,
		UserID:     userID,
		Title:      title,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.conversations[c.ID] = c
	return *c, nil
}

func (s *inMemConversationService) DeleteConversation(_ context.Context, id, notebookID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conversations[id]
	if !ok || c.NotebookID != notebookID || c.UserID != userID {
		return fmt.Errorf("delete conversation: %w", model.ErrNotFound)
	}
	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

func (s *inMemConversationService) ListMessages(_ context.Context, conversationID, _ string) ([]model.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messages[conversationID], nil
}

func (s *inMemConversationService) AddMessage(_ context.Context, conversationID string, role model.Role, content string, _ int, citations []model.Citation) (model.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := model.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		Citations:      citations,
		CreatedAt:      time.Now().UTC(),
	}
	s.messages[conversationID] = append(s.messages[conversationID], msg)
	return msg, nil
}

func (s *inMemConversationService) UpdateConversation(_ context.Context, id, notebookID, userID string, patch db.ConversationPatch) (*model.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conversations[id]
	if !ok || c.NotebookID != notebookID || c.UserID != userID {
		return nil, model.ErrNotFound
	}
	if patch.Title != nil {
		c.Title = *patch.Title
	}
	if patch.RAGEnabled != nil {
		c.RAGEnabled = *patch.RAGEnabled
	}
	if patch.UseReasoning != nil {
		c.UseReasoning = *patch.UseReasoning
	}
	if patch.Model != nil {
		c.Model = patch.Model
	}
	return c, nil
}

// inMemSourceService is a minimal stub — UploadSource always succeeds with status=pending.
type inMemSourceService struct {
	mu      sync.Mutex
	sources map[string]*model.Source
}

func newInMemSourceService() service.SourceService {
	return &inMemSourceService{sources: make(map[string]*model.Source)}
}

func (s *inMemSourceService) ListSources(_ context.Context, notebookID, userID string) ([]model.Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Source
	for _, src := range s.sources {
		if src.NotebookID == notebookID && src.UserID == userID {
			out = append(out, *src)
		}
	}
	return out, nil
}

func (s *inMemSourceService) GetSource(_ context.Context, id, notebookID, userID string) (*model.Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.sources[id]
	if !ok || src.NotebookID != notebookID || src.UserID != userID {
		return nil, fmt.Errorf("get source: %w", model.ErrNotFound)
	}
	return src, nil
}

func (s *inMemSourceService) UploadSource(_ context.Context, notebookID, userID, filename, storageKey, mimeType, _ string, _ io.Reader) (model.Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	src := &model.Source{
		ID:         uuid.NewString(),
		NotebookID: notebookID,
		UserID:     userID,
		Filename:   filename,
		StorageKey: storageKey,
		MimeType:   mimeType,
		Status:     model.SourceStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.sources[src.ID] = src
	return *src, nil
}

func (s *inMemSourceService) DeleteSource(_ context.Context, id, notebookID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.sources[id]
	if !ok || src.NotebookID != notebookID || src.UserID != userID {
		return fmt.Errorf("delete source: %w", model.ErrNotFound)
	}
	delete(s.sources, id)
	return nil
}

func (s *inMemSourceService) ListRagDocIDs(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Test router — bypasses JWT validation; sets userID from a custom header
// ---------------------------------------------------------------------------

// testRouter builds a chi router identical to production but replaces
// AuthMiddleware with a simple "X-Test-User" header reader so tests don't
// need a real JWKS endpoint or valid tokens.
func testRouter(h *Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(LoggerMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"notebook-service"}`))
	})

	r.Route("/notebooks", func(r chi.Router) {
		// Inject userID from the X-Test-User header instead of a JWT.
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID := r.Header.Get("X-Test-User")
				if userID == "" {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		r.Use(RateLimitMiddleware(noopRateLimiter{}, 1000, time.Minute))

		r.Get("/", h.ListNotebooks)
		r.Post("/", h.CreateNotebook)
		r.Route("/{notebookID}", func(r chi.Router) {
			r.Get("/", h.GetNotebook)
			r.Patch("/", h.UpdateNotebook)
			r.Delete("/", h.DeleteNotebook)
			r.Route("/conversations", func(r chi.Router) {
				r.Get("/", h.ListConversations)
				r.Post("/", h.CreateConversation)
				r.Route("/{conversationID}", func(r chi.Router) {
					r.Get("/", h.GetConversation)
					r.Delete("/", h.DeleteConversation)
					r.Get("/messages", h.ListMessages)
				})
			})
			r.Route("/sources", func(r chi.Router) {
				r.Get("/", h.ListSources)
				r.Post("/", h.UploadSource)
				r.Route("/{sourceID}", func(r chi.Router) {
					r.Get("/", h.GetSource)
					r.Delete("/", h.DeleteSource)
				})
			})
		})
	})
	return r
}

// testHandlers returns a Handlers wired with in-memory implementations.
func testHandlers() *Handlers {
	return &Handlers{
		Notebooks:     service.NewInMemNotebookService(),
		Conversations: newInMemConversationService(),
		Sources:       newInMemSourceService(),
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func do(t *testing.T, srv *httptest.Server, method, path, userID string, body any) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, srv.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User", userID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func mustDecode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("want status %d, got %d — body: %s", want, resp.StatusCode, body)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(testRouter(testHandlers()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, resp, http.StatusOK)
}

func TestNotebookCRUD(t *testing.T) {
	srv := httptest.NewServer(testRouter(testHandlers()))
	defer srv.Close()

	userID := "user-alice"

	// 1. List — empty at start
	resp := do(t, srv, http.MethodGet, "/notebooks/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	nbs := mustDecode[[]model.Notebook](t, resp)
	if len(nbs) != 0 {
		t.Fatalf("expected empty list, got %d notebooks", len(nbs))
	}

	// 2. Create
	resp = do(t, srv, http.MethodPost, "/notebooks/", userID, map[string]string{
		"title":       "My Research",
		"description": "Notes on Go concurrency",
	})
	assertStatus(t, resp, http.StatusCreated)
	created := mustDecode[model.Notebook](t, resp)
	if created.ID == "" {
		t.Fatal("expected notebook ID, got empty string")
	}
	if created.Title != "My Research" {
		t.Fatalf("expected title 'My Research', got %q", created.Title)
	}

	// 3. List — now has one entry
	resp = do(t, srv, http.MethodGet, "/notebooks/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	nbs = mustDecode[[]model.Notebook](t, resp)
	if len(nbs) != 1 {
		t.Fatalf("expected 1 notebook, got %d", len(nbs))
	}

	// 4. Get by ID
	resp = do(t, srv, http.MethodGet, "/notebooks/"+created.ID, userID, nil)
	assertStatus(t, resp, http.StatusOK)
	fetched := mustDecode[model.Notebook](t, resp)
	if fetched.ID != created.ID {
		t.Fatalf("ID mismatch: want %s got %s", created.ID, fetched.ID)
	}

	// 5. Update
	resp = do(t, srv, http.MethodPatch, "/notebooks/"+created.ID, userID, map[string]string{
		"title":       "My Research (Updated)",
		"description": "Notes on Go concurrency — updated",
	})
	assertStatus(t, resp, http.StatusOK)
	updated := mustDecode[model.Notebook](t, resp)
	if updated.Title != "My Research (Updated)" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}

	// 6. Other user cannot see this notebook
	resp = do(t, srv, http.MethodGet, "/notebooks/", "user-bob", nil)
	assertStatus(t, resp, http.StatusOK)
	bobNbs := mustDecode[[]model.Notebook](t, resp)
	if len(bobNbs) != 0 {
		t.Fatalf("bob should see 0 notebooks, got %d", len(bobNbs))
	}

	// 7. Delete
	resp = do(t, srv, http.MethodDelete, "/notebooks/"+created.ID, userID, nil)
	// handler currently returns 200; assert 2xx
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("delete returned non-2xx: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 8. List after delete — empty again
	resp = do(t, srv, http.MethodGet, "/notebooks/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	nbs = mustDecode[[]model.Notebook](t, resp)
	if len(nbs) != 0 {
		t.Fatalf("expected 0 notebooks after delete, got %d", len(nbs))
	}
}

func TestConversationWorkflow(t *testing.T) {
	srv := httptest.NewServer(testRouter(testHandlers()))
	defer srv.Close()

	userID := "user-alice"

	// Create a notebook first.
	resp := do(t, srv, http.MethodPost, "/notebooks/", userID, map[string]string{
		"title": "Chat Test Notebook",
	})
	assertStatus(t, resp, http.StatusCreated)
	nb := mustDecode[model.Notebook](t, resp)

	base := "/notebooks/" + nb.ID + "/conversations"

	// 1. List conversations — empty
	resp = do(t, srv, http.MethodGet, base+"/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	convs := mustDecode[[]model.Conversation](t, resp)
	if len(convs) != 0 {
		t.Fatalf("expected 0 conversations, got %d", len(convs))
	}

	// 2. Create conversation
	resp = do(t, srv, http.MethodPost, base+"/", userID, map[string]string{"title": "My First Chat"})
	assertStatus(t, resp, http.StatusCreated)
	conv := mustDecode[model.Conversation](t, resp)
	if conv.ID == "" {
		t.Fatal("expected conversation ID")
	}
	if conv.NotebookID != nb.ID {
		t.Fatalf("notebook ID mismatch: want %s got %s", nb.ID, conv.NotebookID)
	}

	// 3. List — now one conversation
	resp = do(t, srv, http.MethodGet, base+"/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	convs = mustDecode[[]model.Conversation](t, resp)
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}

	// 4. Get conversation by ID
	resp = do(t, srv, http.MethodGet, base+"/"+conv.ID, userID, nil)
	assertStatus(t, resp, http.StatusOK)
	fetched := mustDecode[model.Conversation](t, resp)
	if fetched.ID != conv.ID {
		t.Fatalf("conversation ID mismatch")
	}

	// 5. Get messages — empty
	resp = do(t, srv, http.MethodGet, base+"/"+conv.ID+"/messages", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	msgs := mustDecode[[]model.Message](t, resp)
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}

	// 6. Delete conversation
	resp = do(t, srv, http.MethodDelete, base+"/"+conv.ID, userID, nil)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// 7. List — empty again
	resp = do(t, srv, http.MethodGet, base+"/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	convs = mustDecode[[]model.Conversation](t, resp)
	if len(convs) != 0 {
		t.Fatalf("expected 0 conversations after delete, got %d", len(convs))
	}
}

func TestSourceWorkflow(t *testing.T) {
	srv := httptest.NewServer(testRouter(testHandlers()))
	defer srv.Close()

	userID := "user-alice"

	// Create a notebook.
	resp := do(t, srv, http.MethodPost, "/notebooks/", userID, map[string]string{
		"title": "Source Test Notebook",
	})
	assertStatus(t, resp, http.StatusCreated)
	nb := mustDecode[model.Notebook](t, resp)

	base := "/notebooks/" + nb.ID + "/sources"

	// 1. List sources — empty
	resp = do(t, srv, http.MethodGet, base+"/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	sources := mustDecode[[]model.Source](t, resp)
	if len(sources) != 0 {
		t.Fatalf("expected 0 sources, got %d", len(sources))
	}

	// 2. Upload source (multipart)
	body := &bytes.Buffer{}
	fileContent := strings.NewReader("Hello, this is a test document.\n")
	multipartBody := &bytes.Buffer{}
	boundary := "testboundary"
	fmt.Fprintf(multipartBody, "--%s\r\n", boundary)
	fmt.Fprintf(multipartBody, "Content-Disposition: form-data; name=\"file\"; filename=\"test.txt\"\r\n")
	fmt.Fprintf(multipartBody, "Content-Type: text/plain\r\n\r\n")
	io.Copy(multipartBody, fileContent)
	fmt.Fprintf(multipartBody, "\r\n--%s--\r\n", boundary)
	body = multipartBody

	req, _ := http.NewRequest(http.MethodPost, srv.URL+base+"/", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("X-Test-User", userID)
	req.Header.Set("Authorization", "Bearer test-token")

	uploadResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	assertStatus(t, uploadResp, http.StatusAccepted)
	src := mustDecode[model.Source](t, uploadResp)
	if src.ID == "" {
		t.Fatal("expected source ID")
	}
	if src.Status != model.SourceStatusPending {
		t.Fatalf("expected status=pending, got %q", src.Status)
	}

	// 3. List sources — now one
	resp = do(t, srv, http.MethodGet, base+"/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	sources = mustDecode[[]model.Source](t, resp)
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	// 4. Get source by ID
	resp = do(t, srv, http.MethodGet, base+"/"+src.ID, userID, nil)
	assertStatus(t, resp, http.StatusOK)
	fetched := mustDecode[model.Source](t, resp)
	if fetched.ID != src.ID {
		t.Fatalf("source ID mismatch")
	}

	// 5. Delete source
	resp = do(t, srv, http.MethodDelete, base+"/"+src.ID, userID, nil)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// 6. List — empty again
	resp = do(t, srv, http.MethodGet, base+"/", userID, nil)
	assertStatus(t, resp, http.StatusOK)
	sources = mustDecode[[]model.Source](t, resp)
	if len(sources) != 0 {
		t.Fatalf("expected 0 sources after delete, got %d", len(sources))
	}
}

func TestUnauthorized(t *testing.T) {
	srv := httptest.NewServer(testRouter(testHandlers()))
	defer srv.Close()

	// Request with no X-Test-User header should be rejected.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/notebooks/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusUnauthorized)
}
