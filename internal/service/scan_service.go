package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"review-view/internal/model"
	"review-view/internal/review"
	"review-view/internal/store"
)

const defaultScanPrompt = model.DefaultScanPrompt

// ScanConfig 一次巡检所需的完整配置（由 schedule + 全局配置合并而来）
type ScanConfig struct {
	RepoURL     string
	Cred        *model.RepoCredential
	ModelConfig *model.ModelConfig
	SkillPrompt string
	Prompt      string
	NasURL      string
	NasUsername string
	NasPassword string
	NasSubDir   string
}

type scanLLMCaller func(ctx context.Context, mc *model.ModelConfig, prompt string) (string, error)
type scanDiffLoader func(ctx context.Context, repoDir, branch, fromCommit string) (string, error)
type scanRepoPreparer func(ctx context.Context, repoDir string, cfg *ScanConfig) error
type scanBranchLister func(ctx context.Context, repoDir string) ([]string, error)
type scanBranchHeadGetter func(ctx context.Context, repoDir, branch string) (string, error)
type scanRecentCommitsGetter func(ctx context.Context, repoDir, branch string, n int) ([]commitEntry, error)
type scanCommitsBetweenGetter func(ctx context.Context, repoDir, branch, fromCommit, headCommit string) ([]commitEntry, error)
type scanReportUploader func(cfg *ScanConfig, name, content string) (string, error)
type scanOldReportsCleaner func(ctx context.Context, scheduleID int64, cfg *ScanConfig)

type ScanService struct {
	schedules            store.ScanScheduleStore
	jobs                 store.ScanJobStore
	models               store.ModelConfigStore
	creds                store.RepoCredentialStore
	settings             *SettingsService
	repoMgr              *review.RepositoryManager
	projectStore         store.ProjectStore
	reviewSkills         *ReviewSkillService
	sensitiveWords       *SensitiveWordService
	llmCaller            scanLLMCaller
	diffLoader           scanDiffLoader
	repoPreparer         scanRepoPreparer
	branchLister         scanBranchLister
	branchHeadGetter     scanBranchHeadGetter
	recentCommitsGetter  scanRecentCommitsGetter
	commitsBetweenGetter scanCommitsBetweenGetter
	reportUploader       scanReportUploader
	oldReportsCleaner    scanOldReportsCleaner
	runningMu            sync.Mutex
	runningSchedules     map[int64]struct{}
}

func NewScanService(
	schedules store.ScanScheduleStore,
	jobs store.ScanJobStore,
	models store.ModelConfigStore,
	creds store.RepoCredentialStore,
	settings *SettingsService,
	repoMgr *review.RepositoryManager,
	extra ...any,
) *ScanService {
	s := &ScanService{
		schedules:        schedules,
		jobs:             jobs,
		models:           models,
		creds:            creds,
		settings:         settings,
		repoMgr:          repoMgr,
		runningSchedules: make(map[int64]struct{}),
	}
	for _, item := range extra {
		switch v := item.(type) {
		case store.ProjectStore:
			s.projectStore = v
		case *ReviewSkillService:
			s.reviewSkills = v
		case *SensitiveWordService:
			s.sensitiveWords = v
		}
	}
	return s
}

// List 列出所有巡检配置
func (s *ScanService) List() ([]model.ScanSchedule, error) {
	return s.schedules.List()
}

// Get 获取单个巡检配置
func (s *ScanService) Get(id int64) (*model.ScanSchedule, error) {
	return s.schedules.GetByID(id)
}

// Create 新建巡检配置
func (s *ScanService) Create(v *model.ScanSchedule) error {
	return s.schedules.Create(v)
}

// Update 更新巡检配置
func (s *ScanService) Update(v *model.ScanSchedule) error {
	return s.schedules.Update(v)
}

// Delete 删除巡检配置
func (s *ScanService) Delete(id int64) error {
	running, err := s.jobs.HasRunningBySchedule(id)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("该巡检配置存在运行中的任务，请等待完成后再删除")
	}
	return s.schedules.Delete(id)
}

// ListJobs 获取某个配置的历史执行记录
func (s *ScanService) ListJobs(scheduleID int64, limit int) ([]model.ScanJob, error) {
	return s.jobs.ListBySchedule(scheduleID, limit)
}

func (s *ScanService) HasRunningJob(scheduleID int64) (bool, error) {
	return s.jobs.HasRunningBySchedule(scheduleID)
}

