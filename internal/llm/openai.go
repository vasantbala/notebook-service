package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/vasantbala/notebook-service/internal/model"
)

type openAIClient struct {
	client openai.Client // value type — NewClient returns Client, not *Client
	model  string
}

func NewOpenAIClient(apiKey, model, baseURL string) LLMClient {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &openAIClient{client: openai.NewClient(opts...), model: model}
}

func (c *openAIClient) Complete(ctx context.Context, msgs []model.Message) (string, error) {
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.model),
		Messages: toOpenAIMessages(msgs),
	})
	if err != nil {
		return "", fmt.Errorf("llm complete: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("llm complete: no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *openAIClient) Stream(ctx context.Context, msgs []model.Message, out chan<- string) error {
	defer close(out)

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.model),
		Messages: toOpenAIMessages(msgs),
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			out <- chunk.Choices[0].Delta.Content
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("llm stream: %w", err)
	}
	return nil
}

// toOpenAIMessages converts our internal Message type to the SDK's union param type.
// The SDK provides UserMessage, SystemMessage, and AssistantMessage constructors
// that each return a ChatCompletionMessageParamUnion.
func toOpenAIMessages(msgs []model.Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, len(msgs))
	for i, m := range msgs {
		switch m.Role {
		case model.RoleUser:
			out[i] = openai.UserMessage(m.Content)
		case model.RoleAssistant:
			out[i] = openai.AssistantMessage(m.Content)
		default: // system, or any future role
			out[i] = openai.SystemMessage(m.Content)
		}
	}
	return out
}
