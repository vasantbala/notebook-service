# notebook-service — Phase 2 Implementation Plan

> Covers three new features layered on top of the Phase 1 foundation:
> 1. **RAG toggle** — per-conversation on/off switch stored in the DB
> 2. **Reasoning model toggle** — per-conversation model routing + streaming hints
> 3. **MCP gateway integration** — tool discovery, tool-call forwarding, streamed tool events
>
> **Prerequisites**: Phase 1 complete (JWT auth, PostgreSQL repos, SSE chat, RAG pipeline wired).

---

## Feature 1 — RAG Toggle (per conversation)

### 1a. DB Migration

```sql
-- migrations/003_conversation_settings.up.sql
ALTER TABLE conversations ADD COLUMN rag_enabled   BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE conversations ADD COLUMN use_reasoning BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE conversations ADD COLUMN model         TEXT;     -- NULL = use service default
```

```sql
-- migrations/003_conversation_settings.down.sql
ALTER TABLE conversations DROP COLUMN rag_enabled;
ALTER TABLE conversations DROP COLUMN use_reasoning;
ALTER TABLE conversations DROP COLUMN model;
```

### 1b. Update the Model

```go
// internal/model/conversation.go
type Conversation struct {
    ID           string    `json:"id"            db:"id"`
    NotebookID   string    `json:"notebook_id"   db:"notebook_id"`
    UserID       string    `json:"user_id"       db:"user_id"`
    Title        string    `json:"title"         db:"title"`
    RAGEnabled   bool      `json:"rag_enabled"   db:"rag_enabled"`
    UseReasoning bool      `json:"use_reasoning" db:"use_reasoning"`
    Model        *string   `json:"model"         db:"model"` // nil = service default
    CreatedAt    time.Time `json:"created_at"    db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}
```

### 1c. Expose Toggle via PATCH

```
PATCH /notebooks/{notebookID}/conversations/{conversationID}
Body: { "rag_enabled": false }
      { "use_reasoning": true }
```

The existing PATCH handler (currently updates `title`) should be extended to accept and persist both toggle fields.

```go
// internal/api/handlers_conversations.go
type updateConversationRequest struct {
    Title        *string `json:"title"`
    RAGEnabled   *bool   `json:"rag_enabled"`
    UseReasoning *bool   `json:"use_reasoning"`
    Model        *string `json:"model"` // set to "" to clear override and revert to service default
}
```

Use pointer fields so absent keys are treated as no-op (partial update semantics).

### 1e. Expose Available Models

Add a models endpoint so the UI can populate the model picker without hardcoding model names:

```
GET /models
→ [ { "id": "gpt-4o", "label": "GPT-4o", "reasoning": false },
    { "id": "o3",     "label": "o3 (Reasoning)", "reasoning": true } ]
```

The response is built from `LLM_MODEL` and `LLM_REASONING_MODEL` config values — no DB involved. This is a static handler with no auth scoping needed (list of model names is not sensitive).

In the chat handler, resolve the effective model:

```go
modelName := cfg.LLMModel
if conv.UseReasoning {
    modelName = cfg.LLMReasoningModel
}
if conv.Model != nil && *conv.Model != "" {
    modelName = *conv.Model // per-conversation override wins
}
```

### 1d. Honour the Flag in the Chat Handler

```go
// internal/api/handlers_chat.go
conv, _ := h.Conversations.GetConversation(ctx, convID, userID)

var chunks []model.Chunk
if conv.RAGEnabled {
    chunks, _ = h.Retrieval.Retrieve(ctx, conv.NotebookID, userMessage)
}
// build prompt — chunks slice is empty when RAG is off; prompt builder already handles this
msgs := h.Prompts.Build(history, userMessage, chunks)
```

No changes needed in `service/retrieval.go` or `service/prompt.go` — the empty `chunks` slice already produces a plain prompt.

---

## Feature 2 — Reasoning Model Toggle

### 2a. Extend Config

```go
// internal/config/config.go
type Config struct {
    // ... existing fields ...
    LLMModel          string  // standard model, e.g. "gpt-4o"
    LLMReasoningModel string  // reasoning model, e.g. "o3", "o1"
    LLMBaseURL        string
    LLMAPIKey         string
}

