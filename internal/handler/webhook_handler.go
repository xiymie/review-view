package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
)

type WebhookHandler struct {
	taskService *service.TaskService
}

func NewWebhookHandler(taskService *service.TaskService) *WebhookHandler {
	return &WebhookHandler{taskService: taskService}
}

func (h *WebhookHandler) Trigger(c *gin.Context) {
	projectID, _ := strconv.ParseInt(c.Param("projectId"), 10, 64)

	var payload struct {
		Commit string `json:"commit"`
	}
	_ = c.ShouldBindJSON(&payload)

	task, skipped, err := h.taskService.Trigger(c.Request.Context(), service.TriggerInput{
		ProjectID:    projectID,
		TriggeredBy:  model.TaskTriggeredByWebhook,
		TargetCommit: payload.Commit,
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"skipped": skipped,
		"task_id": func() int64 {
			if task == nil {
				return 0
			}
			return task.ID
		}(),
	})
}
