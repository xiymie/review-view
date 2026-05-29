package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
)

func (h *ProjectHandler) APIList(c *gin.Context) {
	var projects []model.Project
	var err error
	if isAdmin(c) {
		projects, err = h.projects.List()
	} else {
		projects, err = h.projects.ListByUser(callerUID(c))
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	uidSet := make(map[int64]struct{}, len(projects))
	for _, p := range projects {
		if p.CreatedBy != 0 {
			uidSet[p.CreatedBy] = struct{}{}
		}
	}
	ownerIDs := make([]int64, 0, len(uidSet))
	for id := range uidSet {
		ownerIDs = append(ownerIDs, id)
	}
	usernames := buildUsernameMap(h.users, ownerIDs)

	items := make([]gin.H, 0, len(projects))
	for _, p := range projects {
		items = append(items, gin.H{
			"id":                   p.ID,
			"name":                 p.Name,
			"repo_url":             p.RepoURL,
			"branch":               p.Branch,
			"model_config_id":      p.ModelConfigID,
			"repo_credential_id":   p.RepoCredentialID,
			"status":               p.Status,
			"last_reviewed_commit": p.LastReviewedCommit,
			"overflow_strategy":    p.OverflowStrategy,
			"task_timeout":         p.TaskTimeout,
			"cron_expression":      p.CronExpression,
			"cron_enabled":         p.CronEnabled,
			"next_run_at":          p.NextRunAt,
			"created_by":           p.CreatedBy,
			"owner_username":       usernames[p.CreatedBy],
			"created_at":           p.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	c.JSON(http.StatusOK, items)
}

func (h *ProjectHandler) APIGet(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}

	if !isAdmin(c) && project.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权访问"})
		return
	}

	modelConfig, _ := h.models.Get(project.ModelConfigID)
	tasks, _ := h.taskService.ListByProject(project.ID, 20)
	modelConfigs, _ := h.models.List()

	var creds []model.RepoCredential
	if isAdmin(c) {
		creds, _ = h.credentials.List()
	} else {
		creds, _ = h.credentials.ListByUser(callerUID(c))
	}

	taskItems := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		taskItems = append(taskItems, gin.H{
			"id":           t.ID,
			"status":       t.Status,
			"from_commit":  t.FromCommit,
			"to_commit":    t.ToCommit,
			"triggered_by": t.TriggeredBy,
			"created_at":   t.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	mcItems := make([]gin.H, 0, len(modelConfigs))
	for _, mc := range modelConfigs {
		mcItems = append(mcItems, gin.H{
			"id":   mc.ID,
			"name": mc.Name,
		})
	}

	credItems := make([]gin.H, 0, len(creds))
	for _, cr := range creds {
		credItems = append(credItems, gin.H{
			"id":   cr.ID,
			"name": cr.Name,
		})
	}

	var mcID int64
	var mcName string
	if modelConfig != nil {
		mcID = modelConfig.ID
		mcName = modelConfig.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"project": gin.H{
			"id":                   project.ID,
			"name":                 project.Name,
			"repo_url":             project.RepoURL,
			"branch":               project.Branch,
			"model_config_id":      project.ModelConfigID,
			"repo_credential_id":   project.RepoCredentialID,
			"status":               project.Status,
			"last_reviewed_commit": project.LastReviewedCommit,
			"overflow_strategy":    project.OverflowStrategy,
			"task_timeout":         project.TaskTimeout,
			"cron_expression":      project.CronExpression,
			"cron_enabled":         project.CronEnabled,
			"next_run_at":          project.NextRunAt,
			"created_by":           project.CreatedBy,
			"created_at":           project.CreatedAt.Format("2006-01-02 15:04"),
		},
		"model_config": gin.H{
			"id":   mcID,
			"name": mcName,
		},
		"tasks":         taskItems,
		"model_configs": mcItems,
		"credentials":   credItems,
	})
}

func (h *ProjectHandler) APICreate(c *gin.Context) {
	var req struct {
		Name             string                 `json:"name"`
		RepoURL          string                 `json:"repo_url"`
		Branch           string                 `json:"branch"`
		ModelConfigID    int64                  `json:"model_config_id"`
		CustomPrompt     string                 `json:"custom_prompt"`
		OverflowStrategy model.OverflowStrategy `json:"overflow_strategy"`
		TaskTimeout      *int                   `json:"task_timeout"`
		RepoCredentialID *int64                 `json:"repo_credential_id"`
		CronExpression   string                 `json:"cron_expression"`
		CronEnabled      bool                   `json:"cron_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if req.OverflowStrategy == "" {
		req.OverflowStrategy = model.OverflowStrategyQueue
	}

	project, err := h.projects.Create(service.ProjectCreateInput{
		Name:             req.Name,
		RepoURL:          req.RepoURL,
		Branch:           req.Branch,
		ModelConfigID:    req.ModelConfigID,
		CustomPrompt:     req.CustomPrompt,
		OverflowStrategy: req.OverflowStrategy,
		TaskTimeout:      req.TaskTimeout,
		RepoCredentialID: req.RepoCredentialID,
		CronExpression:   req.CronExpression,
		CronEnabled:      req.CronEnabled,
		CreatedBy:        callerUID(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":   project.ID,
		"name": project.Name,
	})
}

func (h *ProjectHandler) APIUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && project.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
		return
	}

	var req struct {
		Name             string                 `json:"name"`
		RepoURL          string                 `json:"repo_url"`
		Branch           string                 `json:"branch"`
		ModelConfigID    int64                  `json:"model_config_id"`
		CustomPrompt     string                 `json:"custom_prompt"`
		OverflowStrategy model.OverflowStrategy `json:"overflow_strategy"`
		TaskTimeout      *int                   `json:"task_timeout"`
		RepoCredentialID *int64                 `json:"repo_credential_id"`
		CronExpression   string                 `json:"cron_expression"`
		CronEnabled      bool                   `json:"cron_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if req.OverflowStrategy == "" {
		req.OverflowStrategy = model.OverflowStrategyQueue
	}

	updated, err := h.projects.Update(id, service.ProjectCreateInput{
		Name:             req.Name,
		RepoURL:          req.RepoURL,
		Branch:           req.Branch,
		ModelConfigID:    req.ModelConfigID,
		CustomPrompt:     req.CustomPrompt,
		OverflowStrategy: req.OverflowStrategy,
		TaskTimeout:      req.TaskTimeout,
		RepoCredentialID: req.RepoCredentialID,
		CronExpression:   req.CronExpression,
		CronEnabled:      req.CronEnabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   updated.ID,
		"name": updated.Name,
	})
}

func (h *ProjectHandler) APIDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && project.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
		return
	}

	if err := h.projects.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *ProjectHandler) APITrigger(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && project.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
		return
	}

	var req struct {
		FromCommit string `json:"from_commit"`
		ToCommit   string `json:"to_commit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	task, skipped, err := h.taskService.Trigger(c.Request.Context(), service.TriggerInput{
		ProjectID:    id,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: req.ToCommit,
		FromCommit:   req.FromCommit,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var taskID int64
	if task != nil {
		taskID = task.ID
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"skipped": skipped,
	})
}

func (h *ProjectHandler) APIInitialize(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && project.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
		return
	}

	if err := h.projects.Initialize(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *ProjectHandler) APIUpdateSchedule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && project.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
		return
	}

	var req struct {
		CronExpression string `json:"cron_expression"`
		CronEnabled    bool   `json:"cron_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	updated, err := h.projects.UpdateSchedule(id, req.CronExpression, req.CronEnabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              updated.ID,
		"cron_expression": updated.CronExpression,
		"cron_enabled":    updated.CronEnabled,
		"next_run_at":     updated.NextRunAt,
	})
}