func Load() Config {
    return Config{
        // ...
        LLMModel:          getEnv("LLM_MODEL", "gpt-4o"),
        LLMReasoningModel: getEnv("LLM_REASONING_MODEL", "o3"),
    }
}
```

### 2b. Extend the LLMClient Interface

```go
// internal/llm/llm.go
type LLMClient interface {
    Complete(ctx context.Context, msgs []model.Message) (string, error)
    Stream(ctx context.Context, msgs []model.Message, out chan<- string) error
    StreamWithOptions(ctx context.Context, msgs []model.Message, opts StreamOptions, out chan<- string) error
}

type StreamOptions struct {
    // Reasoning models require ReasoningEffort instead of Temperature.
    // Standard models ignore ReasoningEffort.
    UseReasoning     bool
    ReasoningEffort  string // "low" | "medium" | "high" — o3/o1 parameter
}
```

### 2c. Implement in openai.go

```go
// internal/llm/openai.go
func (c *openAIClient) StreamWithOptions(
    ctx context.Context,
    msgs []model.Message,
    opts StreamOptions,
    out chan<- string,
) error {
    defer close(out)

    var modelName string
    if opts.UseReasoning {
        modelName = c.reasoningModel
    } else {
        modelName = c.model
    }

    params := openai.ChatCompletionNewParams{
        Model:    openai.ChatModel(modelName),
        Messages: toOpenAIMessages(msgs),
    }
    if opts.UseReasoning && opts.ReasoningEffort != "" {
        // o3 / o1 accept reasoning_effort; Temperature must be omitted
        params.ReasoningEffort = openai.ReasoningEffort(opts.ReasoningEffort)
    }

    stream := c.client.Chat.Completions.NewStreaming(ctx, params)
    for stream.Next() {
        chunk := stream.Current()
        if len(chunk.Choices) > 0 {
            out <- chunk.Choices[0].Delta.Content
        }
    }
    return stream.Err()
}
```

> **Note**: o1/o3 models reject `temperature` and `top_p`. Do not set those fields
> when `UseReasoning` is true.

### 2d. Stream a "Thinking" Event to the UI

Reasoning models have a silent latency before the first token. Signal the UI so it
can show an indicator rather than a blank chat bubble:

```go
// internal/api/handlers_chat.go — before starting the LLM stream
if conv.UseReasoning {
    fmt.Fprint(w, "event: thinking\ndata: {}\n\n")
    flusher.Flush()
}

tokens := make(chan string)
go h.LLM.StreamWithOptions(r.Context(), msgs, llm.StreamOptions{
    UseReasoning:    conv.UseReasoning,
    ReasoningEffort: "medium", // could be user-configurable in a future phase
}, tokens)
```

The UI listens for the custom `thinking` SSE event and shows/dismisses the indicator
when the first `data:` token arrives.

**SSE event shape (chat stream)**:

| Event type | Payload | Meaning |
|---|---|---|
| `thinking` | `{}` | Reasoning model is processing; no tokens yet |
| *(default)* | `{"token":"..."}` | Streamed token |
| `tool_call` | `{"tool":"web_search","input":{...}}` | Tool being invoked (Feature 3) |
| `tool_result` | `{"tool":"web_search","summary":"..."}` | Tool completed |
| *(default)* | `[DONE]` | Stream complete |

---

## Feature 3 — MCP Gateway Integration

### Overview

`notebook-service` acts as the **MCP client**. A separate MCP gateway service
exposes:
- `GET /tools` — catalog of available tools across all configured MCP servers
- `POST /execute` — execute a single tool call, returns the result

`notebook-service` does not manage MCP server processes. It only calls the gateway.

### Open Source Gateway Options

Before building a custom gateway, evaluate:

| Option | Notes |
|---|---|
| [`mcp-proxy`](https://github.com/sparfenyuk/mcp-proxy) | Lightweight SSE/HTTP proxy for stdio MCP servers |
| [`mcphost`](https://github.com/mark3labs/mcphost) | CLI host; not ideal as a service |
| Custom Go service | Full control; build if open source options don't fit deployment needs |

If an open source gateway is adopted, only steps 3b onward are needed (the gateway's tool catalog and execute API may differ slightly — adapt the client accordingly).

### 3a. DB Migration — Store Per-Conversation Tool Selection

```sql
-- migrations/004_conversation_mcp.up.sql
CREATE TABLE conversation_mcp_tools (
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    tool_name       TEXT NOT NULL,
    PRIMARY KEY (conversation_id, tool_name)
);
```

```sql
-- migrations/004_conversation_mcp.down.sql
DROP TABLE conversation_mcp_tools;
```

### 3b. MCP Gateway Client

```go
// internal/mcp/client.go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type Tool struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    InputSchema json.RawMessage `json:"input_schema"` // JSON Schema
}

