package service

import (
	"time"

	"review-view/internal/model"
	"review-view/internal/store"
)

type DashboardStats struct {
	ProjectCount        int
	RunningCount        int
	TodayCompletedCount int
	FailedCount         int
}

type DashboardTaskItem struct {
	Task        model.Task
	ProjectName string
	OwnerUserID int64
}

type DashboardData struct {
	Stats       DashboardStats
	RecentTasks []DashboardTaskItem
}

type DashboardService struct {
	projects store.ProjectStore
	tasks    store.TaskStore
}

func NewDashboardService(projects store.ProjectStore, tasks store.TaskStore) *DashboardService {
	return &DashboardService{projects: projects, tasks: tasks}
}

func (s *DashboardService) Build() (*DashboardData, error) {
	return s.BuildForUser(0, true)
}

func (s *DashboardService) BuildForUser(userID int64, admin bool) (*DashboardData, error) {
	var projects []model.Project
	var err error
	if admin {
		projects, err = s.projects.List()
	} else {
		projects, err = s.projects.ListByUser(userID)
	}
	if err != nil {
		return nil, err
	}

	projectIDs := make([]int64, 0, len(projects))
	projectNames := make(map[int64]string, len(projects))
	projectOwners := make(map[int64]int64, len(projects))
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
		projectNames[p.ID] = p.Name
		projectOwners[p.ID] = p.CreatedBy
	}

	var tasks []model.Task
	if admin {
		tasks, err = s.tasks.ListRecent(20)
	} else {
		tasks, err = s.tasks.ListByProjectIDs(projectIDs, 20)
	}
	if err != nil {
		return nil, err
	}

	stats := DashboardStats{ProjectCount: len(projects)}
	items := make([]DashboardTaskItem, 0, len(tasks))
	today := time.Now().Format("2006-01-02")
	for _, task := range tasks {
		switch task.Status {
		case model.TaskStatusRunning:
			stats.RunningCount++
		case model.TaskStatusCompleted:
			if task.CreatedAt.Format("2006-01-02") == today {
				stats.TodayCompletedCount++
			}
		case model.TaskStatusFailed:
			stats.FailedCount++
		}
		items = append(items, DashboardTaskItem{
			Task:        task,
			ProjectName: projectNames[task.ProjectID],
			OwnerUserID: projectOwners[task.ProjectID],
		})
	}

	return &DashboardData{
		Stats:       stats,
		RecentTasks: items,
	}, nil
}
