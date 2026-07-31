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
	// 新增
	WeekFailedCount  int
	WeekRiskCount    int
	ModelCount       int
	CredentialCount  int
	ScanEnabledCount int
	// 仪表盘顶部新增
	UserCount          int
	SensitiveWordCount int
	ScanTotalCount     int // 所有巡检配置数量（非仅启用）
}

type DashboardTaskItem struct {
	Task        model.Task
	ProjectName string
	OwnerUserID int64
}

// ActivityItem 混合时间轴条目
type ActivityItem struct {
	Kind      string // "task" | "scan"
	ID        int64
	Title     string // 项目名 or 巡检名
	SubTitle  string // commit range or "N分支有改动"
	Status    string
	HasRisk   bool
	RiskCount int
	Time      time.Time
	TimeAgo   string
}

// HeatmapDay 热力图单天数据
type HeatmapDay struct {
	Date      string
	Completed int
	Failed    int
}

type DashboardData struct {
	Stats       DashboardStats
	RecentTasks []DashboardTaskItem
	Activities  []ActivityItem
	Heatmap     []HeatmapDay
}

type DashboardService struct {
	projects       store.ProjectStore
	tasks          store.TaskStore
	models         store.ModelConfigStore
	creds          store.RepoCredentialStore
	scanScheds     store.ScanScheduleStore
	scanJobs       store.ScanJobStore
	users          store.UserStore
	sensitiveWords store.SensitiveWordStore
}

func NewDashboardService(
	projects store.ProjectStore,
	tasks store.TaskStore,
	models store.ModelConfigStore,
	creds store.RepoCredentialStore,
	scanScheds store.ScanScheduleStore,
	scanJobs store.ScanJobStore,
	users store.UserStore,
	sensitiveWords store.SensitiveWordStore,
) *DashboardService {
	return &DashboardService{
		projects:       projects,
		tasks:          tasks,
		models:         models,
		creds:          creds,
		scanScheds:     scanScheds,
		scanJobs:       scanJobs,
		users:          users,
		sensitiveWords: sensitiveWords,
	}
}

func (s *DashboardService) Build() (*DashboardData, error) {
	return s.BuildForUser(0, true)
}

func (s *DashboardService) BuildForUser(userID int64, admin bool) (*DashboardData, error) {
	now := time.Now()
	today := now.Format("2006-01-02")
	weekAgo := now.AddDate(0, 0, -7)

	// ---- 项目 ----
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

	// ---- 任务 ----
	var tasks []model.Task
	if admin {
		tasks, err = s.tasks.ListRecent(50)
	} else {
		tasks, err = s.tasks.ListByProjectIDs(projectIDs, 50)
	}
	if err != nil {
		return nil, err
	}

	stats := DashboardStats{ProjectCount: len(projects)}
	items := make([]DashboardTaskItem, 0, len(tasks))
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
			if task.CreatedAt.After(weekAgo) {
				stats.WeekFailedCount++
			}
		}
		items = append(items, DashboardTaskItem{
			Task:        task,
			ProjectName: projectNames[task.ProjectID],
			OwnerUserID: projectOwners[task.ProjectID],
		})
	}

	// ---- 资产统计 ----
	if modelList, err2 := s.models.List(); err2 == nil {
		stats.ModelCount = len(modelList)
	}
	if credList, err2 := s.creds.List(); err2 == nil {
		stats.CredentialCount = len(credList)
	}
	if scanList, err2 := s.scanScheds.ListEnabled(); err2 == nil {
		stats.ScanEnabledCount = len(scanList)
	}
	if allScanList, err2 := s.scanScheds.List(); err2 == nil {
		stats.ScanTotalCount = len(allScanList)
	}
	if userCount, err2 := s.users.Count(); err2 == nil {
		stats.UserCount = int(userCount)
	}
	if swList, err2 := s.sensitiveWords.List(); err2 == nil {
		stats.SensitiveWordCount = len(swList)
	}

	// ---- 巡检高风险（本周）----
	if allScheds, err2 := s.scanScheds.List(); err2 == nil {
		for _, sched := range allScheds {
			if recentJobs, err3 := s.scanJobs.ListBySchedule(sched.ID, 10); err3 == nil {
				for _, job := range recentJobs {
					if job.TriggeredAt.Before(weekAgo) {
						continue
					}
					stats.WeekRiskCount += job.RiskCount
				}
			}
		}
	}

	// ---- 混合活动时间轴 ----
	activities := buildActivities(items, s.scanScheds, s.scanJobs, projectNames, weekAgo)

	// ---- 7日热力图 ----
	heatmap := buildHeatmap(tasks, now)

	return &DashboardData{
		Stats:       stats,
		RecentTasks: items,
		Activities:  activities,
		Heatmap:     heatmap,
	}, nil
}