// DeleteJob 删除单次执行记录及其分支结果
func (s *ScanService) DeleteJob(jobID int64) error {
	return s.jobs.Delete(jobID)
}

// GetJob 获取单次执行记录
func (s *ScanService) GetJob(jobID int64) (*model.ScanJob, error) {
	return s.jobs.GetByID(jobID)
}

// ListBranchResults 获取单次执行的所有分支结果
func (s *ScanService) ListBranchResults(jobID int64) ([]model.ScanBranchResult, error) {
	return s.jobs.ListBranchResults(jobID)
}

// ListJobLogs 获取单次执行的节点/状态日志
func (s *ScanService) ListJobLogs(jobID int64) ([]model.ScanJobLog, error) {
	return s.jobs.ListJobLogs(jobID)
}

// RunSchedule 执行一次巡检（同步，供调度器调用）
func (s *ScanService) RunSchedule(ctx context.Context, scheduleID int64) error {
	if err := s.acquireScheduleRun(scheduleID); err != nil {
		return err
	}
	defer s.releaseScheduleRun(scheduleID)
	return s.runScheduleOnce(ctx, scheduleID)
}

// TriggerSchedule 异步触发一次巡检；在返回前完成并发占位，避免重复点击创建多个 goroutine。
func (s *ScanService) TriggerSchedule(ctx context.Context, scheduleID int64) error {
	if err := s.acquireScheduleRun(scheduleID); err != nil {
		return err
	}
	if err := s.ensureNoRunningJob(scheduleID); err != nil {
		s.releaseScheduleRun(scheduleID)
		return err
	}
	go func() {
		defer s.releaseScheduleRun(scheduleID)
		_ = newScanScheduleWorkflow(s).Run(ctx, scheduleID)
	}()
	return nil
}

func (s *ScanService) runScheduleOnce(ctx context.Context, scheduleID int64) error {
	if err := s.ensureNoRunningJob(scheduleID); err != nil {
		return err
	}
	return newScanScheduleWorkflow(s).Run(ctx, scheduleID)
}

func (s *ScanService) ensureNoRunningJob(scheduleID int64) error {
	running, err := s.jobs.HasRunningBySchedule(scheduleID)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("巡检配置 %d 已有运行中的任务", scheduleID)
	}
	return nil
}

func (s *ScanService) acquireScheduleRun(scheduleID int64) error {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	if s.runningSchedules == nil {
		s.runningSchedules = make(map[int64]struct{})
	}
	if _, ok := s.runningSchedules[scheduleID]; ok {
		return fmt.Errorf("巡检配置 %d 已有运行中的任务", scheduleID)
	}
	s.runningSchedules[scheduleID] = struct{}{}
	return nil
}

func (s *ScanService) releaseScheduleRun(scheduleID int64) {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	delete(s.runningSchedules, scheduleID)
}

// ---- internal helpers ----

func (s *ScanService) prepareScanRepo(ctx context.Context, repoDir string, cfg *ScanConfig) error {
	if s.repoPreparer != nil {
		return s.repoPreparer(ctx, repoDir, cfg)
	}
	return s.ensureRepo(ctx, repoDir, cfg)
}

func (s *ScanService) loadRemoteBranches(ctx context.Context, repoDir string) ([]string, error) {
	if s.branchLister != nil {
		return s.branchLister(ctx, repoDir)
	}
	return s.listRemoteBranches(ctx, repoDir)
}

func (s *ScanService) loadBranchHead(ctx context.Context, repoDir, branch string) (string, error) {
	if s.branchHeadGetter != nil {
		return s.branchHeadGetter(ctx, repoDir, branch)
	}
	return s.getBranchHead(ctx, repoDir, branch)
}

func (s *ScanService) loadRecentCommits(ctx context.Context, repoDir, branch string, n int) ([]commitEntry, error) {
	if s.recentCommitsGetter != nil {
		return s.recentCommitsGetter(ctx, repoDir, branch, n)
	}
	return s.getRecentCommits(ctx, repoDir, branch, n)
}

func (s *ScanService) loadCommitsBetween(ctx context.Context, repoDir, branch, fromCommit, headCommit string) ([]commitEntry, error) {
	if s.commitsBetweenGetter != nil {
		return s.commitsBetweenGetter(ctx, repoDir, branch, fromCommit, headCommit)
	}
	return s.getCommitsBetween(ctx, repoDir, branch, fromCommit, headCommit)
}

func (s *ScanService) uploadScanReport(cfg *ScanConfig, name, content string) (string, error) {
	if s.reportUploader != nil {
		return s.reportUploader(cfg, name, content)
	}
	return s.uploadToNAS(cfg, name, content)
}

