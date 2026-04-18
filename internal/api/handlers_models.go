package api

import (
	"net/http"

	"github.com/vasantbala/notebook-service/internal/util"
)

// modelInfo describes an available LLM model.
type modelInfo struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Reasoning bool   `json:"reasoning"`
}

// ListModels godoc
//
// @Summary      List available models
// @Description  Returns the LLM models configured for this service instance.
// @Tags         models
// @Produce      json
// @Success      200  {array}  modelInfo
// @Router       /models [get]
func (h *Handlers) ListModels(w http.ResponseWriter, r *http.Request) {
	models := []modelInfo{
		{ID: h.Config.StandardModel, Label: h.Config.StandardModel, Reasoning: false},
	}
	if h.Config.ReasoningModel != "" {
		models = append(models, modelInfo{
			ID:        h.Config.ReasoningModel,
			Label:     h.Config.ReasoningModel + " (Reasoning)",
			Reasoning: true,
		})
	}
	util.WriteJSON(w, http.StatusOK, models)
}
