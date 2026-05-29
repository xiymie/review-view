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

func (h *DashboardHandler) Index(c *gin.Context) {
	data, err := h.service.Build()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "dashboard/index", gin.H{
		"Title":        "仪表盘",
		"Active":       "dashboard",
		"PageTemplate": "dashboard/index_content",
		"DateLabel":    time.Now().Format("2006-01-02"),
		"Stats":        data.Stats,
		"RecentTasks":  data.RecentTasks,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"project_count":         data.Stats.ProjectCount,
			"running_count":         data.Stats.RunningCount,
			"today_completed_count": data.Stats.TodayCompletedCount,
			"failed_count":          data.Stats.FailedCount,
		},
		"recent_tasks": tasks,
		"date":         time.Now().Format("2006-01-02"),
	})
}
