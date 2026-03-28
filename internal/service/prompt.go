package service

import (
	"fmt"
	"strings"

	"github.com/vasantbala/notebook-service/internal/model"
)

const defaultSystemPrompt = `You are a helpful assistant. Answer questions based on the provided document excerpts.
If the answer is not contained in the documents, say so clearly rather than making up information.`

// BuildChatMessages assembles the full message list to send to the LLM:
//  1. System prompt
//  2. Retrieved document chunks as a second system message (if any)
//  3. Conversation history (user + assistant turns)
//  4. The new user query
func BuildChatMessages(systemPrompt string, chunks []ChunkResult, history []model.Message, userQuery string) []model.Message {
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}

	msgs := []model.Message{
		{Role: model.RoleSystem, Content: systemPrompt},
	}

	if len(chunks) > 0 {
		msgs = append(msgs, model.Message{
			Role:    model.RoleSystem,
			Content: "Relevant document excerpts:\n\n" + buildContextBlock(chunks),
		})
	}

	for _, m := range history {
		msgs = append(msgs, model.Message{Role: m.Role, Content: m.Content})
	}

	msgs = append(msgs, model.Message{Role: model.RoleUser, Content: userQuery})
	return msgs
}

// buildContextBlock formats retrieved chunks as a numbered, scored list.
func buildContextBlock(chunks []ChunkResult) string {
	var sb strings.Builder
	for i, c := range chunks {
		fmt.Fprintf(&sb, "[%d] (score: %.2f)\n%s\n\n", i+1, c.RerankerScore, c.Text)
	}
	return sb.String()
}
