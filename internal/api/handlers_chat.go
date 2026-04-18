package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vasantbala/notebook-service/internal/llm"
	"github.com/vasantbala/notebook-service/internal/model"
	"github.com/vasantbala/notebook-service/internal/observability"
	"github.com/vasantbala/notebook-service/internal/service"
	"github.com/vasantbala/notebook-service/internal/util"
)

// chatStreamRequest is the JSON body for the SSE chat endpoint.
type chatStreamRequest struct {
	Query string `json:"query" example:"What are the key ideas in my notes?"`
	TopK  int    `json:"top_k" example:"5"`
}

// ChatStream godoc
//
// @Summary      Stream a chat response (SSE)
// @Description  Retrieves relevant source chunks (when RAG is enabled), streams an LLM completion as Server-Sent Events, and persists the exchange.
// @Tags         conversations
// @Consume      json
// @Produce      text/event-stream
// @Param        notebookID      path  string            true  "Notebook UUID"
// @Param        conversationID  path  string            true  "Conversation UUID"
// @Param        body            body  chatStreamRequest  true  "Query and retrieval options"
// @Success      200  {string}  string  "SSE stream of {\"token\":\"...\"} events, terminated by data: [DONE]"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /notebooks/{notebookID}/conversations/{conversationID}/chat [post]
func (h *Handlers) ChatStream(w http.ResponseWriter, r *http.Request) {

	notebookID := chi.URLParam(r, "notebookID")
	conversationID := chi.URLParam(r, "conversationID")
	userID, _ := r.Context().Value(UserIDKey).(string)
	bearerToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	var req struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"` // optional, defaults to 5
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.TopK == 0 {
		req.TopK = 5
	}

	// 1. Load conversation (provides RAG/reasoning toggles and model override).
	tracer := observability.Tracer()
	ctx, rootSpan := tracer.Start(r.Context(), "notebook-chat")
	defer rootSpan.End()
	observability.SpanSetUser(rootSpan, userID)
	observability.SpanSetInput(rootSpan, req.Query)

	conv, err := h.Conversations.GetConversation(ctx, conversationID, notebookID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// 2. Load conversation history (redis -> db fallback).
	history, err := h.Conversations.ListMessages(ctx, conversationID, userID)
	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// 3. Optionally retrieve RAG chunks — skip when RAG is disabled.
	var chunks []service.ChunkResult
	if conv.RAGEnabled {
		_, ragSpan := tracer.Start(ctx, "rag-retrieval")
		docIDs, err := h.Sources.ListRagDocIDs(ctx, notebookID, userID)
		if err != nil {
			util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			ragSpan.End()
			return
		}
		chunks, err = h.Retrieval.Search(ctx, req.Query, userID, bearerToken, docIDs, req.TopK)
		if err != nil {
			util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			ragSpan.End()
			return
		}
		observability.SpanSetRAGChunks(ragSpan, len(chunks))
		ragSpan.End()
	}

	// 4. Build prompt (empty chunks produce a plain prompt — no special casing needed).
	msgs := service.BuildChatMessages("", chunks, history, req.Query)

	// 5. Persist user message.
	if _, err := h.Conversations.AddMessage(ctx, conversationID, model.RoleUser, req.Query, 0, nil); err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// 6. Stream SSE response.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Resolve model name: per-conversation override > reasoning default > standard default.
	streamOpts := llm.StreamOptions{
		UseReasoning:    conv.UseReasoning,
		ReasoningEffort: h.Config.ReasoningEffort,
		ModelOverride:   conv.Model,
	}

	// Determine the effective model name for the span tag.
	effectiveModel := h.Config.StandardModel
	if conv.UseReasoning && h.Config.ReasoningModel != "" {
		effectiveModel = h.Config.ReasoningModel
	}
	if conv.Model != nil {
		effectiveModel = *conv.Model
	}
	_, llmSpan := tracer.Start(ctx, "llm-generation")
	observability.SpanSetModel(llmSpan, effectiveModel)

	// For reasoning models, emit a thinking event before tokens begin so the UI
	// can show an indicator during the silent pre-generation phase.
	if conv.UseReasoning {
		fmt.Fprint(w, "event: thinking\ndata: {}\n\n")
		flusher.Flush()
	}

	tokens := make(chan string, 32) // buffered so LLM can run slightly ahead of writer
	var assistantReply strings.Builder

	go h.LLM.StreamWithOptions(r.Context(), msgs, streamOpts, tokens)

	for {
		select {
		case token, open := <-tokens:
			if !open {
				observability.SpanSetOutput(llmSpan, assistantReply.String())
				llmSpan.End()
				// 7. Persist assistant message + citations in the background.
				go func() {
					ctx := context.Background()
					citations := make([]model.Citation, len(chunks))
					for i, c := range chunks {
						citations[i] = model.Citation{
							SourceID:   c.DocID,
							ChunkIndex: c.ChunkIndex,
							Score:      c.RerankerScore,
						}
					}
					_, _ = h.Conversations.AddMessage(
						ctx, conversationID,
						model.RoleAssistant, assistantReply.String(), 0, citations)
				}()
				fmt.Fprint(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
			assistantReply.WriteString(token)
			// Emit as a JSON SSE event so the client can parse it easily
			payload, _ := json.Marshal(map[string]string{"token": token})
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		case <-r.Context().Done():
			return // client disconnected
		}
	}
}
