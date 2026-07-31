package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
)

type ReviewSkillHandler struct {
	service *service.ReviewSkillService
}

func NewReviewSkillHandler(service *service.ReviewSkillService) *ReviewSkillHandler {
	return &ReviewSkillHandler{service: service}
}

type reviewSkillResp struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	Enabled     bool   `json:"enabled"`
	BuiltIn     bool   `json:"built_in"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`

	// Skill Registry 扩展字段
	AgentXML          string `json:"agent_xml"`
	SkillRegistryXML  string `json:"skill_registry_xml"`
	ToolRegistryXML   string `json:"tool_registry_xml"`
	PolicyMD          string `json:"policy_md"`
	WorkflowMD        string `json:"workflow_md"`
	ContextSchemaJSON string `json:"context_schema_json"`
	MemorySchemaJSON  string `json:"memory_schema_json"`
	MetadataJSON      string `json:"metadata_json"`
}

type reviewSkillRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`

	// Skill Registry 扩展字段
	AgentXML          string `json:"agent_xml"`
	SkillRegistryXML  string `json:"skill_registry_xml"`
	ToolRegistryXML   string `json:"tool_registry_xml"`
	PolicyMD          string `json:"policy_md"`
	WorkflowMD        string `json:"workflow_md"`
	ContextSchemaJSON string `json:"context_schema_json"`
	MemorySchemaJSON  string `json:"memory_schema_json"`
	MetadataJSON      string `json:"metadata_json"`
}

func toReviewSkillResp(skill model.ReviewSkill) reviewSkillResp {
	return reviewSkillResp{
		ID:                skill.ID,
		Name:              skill.Name,
		Description:       skill.Description,
		Prompt:            skill.Prompt,
		Enabled:           skill.Enabled,
		BuiltIn:           skill.BuiltIn,
		SortOrder:         skill.SortOrder,
		CreatedAt:         skill.CreatedAt.Format("2006-01-02 15:04"),
		UpdatedAt:         skill.UpdatedAt.Format("2006-01-02 15:04"),
		AgentXML:          skill.AgentXML,
		SkillRegistryXML:  skill.SkillRegistryXML,
		ToolRegistryXML:   skill.ToolRegistryXML,
		PolicyMD:          skill.PolicyMD,
		WorkflowMD:        skill.WorkflowMD,
		ContextSchemaJSON: skill.ContextSchemaJSON,
		MemorySchemaJSON:  skill.MemorySchemaJSON,
		MetadataJSON:      skill.MetadataJSON,
	}
}

func (h *ReviewSkillHandler) APIList(c *gin.Context) {
	skills, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	out := make([]reviewSkillResp, 0, len(skills))
	for _, skill := range skills {
		out = append(out, toReviewSkillResp(skill))
	}
	c.JSON(http.StatusOK, out)
}

func (h *ReviewSkillHandler) APIGet(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	skill, err := h.service.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toReviewSkillResp(*skill))
}

func (h *ReviewSkillHandler) APICreate(c *gin.Context) {
	var req reviewSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	skill, err := h.service.Create(service.ReviewSkillInput{
		Name:              req.Name,
		Description:       req.Description,
		Prompt:            req.Prompt,
		Enabled:           req.Enabled,
		SortOrder:         req.SortOrder,
		AgentXML:          req.AgentXML,
		SkillRegistryXML:  req.SkillRegistryXML,
		ToolRegistryXML:   req.ToolRegistryXML,
		PolicyMD:          req.PolicyMD,
		WorkflowMD:        req.WorkflowMD,
		ContextSchemaJSON: req.ContextSchemaJSON,
		MemorySchemaJSON:  req.MemorySchemaJSON,
		MetadataJSON:      req.MetadataJSON,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, toReviewSkillResp(*skill))
}

func (h *ReviewSkillHandler) APIUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req reviewSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	skill, err := h.service.Update(id, service.ReviewSkillInput{
		Name:              req.Name,
		Description:       req.Description,
		Prompt:            req.Prompt,
		Enabled:           req.Enabled,
		SortOrder:         req.SortOrder,
		AgentXML:          req.AgentXML,
		SkillRegistryXML:  req.SkillRegistryXML,
		ToolRegistryXML:   req.ToolRegistryXML,
		PolicyMD:          req.PolicyMD,
		WorkflowMD:        req.WorkflowMD,
		ContextSchemaJSON: req.ContextSchemaJSON,
		MemorySchemaJSON:  req.MemorySchemaJSON,
		MetadataJSON:      req.MetadataJSON,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toReviewSkillResp(*skill))
}

func (h *ReviewSkillHandler) APIDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *ReviewSkillHandler) APIToggle(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	skill, err := h.service.SetEnabled(id, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toReviewSkillResp(*skill))
}
