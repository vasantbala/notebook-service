package api

import (
	"github.com/vasantbala/notebook-service/internal/config"
	"github.com/vasantbala/notebook-service/internal/llm"
	"github.com/vasantbala/notebook-service/internal/service"
)

type Handlers struct {
	Notebooks     service.NotebookService
	Conversations service.ConversationService
	Sources       service.SourceService
	Retrieval     service.RetrievalService
	LLM           llm.LLMClient
	Config        HandlerConfig
}

// HandlerConfig holds the subset of application config the handlers need.
type HandlerConfig struct {
	StandardModel   string
	ReasoningModel  string
	ReasoningEffort string
	LangfuseCfg     config.LangfuseConfig
}
