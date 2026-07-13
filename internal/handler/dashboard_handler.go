package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"review-view/internal/service"
	"review-view/internal/store"
)

type DashboardHandler struct {
	service *service.DashboardService
	users   store.UserStore
}

func NewDashboardHandler(svc *service.DashboardService, users store.UserStore) *DashboardHandler {
	return &DashboardHandler{service: svc, users: users}
}

func (h *DashboardHandler) API(c *gin.Context) {
	data, err := h.service.BuildForUser(callerUID(c), isAdmin(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	type taskItem struct {
		ID            int64  `json:"id"`
		ProjectID     int64  `json:"project_id"`
		ProjectName   string `json:"project_name"`
		OwnerUsername string `json:"owner_username"`
		FromCommit    string `json:"from_commit"`
		ToCommit      string `json:"to_commit"`
		Status        string `json:"status"`
		TriggeredBy   string `json:"triggered_by"`
		CreatedAt     string `json:"created_at"`
	}

	ownerIDSet := make(map[int64]struct{}, len(data.RecentTasks))
	for _, t := range data.RecentTasks {
		if t.OwnerUserID != 0 {
			ownerIDSet[t.OwnerUserID] = struct{}{}
		}
	}
	ownerIDs := make([]int64, 0, len(ownerIDSet))
	for id := range ownerIDSet {
		ownerIDs = append(ownerIDs, id)
	}
	usernames := buildUsernameMap(h.users, ownerIDs)

	tasks := make([]taskItem, 0, len(data.RecentTasks))
	for _, t := range data.RecentTasks {
		tasks = append(tasks, taskItem{
			ID:            t.Task.ID,
			ProjectID:     t.Task.ProjectID,
			ProjectName:   t.ProjectName,
			OwnerUsername: usernames[t.OwnerUserID],
			FromCommit:    t.Task.FromCommit,
			ToCommit:      t.Task.ToCommit,
			Status:        string(t.Task.Status),
			TriggeredBy:   string(t.Task.TriggeredBy),
			CreatedAt:     t.Task.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	type activityItem struct {
		Kind      string `json:"kind"`
		ID        int64  `json:"id"`
		Title     string `json:"title"`
		SubTitle  string `json:"sub_title"`
		Status    string `json:"status"`
		HasRisk   bool   `json:"has_risk"`
		RiskCount int    `json:"risk_count"`
		TimeAgo   string `json:"time_ago"`
		Time      string `json:"time"`
	}

	acts := make([]activityItem, 0, len(data.Activities))
	for _, a := range data.Activities {
		acts = append(acts, activityItem{
			Kind:      a.Kind,
			ID:        a.ID,
			Title:     a.Title,
			SubTitle:  a.SubTitle,
			Status:    a.Status,
			HasRisk:   a.HasRisk,
			RiskCount: a.RiskCount,
			TimeAgo:   a.TimeAgo,
			Time:      a.Time.Format("2006-01-02 15:04"),
		})
	}

	type heatmapDay struct {
		Date      string `json:"date"`
		Completed int    `json:"completed"`
		Failed    int    `json:"failed"`
	}
	heatmap := make([]heatmapDay, 0, len(data.Heatmap))
	for _, h2 := range data.Heatmap {
		heatmap = append(heatmap, heatmapDay{
			Date:      h2.Date,
			Completed: h2.Completed,
			Failed:    h2.Failed,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"project_count":         data.Stats.ProjectCount,
			"running_count":         data.Stats.RunningCount,
			"today_completed_count": data.Stats.TodayCompletedCount,
			"failed_count":          data.Stats.FailedCount,
			"week_failed_count":     data.Stats.WeekFailedCount,
			"week_risk_count":       data.Stats.WeekRiskCount,
			"model_count":           data.Stats.ModelCount,
			"credential_count":      data.Stats.CredentialCount,
			"scan_enabled_count":    data.Stats.ScanEnabledCount,
			"user_count":            data.Stats.UserCount,
			"sensitive_word_count":  data.Stats.SensitiveWordCount,
			"scan_total_count":      data.Stats.ScanTotalCount,
		},
		"recent_tasks": tasks,
		"activities":   acts,
		"heatmap":      heatmap,
		"date":         time.Now().Format("2006-01-02"),
	})
}
