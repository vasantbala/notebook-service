package llm

import (
	"context"

	"github.com/vasantbala/notebook-service/internal/model"
)

type LLMClient interface {
	Complete(ctx context.Context, msgs []model.Message) (string, error)
	Stream(ctx context.Context, msgs []model.Message, out chan<- string) error
}
