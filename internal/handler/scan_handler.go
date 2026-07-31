package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
	"review-view/internal/store"
)

type ScanHandler struct {
	svc      *service.ScanService
	settings *service.SettingsService
	models   store.ModelConfigStore
	creds    store.RepoCredentialStore
}

func NewScanHandler(svc *service.ScanService, settings *service.SettingsService, models store.ModelConfigStore, creds store.RepoCredentialStore) *ScanHandler {
	return &ScanHandler{svc: svc, settings: settings, models: models, creds: creds}
}

// scheduleResp 统一小写 JSON 响应结构
type scheduleResp struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	RepoURL          string `json:"repo_url"`
	RepoCredentialID *int64 `json:"repo_credential_id"`
	ModelConfigID    int64  `json:"model_config_id"`
	ScanTime         string `json:"scan_time"`
	CustomPrompt     string `json:"custom_prompt"`
	NasURL           string `json:"nas_url"`
	NasUsername      string `json:"nas_username"`
	NasSubDir        string `json:"nas_sub_dir"`
	Enabled          bool   `json:"enabled"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func toScheduleResp(v *model.ScanSchedule) scheduleResp {
	return scheduleResp{
		ID:               v.ID,
		Name:             v.Name,
		RepoURL:          v.RepoURL,
		RepoCredentialID: v.RepoCredentialID,
		ModelConfigID:    v.ModelConfigID,
		ScanTime:         v.ScanTime,
		CustomPrompt:     v.CustomPrompt,
		NasURL:           v.NasURL,
		NasUsername:      v.NasUsername,
		NasSubDir:        v.NasSubDir,
		Enabled:          v.Enabled,
		CreatedAt:        v.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:        v.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// jobResp 巡检执行记录
type jobResp struct {
	ID                 int64  `json:"id"`
	ScheduleID         int64  `json:"schedule_id"`
	Status             string `json:"status"`
	ErrorMessage       string `json:"error_message"`
	BranchCount        int    `json:"branch_count"`
	ChangedBranchCount int    `json:"changed_branch_count"`
	RiskCount          int    `json:"risk_count"`
	ReportPath         string `json:"report_path"`
	TriggeredAt        string `json:"triggered_at"`
	FinishedAt         string `json:"finished_at"`
}

func toJobResp(j model.ScanJob) jobResp {
	r := jobResp{
		ID:                 j.ID,
		ScheduleID:         j.ScheduleID,
		Status:             string(j.Status),
		ErrorMessage:       j.ErrorMessage,
		BranchCount:        j.BranchCount,
		ChangedBranchCount: j.ChangedBranchCount,
		RiskCount:          j.RiskCount,
		ReportPath:         j.ReportPath,
		TriggeredAt:        j.TriggeredAt.Format("2006-01-02 15:04:05"),
	}
	if j.FinishedAt != nil {
		r.FinishedAt = j.FinishedAt.Format("2006-01-02 15:04:05")
	}
	return r
}

// jobLogResp 巡检执行日志
type jobLogResp struct {
	ID        int64  `json:"id"`
	JobID     int64  `json:"job_id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