func (s *ScanService) cleanOldScanReports(ctx context.Context, scheduleID int64, cfg *ScanConfig) {
	if s.oldReportsCleaner != nil {
		s.oldReportsCleaner(ctx, scheduleID, cfg)
		return
	}
	s.cleanupOldReports(ctx, scheduleID, cfg)
}

func (s *ScanService) buildConfig(sched *model.ScanSchedule) (*ScanConfig, error) {
	mc, err := s.models.GetByID(sched.ModelConfigID)
	if err != nil {
		return nil, fmt.Errorf("model config %d not found", sched.ModelConfigID)
	}

	var cred *model.RepoCredential
	if sched.RepoCredentialID != nil {
		cred, _ = s.creds.GetByID(*sched.RepoCredentialID)
	}

	// 合并全局配置
	globalNasURL, _ := s.settings.GetRaw(model.GlobalConfigKeyScanNasURL)
	globalNasUser, _ := s.settings.GetRaw(model.GlobalConfigKeyScanNasUsername)
	globalNasPass, _ := s.settings.GetRaw(model.GlobalConfigKeyScanNasPassword)
	globalPrompt, _ := s.settings.GetRaw(model.GlobalConfigKeyScanPrompt)

	nasURL := sched.NasURL
	if nasURL == "" {
		nasURL = globalNasURL
	}
	nasUser := sched.NasUsername
	if nasUser == "" {
		nasUser = globalNasUser
	}
	nasPass := sched.NasPassword
	if nasPass == "" {
		nasPass = globalNasPass
	}
	prompt := globalPrompt
	if strings.TrimSpace(prompt) == "" {
		prompt = defaultScanPrompt
	}
	skillPrompt, projectPrompt, err := s.buildProjectPromptContextByRepoURL(sched.RepoURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(projectPrompt) != "" {
		prompt = prompt + "\n\n## 项目自定义提示词\n\n" + strings.TrimSpace(projectPrompt)
	}
	if strings.TrimSpace(sched.CustomPrompt) != "" {
		prompt = prompt + "\n\n## 巡检配置自定义提示词\n\n" + strings.TrimSpace(sched.CustomPrompt)
	}

	if err := validateNASURL(nasURL); err != nil {
		return nil, err
	}

	nasSubDir := sched.NasSubDir
	if nasSubDir == "" {
		// 用仓库名作为子目录
		parts := strings.Split(strings.TrimSuffix(sched.RepoURL, ".git"), "/")
		if len(parts) > 0 {
			nasSubDir = parts[len(parts)-1]
		} else {
			nasSubDir = fmt.Sprintf("scan-%d", sched.ID)
		}
	}

	return &ScanConfig{
		RepoURL:     sched.RepoURL,
		Cred:        cred,
		ModelConfig: mc,
		SkillPrompt: skillPrompt,
		Prompt:      prompt,
		NasURL:      nasURL,
		NasUsername: nasUser,
		NasPassword: nasPass,
		NasSubDir:   nasSubDir,
	}, nil
}

func (s *ScanService) buildProjectPromptContextByRepoURL(repoURL string) (skillPrompt, projectPrompt string, err error) {
	if s.projectStore == nil || strings.TrimSpace(repoURL) == "" {
		return "", "", nil
	}
	projects, err := s.projectStore.List()
	if err != nil {
		return "", "", err
	}
	var matched *model.Project
	trimmed := strings.TrimSpace(repoURL)
	for i := range projects {
		if strings.TrimSpace(projects[i].RepoURL) == trimmed {
			matched = &projects[i]
			break
		}
	}
	if matched == nil {
		return "", "", nil
	}
	projectPrompt = matched.CustomPrompt
	if s.reviewSkills == nil {
		return "", projectPrompt, nil
	}
	skillIDs, err := s.projectStore.ListSkillIDs(matched.ID)
	if err != nil {
		return "", "", err
	}
	skillPrompt, err = s.reviewSkills.BuildPromptForSkillIDs(skillIDs)
	return skillPrompt, projectPrompt, err
}

func (s *ScanService) ensureRepo(ctx context.Context, repoDir string, cfg *ScanConfig) error {
	cloneURL := cfg.RepoURL
	if cfg.Cred != nil {
		cloneURL = injectCred(cfg.RepoURL, cfg.Cred.Username, cfg.Cred.Password)
	}
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			return err
		}
		_, err := runGit(ctx, "", "git", "clone", "--no-single-branch", cloneURL, repoDir)
		return err
	}
	// 已有：更新 remote URL 再 fetch all
	if cfg.Cred != nil {
		if _, err := runGit(ctx, repoDir, "git", "remote", "set-url", "origin", cloneURL); err != nil {
			return err
		}
	}
	_, err := runGit(ctx, repoDir, "git", "fetch", "--all", "--prune")
	return err
}

