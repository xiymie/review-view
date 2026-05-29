package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
)

func (h *ModelHandler) APIList(c *gin.Context) {
	models, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	type item struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		Type           string `json:"type"`
		Model          string `json:"model"`
		EnableThinking bool   `json:"enable_thinking"`
		CreatedAt      string `json:"created_at"`
	}

	result := make([]item, 0, len(models))
	for _, m := range models {
		result = append(result, item{
			ID:             m.ID,
			Name:           m.Name,
			Type:           string(m.Type),
			Model:          m.Model,
			EnableThinking: m.EnableThinking,
			CreatedAt:      m.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	c.JSON(http.StatusOK, result)
}

func (h *ModelHandler) APIGet(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	config, err := h.service.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}

	cliExtra := model.ClaudeCLIExtraConfig{}
	if config.Type == model.ModelTypeClaudeCLI {
		_ = config.DecodeExtraConfig(&cliExtra)
	}

	var envVarsJSON string
	if len(cliExtra.EnvVars) > 0 {
		b, _ := json.Marshal(cliExtra.EnvVars)
		envVarsJSON = string(b)
	}

	var maxTurns *int
	if config.Type == model.ModelTypeClaudeCLI && cliExtra.MaxTurns != 0 {
		v := cliExtra.MaxTurns
		maxTurns = &v
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              config.ID,
		"name":            config.Name,
		"type":            string(config.Type),
		"base_url":        config.BaseURL,
		"api_key":         config.APIKey,
		"model":           config.Model,
		"prompt":          config.Prompt,
		"max_context":     config.MaxContext,
		"enable_thinking": config.EnableThinking,
		"cli_path":        cliExtra.CLIPath,
		"env_vars_json":   envVarsJSON,
		"max_turns":       maxTurns,
		"created_at":      config.CreatedAt.Format("2006-01-02 15:04"),
	})
}

func (h *ModelHandler) APICreate(c *gin.Context) {
	var req modelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	config, err := h.service.Create(service.ModelConfigCreateInput{
		Name:           req.Name,
		Type:           model.ModelType(req.Type),
		BaseURL:        req.BaseURL,
		APIKey:         req.APIKey,
		Model:          req.Model,
		Prompt:         req.Prompt,
		MaxContext:     req.MaxContext,
		EnableThinking: req.EnableThinking,
		CLIPath:        req.CLIPath,
		EnvVarsJSON:    req.EnvVarsJSON,
		MaxTurns:       req.MaxTurns,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": config.ID, "name": config.Name})
}

func (h *ModelHandler) APIDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *ModelHandler) APIUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req modelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	config, err := h.service.Update(id, service.ModelConfigCreateInput{
		Name:           req.Name,
		Type:           model.ModelType(req.Type),
		BaseURL:        req.BaseURL,
		APIKey:         req.APIKey,
		Model:          req.Model,
		Prompt:         req.Prompt,
		MaxContext:     req.MaxContext,
		EnableThinking: req.EnableThinking,
		CLIPath:        req.CLIPath,
		EnvVarsJSON:    req.EnvVarsJSON,
		MaxTurns:       req.MaxTurns,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": config.ID, "name": config.Name})
}