type ExecuteRequest struct {
    Tool  string          `json:"tool"`
    Input json.RawMessage `json:"input"`
}

type ExecuteResult struct {
    Content string `json:"content"`
    IsError bool   `json:"is_error"`
}

type Client interface {
    ListTools(ctx context.Context) ([]Tool, error)
    Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
}

type httpClient struct {
    base       string
    httpClient *http.Client
}

func NewClient(baseURL string) Client {
    return &httpClient{base: baseURL, httpClient: &http.Client{}}
}
```

### 3c. Config

```go
// internal/config/config.go
MCPGatewayURL string // e.g. "http://mcp-gateway:8090" — empty means MCP disabled
```

```go
MCPGatewayURL: getEnv("MCP_GATEWAY_URL", ""),
```

Wire `mcp.NewClient(cfg.MCPGatewayURL)` in `main.go`; pass `nil` when the URL is
empty and guard against nil in the chat handler.

### 3d. Tool Selection API

```
GET  /notebooks/{notebookID}/conversations/{conversationID}/tools
     → list enabled tool names for this conversation

PUT  /notebooks/{notebookID}/conversations/{conversationID}/tools
     Body: { "tools": ["web_search", "github"] }
     → replace enabled tool set (idempotent)

GET  /tools
     → proxy to MCP gateway's tool catalog (used by the UI to populate the Tools panel)
```

### 3e. Tool-Augmented Chat Handler

```go
// internal/api/handlers_chat.go (sketch)

// 1. Load enabled tools for this conversation
enabledTools, _ := h.ConvTools.ListEnabled(ctx, convID)

// 2. Fetch tool schemas from gateway (or from a short-lived cache)
var toolDefs []mcp.Tool
if h.MCP != nil && len(enabledTools) > 0 {
    allTools, _ := h.MCP.ListTools(ctx)
    toolDefs = filterEnabled(allTools, enabledTools)
}

// 3. Build initial prompt + pass tool schemas to LLM
msgs := h.Prompts.Build(history, userMessage, chunks)

// 4. Agentic loop — keep calling the LLM until no more tool_calls
for {
    toolCall, tokens, err := h.LLM.StreamUntilToolCallOrDone(ctx, msgs, toolDefs, out, flusher)
    if err != nil || toolCall == nil {
        break // stream is done
    }

    // 5. Emit tool_call event to UI
    emitSSE(w, flusher, "tool_call", map[string]any{
        "tool":  toolCall.Name,
        "input": toolCall.Input,
    })

    // 6. Execute via MCP gateway
    result, _ := h.MCP.Execute(ctx, mcp.ExecuteRequest{
        Tool:  toolCall.Name,
        Input: toolCall.Input,
    })

    // 7. Emit tool_result event to UI
    emitSSE(w, flusher, "tool_result", map[string]any{
        "tool":    toolCall.Name,
        "summary": truncate(result.Content, 200),
    })

    // 8. Feed result back into message history and loop
    msgs = appendToolResult(msgs, toolCall, result)
}
```

> **Note**: The agentic loop requires extending `LLMClient` with a method that
> returns when it either finishes streaming or emits a `tool_call`. OpenAI's
> streaming API includes `finish_reason: "tool_calls"` to signal this.

### 3f. Extend LLMClient for Tool Calls

```go
// internal/llm/llm.go
type ToolCall struct {
    ID    string
    Name  string
    Input json.RawMessage
}