func (s *ScanService) listRemoteBranches(ctx context.Context, repoDir string) ([]string, error) {
	out, err := runGit(ctx, repoDir, "git", "branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "HEAD") {
			continue
		}
		// origin/main -> main
		if after, ok := strings.CutPrefix(line, "origin/"); ok {
			branches = append(branches, after)
		}
	}
	return branches, nil
}

// commitEntry 单条 commit 信息
type commitEntry struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Time    string `json:"time"`
}

func (s *ScanService) getCommitsSince(ctx context.Context, repoDir, branch, since string) ([]commitEntry, error) {
	out, err := runGit(ctx, repoDir,
		"git", "log",
		"--pretty=format:%H|%s|%an|%ai",
		"--after="+since,
		"origin/"+branch,
	)
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, err
	}
	var entries []commitEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		entries = append(entries, commitEntry{
			Hash:    parts[0][:min(8, len(parts[0]))],
			Message: parts[1],
			Author:  parts[2],
			Time:    parts[3],
		})
	}
	return entries, nil
}

func (s *ScanService) analyzeBranch(
	ctx context.Context,
	cfg *ScanConfig,
	repoDir, branch string,
	commits []commitEntry,
	repoURL string,
	jobID int64,
) (*model.ScanBranchResult, error) {
	graph := newScanBranchGraph(s)
	return graph.Run(ctx, scanBranchGraphInput{
		Config:  cfg,
		RepoDir: repoDir,
		Branch:  branch,
		Commits: commits,
		RepoURL: repoURL,
		JobID:   jobID,
	})
}

func (s *ScanService) loadBranchDiff(ctx context.Context, repoDir, branch, fromCommit string) (string, error) {
	if s.diffLoader != nil {
		return s.diffLoader(ctx, repoDir, branch, fromCommit)
	}
	rawDiff, err := runGit(ctx, repoDir, "git", "log", "-p",
		fmt.Sprintf("%s..origin/%s", fromCommit, branch))
	if err != nil || strings.TrimSpace(rawDiff) == "" {
		// fallback：取最近 24 小时内的 diff
		rawDiff, _ = runGit(ctx, repoDir, "git", "log", "-p",
			"--after="+time.Now().Add(-24*time.Hour).Format("2006-01-02"),
			"origin/"+branch)
	}
	diffText := strings.TrimSpace(rawDiff)
	if diffText == "" {
		diffText = "（无可用 diff）"
	}
	return diffText, nil
}

func (s *ScanService) callLLM(ctx context.Context, mc *model.ModelConfig, prompt string) (string, error) {
	sendPrompt := s.replaceSensitiveText(prompt)
	if s.llmCaller != nil {
		result, err := s.llmCaller(ctx, mc, sendPrompt)
		if err != nil {
			return "", err
		}
		return s.restoreSensitiveText(result), nil
	}
	provider, err := review.NewProvider(mc)
	if err != nil {
		return "", err
	}
	resp, err := provider.Completion(ctx, anyllm.CompletionParams{
		Model: mc.Model,
		Messages: []anyllm.Message{
			{Role: "user", Content: sendPrompt},
		},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}
	content, ok := resp.Choices[0].Message.Content.(string)
	if !ok {
		return "", fmt.Errorf("unexpected content type from LLM")
	}
	return s.restoreSensitiveText(content), nil
}

func (s *ScanService) replaceSensitiveText(text string) string {
	if s.sensitiveWords == nil {
		return text
	}
	return s.sensitiveWords.Replace(text)
}

func (s *ScanService) restoreSensitiveText(text string) string {
	if s.sensitiveWords == nil {
		return text
	}
	return s.sensitiveWords.Restore(text)
}

func parseRiskLevel(text string) (level string, hasRisk bool) {
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "风险等级") {
			continue
		}
		if _, after, ok := strings.Cut(line, "："); ok {
			line = after
		} else if _, after, ok := strings.Cut(line, ":"); ok {
			line = after
		}
		return classifyRiskLevel(line)
	}
	return classifyRiskLevel(text)
}