// buildActivities 合并 Review 任务和巡检执行，按时间倒序取最近15条
func buildActivities(
	taskItems []DashboardTaskItem,
	scanScheds store.ScanScheduleStore,
	scanJobs store.ScanJobStore,
	projectNames map[int64]string,
	since time.Time,
) []ActivityItem {
	var activities []ActivityItem

	// Review 任务
	for _, t := range taskItems {
		if t.Task.CreatedAt.Before(since) {
			continue
		}
		sub := t.Task.ToCommit
		if len(sub) > 7 {
			sub = sub[:7]
		}
		if t.Task.FromCommit != "" {
			from := t.Task.FromCommit
			if len(from) > 7 {
				from = from[:7]
			}
			sub = from + ".." + sub
		}
		activities = append(activities, ActivityItem{
			Kind:     "task",
			ID:       t.Task.ID,
			Title:    t.ProjectName,
			SubTitle: sub,
			Status:   string(t.Task.Status),
			Time:     t.Task.CreatedAt,
			TimeAgo:  timeAgo(t.Task.CreatedAt),
		})
	}

	// 巡检执行
	if scheds, err := scanScheds.List(); err == nil {
		for _, sched := range scheds {
			jobs, err := scanJobs.ListBySchedule(sched.ID, 5)
			if err != nil {
				continue
			}
			for _, job := range jobs {
				if job.TriggeredAt.Before(since) {
					continue
				}
				riskCount := job.RiskCount
				sub := "无改动分支"
				if job.ChangedBranchCount > 0 {
					sub = formatBranchCount(job.ChangedBranchCount, job.BranchCount)
				}
				activities = append(activities, ActivityItem{
					Kind:      "scan",
					ID:        job.ID,
					Title:     sched.Name,
					SubTitle:  sub,
					Status:    string(job.Status),
					HasRisk:   riskCount > 0,
					RiskCount: riskCount,
					Time:      job.TriggeredAt,
					TimeAgo:   timeAgo(job.TriggeredAt),
				})
			}
		}
	}

	// 按时间倒序
	for i := 0; i < len(activities)-1; i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[j].Time.After(activities[i].Time) {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}
	if len(activities) > 15 {
		activities = activities[:15]
	}
	return activities
}

func buildHeatmap(tasks []model.Task, now time.Time) []HeatmapDay {
	days := make([]HeatmapDay, 7)
	for i := 6; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		days[6-i] = HeatmapDay{Date: d.Format("2006-01-02")}
	}
	idx := make(map[string]int, 7)
	for i, d := range days {
		idx[d.Date] = i
	}
	for _, t := range tasks {
		date := t.CreatedAt.Format("2006-01-02")
		if i, ok := idx[date]; ok {
			if t.Status == model.TaskStatusCompleted {
				days[i].Completed++
			} else if t.Status == model.TaskStatusFailed {
				days[i].Failed++
			}
		}
	}
	return days
}

func timeAgo(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return formatInt(int(diff.Minutes())) + " 分钟前"
	case diff < 24*time.Hour:
		return formatInt(int(diff.Hours())) + " 小时前"
	default:
		return formatInt(int(diff.Hours()/24)) + " 天前"
	}
}

func formatInt(n int) string {
	if n <= 0 {
		return "1"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func formatBranchCount(changed, total int) string {
	return formatInt(changed) + "/" + formatInt(total) + " 分支有改动"
}
