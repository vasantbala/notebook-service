package api

import (
	"github.com/vasantbala/notebook-service/internal/llm"
	"github.com/vasantbala/notebook-service/internal/service"
)

type Handlers struct {
	Notebooks     service.NotebookService
	Conversations service.ConversationService
	Sources       service.SourceService
	Retrieval     service.RetrievalService
	LLM           llm.LLMClient
}