func classifyRiskLevel(text string) (level string, hasRisk bool) {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "高风险"):
		return "high", true
	case strings.Contains(lower, "中风险"):
		return "medium", true
	case strings.Contains(lower, "低风险"):
		return "low", true
	case strings.Contains(lower, "无风险"):
		return "none", false
	default:
		return "unknown", false
	}
}

func (s *ScanService) formatBranchSection(branch string, result *model.ScanBranchResult, commits []commitEntry) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 分支: %s\n\n", branch))
	sb.WriteString(fmt.Sprintf("- Commit 数量: %d\n", result.CommitCount))
	sb.WriteString(fmt.Sprintf("- 风险等级: **%s**\n", result.RiskLevel))
	stage := "仅 commit message"
	if result.AnalysisStage == model.ScanAnalysisStageWithDiff {
		stage = "commit message + diff"
	}
	sb.WriteString(fmt.Sprintf("- 分析阶段: %s\n\n", stage))
	sb.WriteString("### Commit 列表\n\n")
	for _, c := range commits {
		sb.WriteString(fmt.Sprintf("- `%s` %s (%s)\n", c.Hash, c.Message, c.Author))
	}
	sb.WriteString("\n### 分析结果\n\n")
	sb.WriteString(result.Result)
	sb.WriteString("\n\n---\n\n")
	return sb.String()
}

func (s *ScanService) buildReport(name, repoURL string, sections []string) string {
	var sb strings.Builder
	date := time.Now().Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("# 巡检报告: %s\n\n", name))
	sb.WriteString(fmt.Sprintf("- 仓库: %s\n", repoURL))
	sb.WriteString(fmt.Sprintf("- 日期: %s\n", date))
	sb.WriteString(fmt.Sprintf("- 有改动分支数: %d\n\n", len(sections)))
	sb.WriteString("---\n\n")
	for _, sec := range sections {
		sb.WriteString(sec)
	}
	return sb.String()
}

