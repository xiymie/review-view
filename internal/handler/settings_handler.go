package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
)

type SettingsHandler struct {
	service *service.SettingsService
}

func NewSettingsHandler(service *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

func (h *SettingsHandler) Index(c *gin.Context) {
	settings, err := h.service.Get()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "settings/index", gin.H{
		"Title":        "设置",
		"Active":       "settings",
		"PageTemplate": "settings/index_content",
		"Settings":     settings,
	})
}

func (h *SettingsHandler) Update(c *gin.Context) {
	maxConcurrent, _ := strconv.Atoi(c.PostForm("max_concurrent_tasks"))
	taskTimeout, _ := strconv.Atoi(c.PostForm("task_timeout"))

	err := h.service.Update(service.SettingsInput{
		MaxConcurrentTasks: maxConcurrent,
		OverflowStrategy:   model.OverflowStrategy(c.PostForm("global_overflow_strategy")),
		TaskTimeout:        taskTimeout,
		RepoBaseDir:        c.PostForm("repo_base_dir"),
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/settings")
}