func toJobLogResp(log model.ScanJobLog) jobLogResp {
	return jobLogResp{
		ID:        log.ID,
		JobID:     log.JobID,
		Level:     string(log.Level),
		Message:   log.Message,
		CreatedAt: log.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// branchResultResp 分支分析结果
type branchResultResp struct {
	ID             int64  `json:"id"`
	JobID          int64  `json:"job_id"`
	BranchName     string `json:"branch_name"`
	CommitCount    int    `json:"commit_count"`
	CommitMessages string `json:"commit_messages"`
	FromCommit     string `json:"from_commit"`
	ToCommit       string `json:"to_commit"`
	AnalysisStage  string `json:"analysis_stage"`
	Result         string `json:"result"`
	HasRisk        bool   `json:"has_risk"`
	RiskLevel      string `json:"risk_level"`
	CreatedAt      string `json:"created_at"`
}

func toBranchResultResp(r model.ScanBranchResult) branchResultResp {
	return branchResultResp{
		ID:             r.ID,
		JobID:          r.JobID,
		BranchName:     r.BranchName,
		CommitCount:    r.CommitCount,
		CommitMessages: r.CommitMessages,
		FromCommit:     r.FromCommit,
		ToCommit:       r.ToCommit,
		AnalysisStage:  string(r.AnalysisStage),
		Result:         r.Result,
		HasRisk:        r.HasRisk,
		RiskLevel:      r.RiskLevel,
		CreatedAt:      r.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func validateScanTime(scanTime string) error {
	if strings.TrimSpace(scanTime) == "" {
		return nil
	}
	if _, err := time.Parse("15:04", scanTime); err != nil {
		return fmt.Errorf("scan_time 格式应为 HH:MM")
	}
	return nil
}

func validateNASURL(raw string) error {
	return service.ValidateNASURL(raw)
}

// APIList GET /api/scan-schedules
func (h *ScanHandler) APIList(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]scheduleResp, 0, len(list))
	for i := range list {
		out = append(out, toScheduleResp(&list[i]))
	}
	c.JSON(http.StatusOK, out)
}

// APIGet GET /api/scan-schedules/:id
func (h *ScanHandler) APIGet(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	v, err := h.svc.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, toScheduleResp(v))
}

// APIToggleEnabled PATCH /api/scan-schedules/:id/toggle
func (h *ScanHandler) APIToggleEnabled(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	v, err := h.svc.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	v.Enabled = !v.Enabled
	if err := h.svc.Update(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": v.Enabled})
}

// APICreate POST /api/scan-schedules
func (h *ScanHandler) APICreate(c *gin.Context) {
	var req struct {
		Name             string `json:"name"`
		RepoURL          string `json:"repo_url"`
		RepoCredentialID *int64 `json:"repo_credential_id"`
		ModelConfigID    int64  `json:"model_config_id"`
		ScanTime         string `json:"scan_time"`
		CustomPrompt     string `json:"custom_prompt"`
		NasURL           string `json:"nas_url"`
		NasUsername      string `json:"nas_username"`
		NasPassword      string `json:"nas_password"`
		NasSubDir        string `json:"nas_sub_dir"`
		Enabled          bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateScanTime(req.ScanTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateNASURL(req.NasURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v := &model.ScanSchedule{
		Name:             req.Name,
		RepoURL:          req.RepoURL,
		RepoCredentialID: req.RepoCredentialID,
		ModelConfigID:    req.ModelConfigID,
		ScanTime:         req.ScanTime,
		CustomPrompt:     req.CustomPrompt,
		NasURL:           req.NasURL,
		NasUsername:      req.NasUsername,
		NasPassword:      req.NasPassword,
		NasSubDir:        req.NasSubDir,
		Enabled:          req.Enabled,
	}
	if err := h.svc.Create(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toScheduleResp(v))
}

// APIUpdate PUT /api/scan-schedules/:id
func (h *ScanHandler) APIUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	existing, err := h.svc.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var req struct {
		Name             string `json:"name"`
		RepoURL          string `json:"repo_url"`
		RepoCredentialID *int64 `json:"repo_credential_id"`
		ModelConfigID    int64  `json:"model_config_id"`
		ScanTime         string `json:"scan_time"`
		CustomPrompt     string `json:"custom_prompt"`
		NasURL           string `json:"nas_url"`
		NasUsername      string `json:"nas_username"`
		NasPassword      string `json:"nas_password"`
		NasSubDir        string `json:"nas_sub_dir"`
		Enabled          bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateScanTime(req.ScanTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateNASURL(req.NasURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.Name = req.Name
	existing.RepoURL = req.RepoURL
	existing.RepoCredentialID = req.RepoCredentialID
	existing.ModelConfigID = req.ModelConfigID
	existing.ScanTime = req.ScanTime
	existing.CustomPrompt = req.CustomPrompt
	existing.NasURL = req.NasURL
	existing.NasUsername = req.NasUsername
	if req.NasPassword != "" {
		existing.NasPassword = req.NasPassword
	}
	existing.NasSubDir = req.NasSubDir
	existing.Enabled = req.Enabled
	if err := h.svc.Update(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toScheduleResp(existing))
}

// APIDelete DELETE /api/scan-schedules/:id
func (h *ScanHandler) APIDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// APIListJobs GET /api/scan-schedules/:id/jobs
func (h *ScanHandler) APIListJobs(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jobs, err := h.svc.ListJobs(id, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]jobResp, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toJobResp(j))
	}
	c.JSON(http.StatusOK, out)
}

// APIDeleteJob DELETE /api/scan-jobs/:id
func (h *ScanHandler) APIDeleteJob(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	job, err := h.svc.GetJob(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if job.Status == model.ScanJobStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "运行中的巡检不能删除"})
		return
	}
	if err := h.svc.DeleteJob(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// APIGetJob GET /api/scan-jobs/:id
func (h *ScanHandler) APIGetJob(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	job, err := h.svc.GetJob(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	results, _ := h.svc.ListBranchResults(id)
	branchOut := make([]branchResultResp, 0, len(results))
	for _, r := range results {
		branchOut = append(branchOut, toBranchResultResp(r))
	}
	logs, _ := h.svc.ListJobLogs(id)
	logOut := make([]jobLogResp, 0, len(logs))
	for _, log := range logs {
		logOut = append(logOut, toJobLogResp(log))
	}
	c.JSON(http.StatusOK, gin.H{
		"job":     toJobResp(*job),
		"results": branchOut,
		"logs":    logOut,
	})
}

// APITrigger POST /api/scan-schedules/:id/trigger
func (h *ScanHandler) APITrigger(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.TriggerSchedule(context.Background(), id); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "巡检任务已启动"})
}

// APITestNas POST /api/scan/test-nas
func (h *ScanHandler) APITestNas(c *gin.Context) {
	var req struct {
		NasURL      string `json:"nas_url"`
		NasUsername string `json:"nas_username"`
		NasPassword string `json:"nas_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	if req.NasPassword == "" {
		req.NasPassword, _ = h.settings.GetRaw(model.GlobalConfigKeyScanNasPassword)
	}
	nasURL := req.NasURL
	if nasURL == "" {
		nasURL, _ = h.settings.GetRaw(model.GlobalConfigKeyScanNasURL)
	}
	if nasURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "NAS 地址不能为空"})
		return
	}
	if err := validateNASURL(nasURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	propfindReq, err := http.NewRequest("PROPFIND", strings.TrimRight(nasURL, "/")+"/", nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": fmt.Sprintf("构建请求失败: %v", err)})
		return
	}
	propfindReq.SetBasicAuth(req.NasUsername, req.NasPassword)
	propfindReq.Header.Set("Depth", "0")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(propfindReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": fmt.Sprintf("连接失败: %v", err)})
		return
	}
	defer func() { _, _ = io.ReadAll(resp.Body); _ = resp.Body.Close() }()
	if resp.StatusCode == 207 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "连接成功"})
	} else if resp.StatusCode == 401 {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "认证失败，请检查用户名和密码"})
	} else {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": fmt.Sprintf("服务器返回 %d", resp.StatusCode)})
	}
}

// APIModels GET /api/scan/models
func (h *ScanHandler) APIModels(c *gin.Context) {
	list, err := h.models.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type item struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	out := make([]item, 0, len(list))
	for _, m := range list {
		out = append(out, item{ID: m.ID, Name: m.Name, Type: string(m.Type)})
	}
	c.JSON(http.StatusOK, out)
}

// APICredentials GET /api/scan/credentials
func (h *ScanHandler) APICredentials(c *gin.Context) {
	list, err := h.creds.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type item struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	out := make([]item, 0, len(list))
	for _, cr := range list {
		out = append(out, item{ID: cr.ID, Name: cr.Name})
	}
	c.JSON(http.StatusOK, out)
}