// uploadToNAS 通过 WebDAV PUT 上传报告文件
// 路径格式：{NasURL}/TMMTMM/代码巡检记录/{subDir}/{subDir}巡检-20060102-1504.md
func (s *ScanService) uploadToNAS(cfg *ScanConfig, name, content string) (string, error) {
	if cfg.NasURL == "" {
		return "", nil
	}
	if err := validateNASURL(cfg.NasURL); err != nil {
		return "", err
	}
	now := time.Now()
	filename := fmt.Sprintf("%s巡检-%s.md", cfg.NasSubDir, now.Format("20060102-1504"))
	baseURL := strings.TrimRight(cfg.NasURL, "/")

	// 群晖 WebDAV 中文路径需要逐段 PathEscape，不能整体 encode
	seg1 := url.PathEscape("TMMTMM")
	seg2 := url.PathEscape("代码巡检记录")
	seg3 := url.PathEscape(cfg.NasSubDir)
	segFile := url.PathEscape(filename)

	dirURL := baseURL + "/" + seg1 + "/" + seg2 + "/" + seg3 + "/"
	fileURL := dirURL + segFile

	// 逐层 MKCOL 确保目录存在（PUT 也复用这个 client）
	client := &http.Client{Timeout: 30 * time.Second}
	for _, dir := range []string{
		baseURL + "/" + seg1 + "/",
		baseURL + "/" + seg1 + "/" + seg2 + "/",
		dirURL,
	} {
		mkcolReq, err := http.NewRequest("MKCOL", dir, nil)
		if err != nil {
			continue
		}
		mkcolReq.SetBasicAuth(cfg.NasUsername, cfg.NasPassword)
		resp, err := client.Do(mkcolReq)
		if err == nil {
			_ = resp.Body.Close()
		}
	}

	// PUT 文件
	putReq, err := http.NewRequest("PUT", fileURL, strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("build PUT request: %w", err)
	}
	putReq.SetBasicAuth(cfg.NasUsername, cfg.NasPassword)
	putReq.Header.Set("Content-Type", "text/markdown; charset=utf-8")
	putReq.ContentLength = int64(len(content))

	resp, err := client.Do(putReq)
	if err != nil {
		return "", fmt.Errorf("WebDAV PUT: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("WebDAV PUT returned %d", resp.StatusCode)
	}
	return fileURL, nil
}

// cleanupOldReports 删除 NAS 上超过保留天数的报告文件（0 = 永久保留）
func (s *ScanService) cleanupOldReports(ctx context.Context, scheduleID int64, cfg *ScanConfig) {
	if cfg.NasURL == "" {
		return
	}
	if err := validateNASURL(cfg.NasURL); err != nil {
		return
	}
	retainDaysStr, _ := s.settings.GetRaw(model.GlobalConfigKeyScanRetainDays)
	retainDays := 0
	if v, err := strconv.Atoi(retainDaysStr); err == nil {
		retainDays = v
	}
	if retainDays <= 0 {
		return
	}

	before := time.Now().AddDate(0, 0, -retainDays)
	oldJobs, err := s.jobs.ListJobsOlderThan(scheduleID, before)
	if err != nil || len(oldJobs) == 0 {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	for _, j := range oldJobs {
		if j.ReportPath == "" {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, "DELETE", j.ReportPath, nil)
		if err != nil {
			continue
		}
		req.SetBasicAuth(cfg.NasUsername, cfg.NasPassword)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			// 清除 DB 里的 report_path，避免重复尝试删除
			j.ReportPath = ""
			_ = s.jobs.Update(&j)
		}
	}
}

// loadCheckpoints 从 schedule.BranchCheckpoints 读取 map[branch]lastCommit
func (s *ScanService) loadCheckpoints(sched *model.ScanSchedule) map[string]string {
	m := make(map[string]string)
	if sched.BranchCheckpoints == "" {
		return m
	}
	_ = json.Unmarshal([]byte(sched.BranchCheckpoints), &m)
	return m
}

// saveCheckpoints 将 checkpoints 序列化后写回 DB
func (s *ScanService) saveCheckpoints(sched *model.ScanSchedule, checkpoints map[string]string) error {
	b, err := json.Marshal(checkpoints)
	if err != nil {
		return err
	}
	sched.BranchCheckpoints = string(b)
	return s.schedules.Update(sched)
}

// getBranchHead 获取分支当前 HEAD commit hash
func (s *ScanService) getBranchHead(ctx context.Context, repoDir, branch string) (string, error) {
	out, err := runGit(ctx, repoDir, "git", "rev-parse", "origin/"+branch)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// getRecentCommits 获取分支最近 n 条 commit（第一次巡检用）
func (s *ScanService) getRecentCommits(ctx context.Context, repoDir, branch string, n int) ([]commitEntry, error) {
	out, err := runGit(ctx, repoDir,
		"git", "log",
		"--pretty=format:%H|%s|%an|%ai",
		fmt.Sprintf("-n%d", n),
		"origin/"+branch,
	)
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, err
	}
	return parseCommitLog(out), nil
}

// getCommitsBetween 获取 from..head 之间的 commit
func (s *ScanService) getCommitsBetween(ctx context.Context, repoDir, branch, fromCommit, headCommit string) ([]commitEntry, error) {
	out, err := runGit(ctx, repoDir,
		"git", "log",
		"--pretty=format:%H|%s|%an|%ai",
		fromCommit+".."+headCommit,
	)
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, err
	}
	return parseCommitLog(out), nil
}

func parseCommitLog(out string) []commitEntry {
	var entries []commitEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		entries = append(entries, commitEntry{
			Hash:    parts[0][:min(8, len(parts[0]))],
			Message: parts[1],
			Author:  parts[2],
			Time:    parts[3],
		})
	}
	return entries
}

func (s *ScanService) failJob(job *model.ScanJob, msg string) error {
	job.Status = model.ScanJobStatusFailed
	job.ErrorMessage = msg
	now := time.Now()
	job.FinishedAt = &now
	_ = s.jobs.Update(job)
	return fmt.Errorf("%s", msg)
}

// ---- util ----

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func ValidateNASURL(raw string) error {
	return validateNASURL(raw)
}

func validateNASURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid NAS URL")
	}
	return validateNASHost(u.Hostname())
}

func validateNASHost(host string) error {
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("invalid NAS host")
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return rejectUnsafeNASAddr(addr)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve NAS host: %w", err)
	}
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			return fmt.Errorf("invalid NAS host address")
		}
		if err := rejectUnsafeNASAddr(addr); err != nil {
			return err
		}
	}
	return nil
}

func rejectUnsafeNASAddr(addr netip.Addr) error {
	if addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return fmt.Errorf("NAS URL 不允许指向本机、链路本地、组播或未指定地址")
	}
	return nil
}

func injectCred(repoURL, username, password string) string {
	u, err := url.Parse(repoURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return repoURL
	}
	u.User = url.UserPassword(username, password)
	return u.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
