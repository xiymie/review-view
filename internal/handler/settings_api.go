package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/notify"
	"review-view/internal/service"
)

func (h *SettingsHandler) APIGet(c *gin.Context) {
	settings, err := h.service.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	smtpHost, _ := h.service.GetRaw(model.GlobalConfigKeySMTPHost)
	smtpPort, _ := h.service.GetRaw(model.GlobalConfigKeySMTPPort)
	smtpUsername, _ := h.service.GetRaw(model.GlobalConfigKeySMTPUsername)
	smtpFrom, _ := h.service.GetRaw(model.GlobalConfigKeySMTPFrom)
	smtpFromName, _ := h.service.GetRaw(model.GlobalConfigKeySMTPFromName)
	smtpTLS, _ := h.service.GetRaw(model.GlobalConfigKeySMTPTLS)

	c.JSON(http.StatusOK, gin.H{
		"max_concurrent_tasks":        settings.MaxConcurrentTasks,
		"overflow_strategy":           string(settings.OverflowStrategy),
		"task_timeout":                settings.TaskTimeout,
		"repo_base_dir":               settings.RepoBaseDir,
		"scheduled_scan_unchanged":    settings.ScheduledScanUnchanged,
		"manual_scan_unchanged":       settings.ManualScanUnchanged,
		"smtp_host":                   smtpHost,
		"smtp_port":                   smtpPort,
		"smtp_username":               smtpUsername,
		"smtp_from":                   smtpFrom,
		"smtp_from_name":              smtpFromName,
		"smtp_tls":                    smtpTLS,
	})
}

func (h *SettingsHandler) APIUpdate(c *gin.Context) {
	var req struct {
		MaxConcurrentTasks     int    `json:"max_concurrent_tasks"`
		OverflowStrategy       string `json:"overflow_strategy"`
		TaskTimeout            int    `json:"task_timeout"`
		RepoBaseDir            string `json:"repo_base_dir"`
		ScheduledScanUnchanged bool   `json:"scheduled_scan_unchanged"`
		ManualScanUnchanged    bool   `json:"manual_scan_unchanged"`
		SMTPHost               string `json:"smtp_host"`
		SMTPPort               string `json:"smtp_port"`
		SMTPUsername           string `json:"smtp_username"`
		SMTPPassword           string `json:"smtp_password"`
		SMTPFrom               string `json:"smtp_from"`
		SMTPFromName           string `json:"smtp_from_name"`
		SMTPTLS                string `json:"smtp_tls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	err := h.service.Update(service.SettingsInput{
		MaxConcurrentTasks:     req.MaxConcurrentTasks,
		OverflowStrategy:       model.OverflowStrategy(req.OverflowStrategy),
		TaskTimeout:            req.TaskTimeout,
		RepoBaseDir:            req.RepoBaseDir,
		ScheduledScanUnchanged: req.ScheduledScanUnchanged,
		ManualScanUnchanged:    req.ManualScanUnchanged,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if err := h.service.SetSMTP(req.SMTPHost, req.SMTPPort, req.SMTPUsername, req.SMTPPassword, req.SMTPFrom, req.SMTPFromName, req.SMTPTLS); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *SettingsHandler) APITestEmail(c *gin.Context) {
	var req struct {
		To           string `json:"to"`
		SMTPHost     string `json:"smtp_host"`
		SMTPPort     string `json:"smtp_port"`
		SMTPUsername string `json:"smtp_username"`
		SMTPPassword string `json:"smtp_password"`
		SMTPFrom     string `json:"smtp_from"`
		SMTPFromName string `json:"smtp_from_name"`
		SMTPTLS      string `json:"smtp_tls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "收件地址不能为空"})
		return
	}

	// 密码留空时，从数据库读取已保存的密码
	password := req.SMTPPassword
	if password == "" {
		password, _ = h.service.GetRaw(model.GlobalConfigKeySMTPPassword)
	}

	cfg := notify.ParseSMTPConfig(req.SMTPHost, req.SMTPPort, req.SMTPUsername, password, req.SMTPFrom, req.SMTPFromName, req.SMTPTLS)
	if err := notify.SendTestEmail(cfg, req.To); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": "发送失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试邮件已发送，请检查收件箱"})
}
