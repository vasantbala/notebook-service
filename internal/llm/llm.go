package llm

import (
	"context"

	"github.com/vasantbala/notebook-service/internal/model"
)

// StreamOptions controls model selection and reasoning behaviour for a chat request.
type StreamOptions struct {
	// UseReasoning routes the request to the configured reasoning model (e.g. o3).
	// Reasoning models use ReasoningEffort instead of temperature.
	UseReasoning bool
	// ReasoningEffort is passed to reasoning models ("low", "medium", "high").
	// Ignored when UseReasoning is false.
	ReasoningEffort string
	// ModelOverride pins this specific model, overriding both defaults.
	// Nil means use the service default (or reasoning model when UseReasoning is true).
	ModelOverride *string
}

type LLMClient interface {
	Complete(ctx context.Context, msgs []model.Message) (string, error)
	Stream(ctx context.Context, msgs []model.Message, out chan<- string) error
	StreamWithOptions(ctx context.Context, msgs []model.Message, opts StreamOptions, out chan<- string) error
}