type LLMClient interface {
    Complete(ctx context.Context, msgs []model.Message) (string, error)
    Stream(ctx context.Context, msgs []model.Message, out chan<- string) error
    StreamWithOptions(ctx context.Context, msgs []model.Message, opts StreamOptions, out chan<- string) error
    // StreamUntilToolCallOrDone streams tokens to `out` until the model either
    // finishes or requests a tool call. Returns the tool call if one was made.
    StreamUntilToolCallOrDone(
        ctx context.Context,
        msgs []model.Message,
        tools []mcp.Tool,
        opts StreamOptions,
        out chan<- string,
    ) (*ToolCall, error)
}
```

---

## Feature 4 — Langfuse Observability

`rag-anything` already uses Langfuse with the same trace pattern (retrieval span + LLM generation span). `notebook-service` adopts the same approach so both services produce traces in the same Langfuse project, giving a unified view of token usage, cost, latency, and RAG quality across the platform.

> Langfuse also accepts OTLP, which aligns with the existing OpenTelemetry deps in `go.sum`. Either integration path works; the native SDK approach below is simpler.

### 4a. Extend Config

```go
// internal/config/config.go
type LangfuseConfig struct {
    PublicKey string
    SecretKey string
    Host      string
}

type Config struct {
    // ... existing fields ...
    Langfuse LangfuseConfig
}

func Load() Config {
    return Config{
        // ...
        Langfuse: LangfuseConfig{
            PublicKey: getEnv("LANGFUSE_PUBLIC_KEY", ""),
            SecretKey: getEnv("LANGFUSE_SECRET_KEY", ""),
            Host:      getEnv("LANGFUSE_HOST", "https://cloud.langfuse.com"),
        },
    }
}
```

When `PublicKey` is empty, observability is silently disabled — no crash on startup.

### 4b. Create `internal/observability` Package

```go
// internal/observability/langfuse.go
package observability

import (
    "sync"
    "github.com/langfuse/langfuse-go"
    "github.com/vasantbala/notebook-service/internal/config"
)

var (
    once   sync.Once
    client *langfuse.Langfuse
)

func Init(cfg config.LangfuseConfig) {
    if cfg.PublicKey == "" {
        return // disabled
    }
    once.Do(func() {
        client = langfuse.New(
            langfuse.WithPublicKey(cfg.PublicKey),
            langfuse.WithSecretKey(cfg.SecretKey),
            langfuse.WithHost(cfg.Host),
        )
    })
}

// Get returns the client, or nil if observability is disabled.
func Get() *langfuse.Langfuse { return client }
```

Call `observability.Init(cfg.Langfuse)` in `main.go`.

### 4c. Instrument the Chat Handler

Wrap the chat request in a trace, mirroring `rag-anything`'s pattern:

```go
// internal/api/handlers_chat.go
lf := observability.Get()

var trace *langfuse.Trace
if lf != nil {
    trace = lf.Trace(&langfuse.TraceInput{
        Name:   "notebook-chat",
        UserID: &userID,
        Input:  &userMessage,
    })
}

// --- Retrieval span ---
var chunks []model.Chunk
if conv.RAGEnabled {
    chunks, _ = h.Retrieval.Retrieve(ctx, conv.NotebookID, userMessage)
    if trace != nil {
        span := trace.Span(&langfuse.SpanInput{Name: "retrieval"})
        span.End(&langfuse.SpanEndInput{
            Output: map[string]any{"chunk_count": len(chunks)},
        })
    }
}

// --- LLM generation span ---
msgs := h.Prompts.Build(history, userMessage, chunks)
var generation *langfuse.Generation
if trace != nil {
    generation = trace.Generation(&langfuse.GenerationInput{
        Name:  "llm-stream",
        Model: &modelName,
        Input: msgs,
    })
}

// ... stream tokens to client ...

