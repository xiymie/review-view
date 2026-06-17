package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
)

func (h *TaskHandler) APIDelete(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	task, err := h.tasks.GetByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "任务不存在"})
		return
	}

	if !isAdmin(c) {
		project, err := h.projects.GetByID(task.ProjectID)
		if err != nil || project.CreatedBy != callerUID(c) {
			c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
			return
		}
	}

	if task.Status == "running" || task.Status == "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "运行中或等待中的任务不能删除，请先取消"})
		return
	}
	if err := h.tasks.Delete(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func formatNullableTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func (h *TaskHandler) APIList(c *gin.Context) {
	var tasks []model.Task
	var err error

	if isAdmin(c) {
		tasks, err = h.tasks.ListRecent(100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else {
		userProjects, pErr := h.projects.ListByUser(callerUID(c))
		if pErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": pErr.Error()})
			return
		}
		ids := make([]int64, 0, len(userProjects))
		for _, p := range userProjects {
			ids = append(ids, p.ID)
		}
		tasks, err = h.tasks.ListByProjectIDs(ids, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}

	// 构建 projectID→name 映射（只查实际出现在任务里的项目）
	pidSet := make(map[int64]struct{}, len(tasks))
	for _, t := range tasks {
		pidSet[t.ProjectID] = struct{}{}
	}
	projectNames := make(map[int64]string, len(pidSet))
	projectOwnerIDs := make(map[int64]int64, len(pidSet))
	for pid := range pidSet {
		if p, e := h.projects.GetByID(pid); e == nil {
			projectNames[pid] = p.Name
			projectOwnerIDs[pid] = p.CreatedBy
		}
	}

	ownerIDSet := make(map[int64]struct{}, len(projectOwnerIDs))
	for _, uid := range projectOwnerIDs {
		if uid != 0 {
			ownerIDSet[uid] = struct{}{}
		}
	}
	ownerIDs := make([]int64, 0, len(ownerIDSet))
	for id := range ownerIDSet {
		ownerIDs = append(ownerIDs, id)
	}
	usernames := buildUsernameMap(h.users, ownerIDs)

	items := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		ownerUID := projectOwnerIDs[t.ProjectID]
		items = append(items, gin.H{
			"id":             t.ID,
			"project_id":     t.ProjectID,
			"project_name":   projectNames[t.ProjectID],
			"owner_username": usernames[ownerUID],
			"status":         t.Status,
			"from_commit":    t.FromCommit,
			"to_commit":      t.ToCommit,
			"from_subject":   t.FromSubject,
			"to_subject":     t.ToSubject,
			"triggered_by":   t.TriggeredBy,
			"created_at":     t.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, items)
}

func (h *TaskHandler) APIGet(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	task, err := h.tasks.GetByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}

	project, err := h.projects.GetByID(task.ProjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	if !isAdmin(c) && project.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权访问"})
		return
	}

	logs, _ := h.tasks.ListLogs(task.ID)

	logItems := make([]gin.H, 0, len(logs))
	for _, l := range logs {
		logItems = append(logItems, gin.H{
			"id":         l.ID,
			"level":      l.Level,
			"message":    l.Message,
			"created_at": l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"task": gin.H{
			"id":              task.ID,
			"project_id":      task.ProjectID,
			"status":          task.Status,
			"from_commit":     task.FromCommit,
			"to_commit":       task.ToCommit,
			"from_subject":    task.FromSubject,
			"to_subject":      task.ToSubject,
			"triggered_by":    task.TriggeredBy,
			"error_message":   task.ErrorMessage,
			"result":          task.Result,
			"diff_content":    task.DiffContent,
			"commit_messages": task.CommitMessages,
			"input_tokens":    task.InputTokens,
			"output_tokens":   task.OutputTokens,
			"created_at":      task.CreatedAt.Format("2006-01-02 15:04:05"),
			"started_at":      formatNullableTime(task.StartedAt),
			"finished_at":     formatNullableTime(task.FinishedAt),
		},
		"project_name": project.Name,
		"logs":         logItems,
	})
}
