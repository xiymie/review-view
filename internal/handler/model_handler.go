package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/review"
	"review-view/internal/service"
)

type modelTestRequest struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key"`
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	MaxContext     int    `json:"max_context"`
	EnableThinking bool   `json:"enable_thinking"`
	CLIPath        string `json:"cli_path"`
	EnvVarsJSON    string `json:"env_vars_json"`
	MaxTurns       *int   `json:"max_turns"`
}

type ModelHandler struct {
	service *service.ModelConfigService
}

func NewModelHandler(service *service.ModelConfigService) *ModelHandler {
	return &ModelHandler{service: service}
}

func (h *ModelHandler) Index(c *gin.Context) {
	models, err := h.service.List()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "models/index", gin.H{
		"Title":        "模型配置",
		"Active":       "models",
		"PageTemplate": "models/index_content",
		"Models":       models,
	})
}

func (h *ModelHandler) New(c *gin.Context) {
	c.HTML(http.StatusOK, "models/new", gin.H{
		"Title":        "新建配置",
		"Active":       "models",
		"PageTemplate": "models/new_content",
		"Breadcrumbs": []breadcrumb{
			{Label: "模型配置", Href: "/models"},
			{Label: "新建配置"},
		},
	})
}

func (h *ModelHandler) Create(c *gin.Context) {
	maxContext, _ := strconv.Atoi(c.PostForm("max_context"))
	maxTurns := parseOptionalInt(c.PostForm("max_turns"))

	_, err := h.service.Create(service.ModelConfigCreateInput{
		Name:           c.PostForm("name"),
		Type:           model.ModelType(c.PostForm("type")),
		BaseURL:        c.PostForm("base_url"),
		APIKey:         c.PostForm("api_key"),
		Model:          c.PostForm("model"),
		Prompt:         c.PostForm("prompt"),
		MaxContext:     maxContext,
		EnableThinking: c.PostForm("enable_thinking") == "on",
		CLIPath:        c.PostForm("cli_path"),
		EnvVarsJSON:    c.PostForm("env_vars_json"),
		MaxTurns:       maxTurns,
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/models")
}

func (h *ModelHandler) Edit(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	config, err := h.service.Get(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	// 反序列化 CLI ExtraConfig
	cliExtra := model.ClaudeCLIExtraConfig{}
	if config.Type == model.ModelTypeClaudeCLI {
		_ = config.DecodeExtraConfig(&cliExtra)
	}

	c.HTML(http.StatusOK, "models/edit", gin.H{
		"Title":        "编辑配置",
		"Active":       "models",
		"PageTemplate": "models/edit_content",
		"Config":       config,
		"CLIExtra":     cliExtra,
		"Breadcrumbs": []breadcrumb{
			{Label: "模型配置", Href: "/models"},
			{Label: config.Name},
		},
	})
}

func (h *ModelHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	maxContext, _ := strconv.Atoi(c.PostForm("max_context"))
	maxTurns := parseOptionalInt(c.PostForm("max_turns"))

	_, err := h.service.Update(id, service.ModelConfigCreateInput{
		Name:           c.PostForm("name"),
		Type:           model.ModelType(c.PostForm("type")),
		BaseURL:        c.PostForm("base_url"),
		APIKey:         c.PostForm("api_key"),
		Model:          c.PostForm("model"),
		Prompt:         c.PostForm("prompt"),
		MaxContext:     maxContext,
		EnableThinking: c.PostForm("enable_thinking") == "on",
		CLIPath:        c.PostForm("cli_path"),
		EnvVarsJSON:    c.PostForm("env_vars_json"),
		MaxTurns:       maxTurns,
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/models")
}

// Test 测试模型配置是否可用
func (h *ModelHandler) Test(c *gin.Context) {
	var req modelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	input := service.ModelConfigCreateInput{
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
	}

	config, err := h.service.BuildConfig(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	if config.Type == model.ModelTypeClaudeCLI {
		// CLI 模式暂时只校验配置合法性
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "CLI 配置已就绪"})
		return
	}

	provider, err := review.NewProvider(config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "创建 provider 失败: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := provider.Completion(ctx, review.TestCompletionParams(config))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "请求失败: " + err.Error()})
		return
	}

	msg := ""
	if len(result.Choices) > 0 {
		msg = fmt.Sprint(result.Choices[0].Message.Content)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": msg})
}