if generation != nil {
    generation.End(&langfuse.GenerationEndInput{
        Output: &fullResponse,
        Usage: &langfuse.Usage{
            Input:  promptTokens,
            Output: completionTokens,
        },
    })
}
if trace != nil {
    trace.Update(&langfuse.TraceUpdateInput{Output: &fullResponse})
}
```

### 4d. MCP Tool Spans (when Feature 3 is active)

Each tool invocation gets its own span nested inside the trace:

```go
if trace != nil {
    toolSpan := trace.Span(&langfuse.SpanInput{
        Name:  "tool-call:" + toolCall.Name,
        Input: toolCall.Input,
    })
    // ... execute tool ...
    toolSpan.End(&langfuse.SpanEndInput{Output: result.Content})
}
```

### What Langfuse Shows

| View | Data |
|---|---|
| Per-trace | Full conversation turn: retrieval → LLM → tool calls |
| Per-user | Total tokens, cost, request count over time |
| Model breakdown | Token usage and latency per model (standard vs. reasoning) |
| RAG quality | Chunk count per retrieval, latency |
| Tool usage | Which MCP tools are invoked and how often |

---

## Build Order

| Step | Feature | What to build |
|---|---|---|
| 1 | RAG toggle | DB migration 003 (rag_enabled, use_reasoning, model columns), model update, PATCH handler extension, chat handler guard |
| 2 | Model selection | `GET /models` handler, model resolution logic in chat handler |
| 3 | Reasoning toggle | Config fields, `StreamOptions`, `StreamWithOptions` in openai.go, `thinking` SSE event |
| 4 | Langfuse observability | Config, `internal/observability` package, trace/span wrappers in chat handler |
| 5 | MCP catalog | `mcp.Client`, config, `GET /tools` proxy endpoint |
| 6 | MCP tool selection | `conversation_mcp_tools` table (migration 004), `GET/PUT /conversations/.../tools` handlers |
| 7 | MCP tool execution | Agentic loop in chat handler, `StreamUntilToolCallOrDone` LLM method |

---

## New Config Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `LLM_REASONING_MODEL` | `o3` | Model name used when `use_reasoning = true` |
| `MCP_GATEWAY_URL` | *(empty — MCP disabled)* | Base URL of the MCP gateway service |
| `LANGFUSE_PUBLIC_KEY` | *(required if Langfuse enabled)* | Langfuse project public key |
| `LANGFUSE_SECRET_KEY` | *(required if Langfuse enabled)* | Langfuse project secret key |
| `LANGFUSE_HOST` | `https://cloud.langfuse.com` | Langfuse host (use self-hosted URL if applicable) |

> **Token usage tracking**: `messages.token_count` (Phase 1 schema) is sufficient for context-window management. Per-request token usage, cost, latency, and RAG quality are tracked via **Langfuse** — see Feature 4.
>
> **Per-user MCP credentials**: deferred. Would require AES-GCM encrypted storage in a `user_mcp_credentials` table, a write-only CRUD API, and credential injection at execution time. Document only — do not implement in Phase 2.

---

## New Package Cheat Sheet

| Need | Package |
|---|---|
| HTTP client for MCP gateway | `net/http` (stdlib) |
| JSON Schema handling | `encoding/json` (stdlib) |
| Short-lived in-process cache (tool catalog) | `github.com/patrickmn/go-cache` |
| Langfuse tracing | `github.com/langfuse/langfuse-go` |

---

## Testing Notes

- **RAG toggle**: call `ChatStream` with `rag_enabled=false` on a conversation that
  has sources; assert that the retrieval service mock receives zero calls.
- **Reasoning toggle**: assert that `StreamWithOptions` is called with `UseReasoning=true`
  and that a `thinking` SSE event is emitted before the first token.
- **Langfuse**: inject a mock Langfuse client; assert that a trace is created with
  the correct `user_id`, that a retrieval span is recorded when RAG is on, that the
  LLM generation span captures `prompt_tokens` and `completion_tokens`, and that
  tool-call spans are created for each MCP tool invocation.
- **MCP tools**: inject a mock `mcp.Client`; assert that `Execute` is called with
  the correct tool name and that `tool_call` / `tool_result` SSE events are emitted
  in the right order.
