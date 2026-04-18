package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/vasantbala/notebook-service/internal/model"
)

type openAIClient struct {
	client         openai.Client // value type — NewClient returns Client, not *Client
	model          string
	reasoningModel string
}

func NewOpenAIClient(apiKey, model, baseURL, reasoningModel string) LLMClient {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &openAIClient{
		client:         openai.NewClient(opts...),
		model:          model,
		reasoningModel: reasoningModel,
	}
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
	return c.StreamWithOptions(ctx, msgs, StreamOptions{}, out)
}

func (c *openAIClient) StreamWithOptions(ctx context.Context, msgs []model.Message, opts StreamOptions, out chan<- string) error {
	defer close(out)

	// Resolve effective model: explicit override > reasoning default > standard default.
	modelName := c.model
	if opts.UseReasoning && c.reasoningModel != "" {
		modelName = c.reasoningModel
	}
	if opts.ModelOverride != nil && *opts.ModelOverride != "" {
		modelName = *opts.ModelOverride
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(modelName),
		Messages: toOpenAIMessages(msgs),
	}

	// Reasoning models (o1/o3) do not accept temperature; they use reasoning_effort.
	if opts.UseReasoning && opts.ReasoningEffort != "" {
		params.ReasoningEffort = openai.ReasoningEffort(opts.ReasoningEffort)
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
