package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
	"review-view/internal/store"
)

type ProjectHandler struct {
	projects    *service.ProjectService
	models      *service.ModelConfigService
	taskService *service.TaskService
	tasks       store.TaskStore
	credentials *service.RepoCredentialService
	users       store.UserStore
}

func NewProjectHandler(projects *service.ProjectService, models *service.ModelConfigService, taskService *service.TaskService, tasks store.TaskStore, credentials *service.RepoCredentialService, users store.UserStore) *ProjectHandler {
	return &ProjectHandler{
		projects:    projects,
		models:      models,
		taskService: taskService,
		tasks:       tasks,
		credentials: credentials,
		users:       users,
	}
}

func (h *ProjectHandler) Index(c *gin.Context) {
	projects, err := h.projects.List()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "projects/index", gin.H{
		"Title":        "项目",
		"Active":       "projects",
		"PageTemplate": "projects/index_content",
		"Projects":     projects,
	})
}

func (h *ProjectHandler) New(c *gin.Context) {
	models, err := h.models.List()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	creds, _ := h.credentials.List()

	data := gin.H{
		"Title":        "新建项目",
		"Active":       "projects",
		"PageTemplate": "projects/new_content",
		"Breadcrumbs": []breadcrumb{
			{Label: "项目", Href: "/projects"},
			{Label: "新建项目"},
		},
		"ModelConfigs": models,
		"Credentials":  creds,
	}

	// 支持克隆：从 clone_from 参数读取源项目配置
	if cloneID := c.Query("clone_from"); cloneID != "" {
		if id, err := strconv.ParseInt(cloneID, 10, 64); err == nil {
			if project, err := h.projects.Get(id); err == nil {
				data["CloneFrom"] = project
			}
		}
	}

	c.HTML(http.StatusOK, "projects/new", data)
}

func (h *ProjectHandler) Create(c *gin.Context) {
	modelConfigID, _ := strconv.ParseInt(c.PostForm("model_config_id"), 10, 64)
	timeout := parseOptionalInt(c.PostForm("task_timeout"))

	overflow := model.OverflowStrategy(c.PostForm("overflow_strategy"))
	if overflow == "" {
		overflow = model.OverflowStrategyQueue
	}

	var credID *int64
	if v := c.PostForm("repo_credential_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		credID = &id
	}

	project, err := h.projects.Create(service.ProjectCreateInput{
		Name:             c.PostForm("name"),
		RepoURL:          c.PostForm("repo_url"),
		Branch:           c.PostForm("branch"),
		ModelConfigID:    modelConfigID,
		CustomPrompt:     c.PostForm("custom_prompt"),
		OverflowStrategy: overflow,
		TaskTimeout:      timeout,
		RepoCredentialID: credID,
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/projects/"+strconv.FormatInt(project.ID, 10))
}

func (h *ProjectHandler) Show(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	modelConfig, _ := h.models.Get(project.ModelConfigID)
	tasks, _ := h.taskService.ListByProject(project.ID, 20)

	c.HTML(http.StatusOK, "projects/show", gin.H{
		"Title":        project.Name,
		"Active":       "projects",
		"PageTemplate": "projects/show_content",
		"Project":      project,
		"ModelConfig":  modelConfig,
		"Tasks":        tasks,
		"Breadcrumbs": []breadcrumb{
			{Label: "项目", Href: "/projects"},
			{Label: project.Name},
		},
	})
}

func (h *ProjectHandler) Edit(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	modelConfigs, err := h.models.List()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	creds, _ := h.credentials.List()

	c.HTML(http.StatusOK, "projects/edit", gin.H{
		"Title":        "编辑项目",
		"Active":       "projects",
		"PageTemplate": "projects/edit_content",
		"Project":      project,
		"ModelConfigs": modelConfigs,
		"Credentials":  creds,
		"Breadcrumbs": []breadcrumb{
			{Label: "项目", Href: "/projects"},
			{Label: project.Name, Href: "/projects/" + strconv.FormatInt(project.ID, 10)},
			{Label: "编辑"},
		},
	})
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	modelConfigID, _ := strconv.ParseInt(c.PostForm("model_config_id"), 10, 64)
	timeout := parseOptionalInt(c.PostForm("task_timeout"))

	overflow := model.OverflowStrategy(c.PostForm("overflow_strategy"))
	if overflow == "" {
		overflow = model.OverflowStrategyQueue
	}

	var credID *int64
	if v := c.PostForm("repo_credential_id"); v != "" {
		credIDInt, _ := strconv.ParseInt(v, 10, 64)
		credID = &credIDInt
	}

	if _, err := h.projects.Update(id, service.ProjectCreateInput{
		Name:             c.PostForm("name"),
		RepoURL:          c.PostForm("repo_url"),
		Branch:           c.PostForm("branch"),
		ModelConfigID:    modelConfigID,
		CustomPrompt:     c.PostForm("custom_prompt"),
		OverflowStrategy: overflow,
		TaskTimeout:      timeout,
		RepoCredentialID: credID,
	}); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/projects/"+strconv.FormatInt(id, 10))
}

func (h *ProjectHandler) TriggerManual(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	task, skipped, err := h.taskService.Trigger(c.Request.Context(), service.TriggerInput{
		ProjectID:    id,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: c.PostForm("to_commit"),
		FromCommit:   c.PostForm("from_commit"),
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	if skipped {
		c.Redirect(http.StatusSeeOther, "/projects/"+strconv.FormatInt(id, 10))
		return
	}

	c.Redirect(http.StatusSeeOther, "/tasks/"+strconv.FormatInt(task.ID, 10))
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.projects.Delete(id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/projects")
}

func (h *ProjectHandler) Initialize(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.projects.Initialize(id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/projects/"+strconv.FormatInt(id, 10))
}

func (h *ProjectHandler) Commits(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	project, err := h.projects.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 200 {
		limit = l
	}

	commits, err := h.taskService.ListCommits(c.Request.Context(), project.ID, project.Branch, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, commits)
}
