package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
	"review-view/internal/store"
)

type TaskHandler struct {
	tasks      store.TaskStore
	projects   store.ProjectStore
	taskService *service.TaskService
	scheduler  *service.Scheduler
	cache      *service.TaskCache
	users      store.UserStore
}

func NewTaskHandler(tasks store.TaskStore, projects store.ProjectStore, taskService *service.TaskService, scheduler *service.Scheduler, cache *service.TaskCache, users store.UserStore) *TaskHandler {
	return &TaskHandler{
		tasks:       tasks,
		projects:    projects,
		taskService: taskService,
		scheduler:   scheduler,
		cache:       cache,
		users:       users,
	}
}

func (h *TaskHandler) Index(c *gin.Context) {
	tasks, err := h.tasks.ListRecent(100)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	projects, err := h.projects.List()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	projectNames := map[int64]string{}
	for _, project := range projects {
		projectNames[project.ID] = project.Name
	}

	items := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, gin.H{
			"Task":        task,
			"ProjectName": projectNames[task.ProjectID],
		})
	}

	c.HTML(http.StatusOK, "tasks/index", gin.H{
		"Title":        "任务",
		"Active":       "tasks",
		"Tasks":        items,
	})
}

func (h *TaskHandler) Show(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	task, err := h.tasks.GetByID(taskID)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}
	project, err := h.projects.GetByID(task.ProjectID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	logs, _ := h.tasks.ListLogs(task.ID)

	c.HTML(http.StatusOK, "tasks/show", gin.H{
		"Title":   "任务详情",
		"Active":  "tasks",
		"Task":    task,
		"Project": project,
		"Logs":    logs,
		"Breadcrumbs": []breadcrumb{
			{Label: "任务", Href: "/tasks"},
			{Label: "#" + strconv.FormatInt(task.ID, 10) + " 详情"},
		},
	})
}

func (h *TaskHandler) Cancel(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if !isAdmin(c) {
		task, err := h.tasks.GetByID(taskID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "任务不存在"})
			return
		}
		project, err := h.projects.GetByID(task.ProjectID)
		if err != nil || project.CreatedBy != callerUID(c) {
			c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
			return
		}
	}

	if err := h.scheduler.CancelTask(taskID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *TaskHandler) Retry(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if !isAdmin(c) {
		task, err := h.tasks.GetByID(taskID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "任务不存在"})
			return
		}
		project, err := h.projects.GetByID(task.ProjectID)
		if err != nil || project.CreatedBy != callerUID(c) {
			c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
			return
		}
	}

	task, err := h.taskService.Retry(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task_id": task.ID})
}

// Stream 通过 SSE 推送运行中任务的增量日志和状态变更
func (h *TaskHandler) Stream(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	task, err := h.tasks.GetByID(taskID)
	if err != nil {
		c.String(http.StatusNotFound, "task not found")
		return
	}

	// 已结束的任务直接返回当前状态，不建立 SSE
	if task.Status != model.TaskStatusRunning && task.Status != model.TaskStatusPending {
		c.JSON(http.StatusOK, gin.H{
			"status":        string(task.Status),
			"result":        task.Result,
			"done":          true,
			"input_tokens":  task.InputTokens,
			"output_tokens": task.OutputTokens,
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 发送初始全量状态（DB 中已有的日志）
	initLogs, _ := h.tasks.ListLogs(taskID)
	for _, log := range initLogs {
		data, _ := json.Marshal(gin.H{
			"id":        log.ID,
			"level":     string(log.Level),
			"message":   log.Message,
			"createdAt": log.CreatedAt.Format(time.RFC3339),
		})
		fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", data)
	}
	c.Writer.(http.Flusher).Flush()

	// 订阅增量通知
	notify := h.cache.Notify()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	sentCount := 0
	sentOutputTokens := int64(-1)
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case notifiedID := <-notify:
			if notifiedID != taskID {
				continue
			}
			// 推送增量日志
			logs := h.cache.GetLogs(taskID)
			for i := sentCount; i < len(logs); i++ {
				log := logs[i]
				data, _ := json.Marshal(gin.H{
					"id":        log.ID,
					"level":     string(log.Level),
					"message":   log.Message,
					"createdAt": log.CreatedAt.Format(time.RFC3339),
				})
				fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", data)
			}
			sentCount = len(logs)

			// 推送结果快照
			if result := h.cache.GetResult(taskID); result != "" {
				data, _ := json.Marshal(gin.H{"content": result})
				fmt.Fprintf(c.Writer, "event: result\ndata: %s\n\n", data)
			}

			// 推送实时 token 数（仅变化时推送）
			inTok, outTok := h.cache.GetTokens(taskID)
			if outTok != sentOutputTokens {
				data, _ := json.Marshal(gin.H{"input_tokens": inTok, "output_tokens": outTok})
				fmt.Fprintf(c.Writer, "event: token\ndata: %s\n\n", data)
				sentOutputTokens = outTok
			}
			c.Writer.(http.Flusher).Flush()

			// 检查任务是否已结束
			latest, err := h.tasks.GetByID(taskID)
			if err != nil {
				return
			}
			if latest.Status != model.TaskStatusRunning && latest.Status != model.TaskStatusPending {
				data, _ := json.Marshal(gin.H{
					"status":        string(latest.Status),
					"result":        latest.Result,
					"input_tokens":  latest.InputTokens,
					"output_tokens": latest.OutputTokens,
				})
				fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", data)
				c.Writer.(http.Flusher).Flush()
				return
			}
		case <-ticker.C:
			// 定期推送结果快照并检查任务状态
			if result := h.cache.GetResult(taskID); result != "" {
				data, _ := json.Marshal(gin.H{"content": result})
				fmt.Fprintf(c.Writer, "event: result\ndata: %s\n\n", data)
			}

			// 定期推送 token 数
			inTok, outTok := h.cache.GetTokens(taskID)
			if outTok != sentOutputTokens {
				data, _ := json.Marshal(gin.H{"input_tokens": inTok, "output_tokens": outTok})
				fmt.Fprintf(c.Writer, "event: token\ndata: %s\n\n", data)
				sentOutputTokens = outTok
			}
			c.Writer.(http.Flusher).Flush()

			latest, err := h.tasks.GetByID(taskID)
			if err != nil {
				return
			}
			if latest.Status != model.TaskStatusRunning && latest.Status != model.TaskStatusPending {
				data, _ := json.Marshal(gin.H{
					"status":        string(latest.Status),
					"result":        latest.Result,
					"input_tokens":  latest.InputTokens,
					"output_tokens": latest.OutputTokens,
				})
				fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", data)
				c.Writer.(http.Flusher).Flush()
				return
			}
		}
	}
}
