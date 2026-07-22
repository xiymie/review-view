package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"review-view/internal/model"
	"review-view/internal/review"
	"review-view/internal/store"
)

const defaultScanPrompt = `仓库 %s，分支 %s，日期 %s，共 %d 个提交：

%s

以下是本次变更的代码 diff：

%s

请结合 commit 信息和代码变更完成以下两项分析：

**一、风险识别**
识别以下类型的风险（有则列出，无则不提）：
- DB：数据库结构变更（建表/删表/加减字段/索引）
- DEP：依赖版本升级（可能引入 breaking change）
- CFG：配置变更（环境变量/密钥/服务地址/端口）
- BIZ：核心业务逻辑改动（支付/权限/核心流程）
- SEC：安全相关（认证/鉴权/加密/敏感数据）

**二、逻辑漏洞检查**
仅关注逻辑层面的问题，忽略语法错误，检查：
- 边界条件未处理（空值/零值/越界）
- 并发/竞态条件
- 错误处理缺失或忽略错误返回值
- 条件判断不严谨导致意外分支
- 数据流中的遗漏校验或状态不一致

**输出格式：**
**风险等级**：无风险 / 低风险 / 中风险 / 高风险
**命中风险类型**：（命中的标签，无则填"—"）
**风险说明**：（简要说明触发原因）
**逻辑漏洞**：（列出发现的逻辑问题；若无则填"未发现"）`

// ScanConfig 一次巡检所需的完整配置（由 schedule + 全局配置合并而来）
type ScanConfig struct {
	RepoURL     string
	Cred        *model.RepoCredential
	ModelConfig *model.ModelConfig
	Prompt      string
	NasURL      string
	NasUsername string
	NasPassword string
	NasSubDir   string
}

type ScanService struct {
	schedules store.ScanScheduleStore
	jobs      store.ScanJobStore
	models    store.ModelConfigStore
	creds     store.RepoCredentialStore
	settings  *SettingsService
	repoMgr   *review.RepositoryManager
}

func NewScanService(
	schedules store.ScanScheduleStore,
	jobs store.ScanJobStore,
	models store.ModelConfigStore,
	creds store.RepoCredentialStore,
	settings *SettingsService,
	repoMgr *review.RepositoryManager,
) *ScanService {
	return &ScanService{
		schedules: schedules,
		jobs:      jobs,
		models:    models,
		creds:     creds,
		settings:  settings,
		repoMgr:   repoMgr,
	}
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
	return s.schedules.Delete(id)
}

// ListJobs 获取某个配置的历史执行记录
func (s *ScanService) ListJobs(scheduleID int64, limit int) ([]model.ScanJob, error) {
	return s.jobs.ListBySchedule(scheduleID, limit)
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

// RunSchedule 执行一次巡检（同步，供调度器和手动触发调用）
func (s *ScanService) RunSchedule(ctx context.Context, scheduleID int64) error {
	sched, err := s.schedules.GetByID(scheduleID)
	if err != nil {
		return fmt.Errorf("get schedule: %w", err)
	}

	now := time.Now()
	job := &model.ScanJob{
		ScheduleID:  scheduleID,
		Status:      model.ScanJobStatusRunning,
		TriggeredAt: now,
	}
	if err := s.jobs.Create(job); err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	cfg, err := s.buildConfig(sched)
	if err != nil {
		return s.failJob(job, fmt.Sprintf("build config: %v", err))
	}

	// 确保仓库存在，使用 scan/ 子目录隔离，避免与项目仓库混用
	repoKey := fmt.Sprintf("scan/%d", scheduleID)
	repoDir := filepath.Join(s.repoMgr.BaseDir(), repoKey)
	if err := s.ensureRepo(ctx, repoDir, cfg); err != nil {
		return s.failJob(job, fmt.Sprintf("ensure repo: %v", err))
	}

	// 列出所有远程分支
	branches, err := s.listRemoteBranches(ctx, repoDir)
	if err != nil {
		return s.failJob(job, fmt.Sprintf("list branches: %v", err))
	}
	job.BranchCount = len(branches)

	// 读取上次各分支的 checkpoint（上次巡检到的 commit hash）
	checkpoints := s.loadCheckpoints(sched)

	changedCount := 0
	var reportSections []string

	for _, branch := range branches {
		// 获取该分支的 HEAD commit
		headCommit, err := s.getBranchHead(ctx, repoDir, branch)
		if err != nil || headCommit == "" {
			continue
		}

		lastCommit := checkpoints[branch]

		var commits []commitEntry
		if lastCommit == "" {
			// 第一次巡检：取最近 3 次 commit
			commits, err = s.getRecentCommits(ctx, repoDir, branch, 3)
		} else if lastCommit == headCommit {
			// 和上次一样，无改动
			continue
		} else {
			// 有新 commit：取 lastCommit 到 HEAD 之间的
			commits, err = s.getCommitsBetween(ctx, repoDir, branch, lastCommit, headCommit)
		}
		if err != nil || len(commits) == 0 {
			// 更新 checkpoint 即使没有 commit（可能 lastCommit 为空且最近无提交）
			if lastCommit == "" && headCommit != "" {
				checkpoints[branch] = headCommit
			}
			continue
		}
		changedCount++

		result, err := s.analyzeBranch(ctx, cfg, repoDir, branch, commits, sched.RepoURL, job.ID)
		if err != nil {
			result = &model.ScanBranchResult{
				JobID:         job.ID,
				BranchName:    branch,
				CommitCount:   len(commits),
				AnalysisStage: model.ScanAnalysisStageMessageOnly,
				Result:        fmt.Sprintf("分析失败: %v", err),
				HasRisk:       false,
				RiskLevel:     "unknown",
			}
		}
		if err2 := s.jobs.CreateBranchResult(result); err2 != nil {
			_ = err2
		}
		reportSections = append(reportSections, s.formatBranchSection(branch, result, commits))
		// 更新 checkpoint 为本次 HEAD
		checkpoints[branch] = headCommit
	}

	// 保存更新后的 checkpoints
	s.saveCheckpoints(sched, checkpoints)

	job.ChangedBranchCount = changedCount

	// 生成报告并上传 NAS
	var reportPath string
	if len(reportSections) > 0 {
		reportContent := s.buildReport(sched.Name, sched.RepoURL, reportSections)
		reportPath, err = s.uploadToNAS(cfg, sched.Name, reportContent)
		if err != nil {
			// 上传失败不中断，记录错误
			job.ErrorMessage = fmt.Sprintf("NAS upload failed: %v", err)
		}
	}

	finishedAt := time.Now()
	job.Status = model.ScanJobStatusCompleted
	job.FinishedAt = &finishedAt
	job.ReportPath = reportPath
	if err := s.jobs.Update(job); err != nil {
		return err
	}

	// 清理 NAS 过期报告
	s.cleanupOldReports(ctx, scheduleID, cfg)
	return nil
}

// ---- internal helpers ----

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
	prompt := sched.CustomPrompt
	if prompt == "" {
		prompt = globalPrompt
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
		Prompt:      prompt,
		NasURL:      nasURL,
		NasUsername: nasUser,
		NasPassword: nasPass,
		NasSubDir:   nasSubDir,
	}, nil
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
	// 构建 commit message 列表文本
	var sb strings.Builder
	for i, c := range commits {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (%s, %s)\n", i+1, c.Hash, c.Message, c.Author, c.Time))
	}
	commitText := sb.String()

	// 存储 commits JSON
	commitsJSON, _ := json.Marshal(commits)

	// 确定 fromCommit / toCommit
	var fromCommit, toCommit string
	if len(commits) > 0 {
		toCommit = commits[0].Hash
		fromCommit = commits[len(commits)-1].Hash
	}

	// 始终拉 diff
	var diffText string
	rawDiff, err := runGit(ctx, repoDir, "git", "log", "-p",
		fmt.Sprintf("%s..origin/%s", fromCommit, branch))
	if err != nil || strings.TrimSpace(rawDiff) == "" {
		// fallback：取最近 24 小时内的 diff
		rawDiff, _ = runGit(ctx, repoDir, "git", "log", "-p",
			"--after="+time.Now().Add(-24*time.Hour).Format("2006-01-02"),
			"origin/"+branch)
	}
	diffText = strings.TrimSpace(rawDiff)
	if len(diffText) > 12000 {
		diffText = diffText[:12000] + "\n...(已截断)"
	}
	if diffText == "" {
		diffText = "（无可用 diff）"
	}

	// 单次 LLM 调用，commit 信息 + diff 一起送入
	promptBase := cfg.Prompt
	if promptBase == "" {
		promptBase = defaultScanPrompt
	}
	today := time.Now().Format("2006-01-02")
	finalPrompt := fmt.Sprintf(promptBase, repoURL, branch, today, len(commits), commitText, diffText)

	finalResult, err := s.callLLM(ctx, cfg.ModelConfig, finalPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}

	stage := model.ScanAnalysisStageWithDiff
	riskLevel, hasRisk := parseRiskLevel(finalResult)

	return &model.ScanBranchResult{
		JobID:          jobID,
		BranchName:     branch,
		CommitCount:    len(commits),
		CommitMessages: string(commitsJSON),
		FromCommit:     fromCommit,
		ToCommit:       toCommit,
		AnalysisStage:  stage,
		Result:         finalResult,
		HasRisk:        hasRisk,
		RiskLevel:      riskLevel,
	}, nil
}

func (s *ScanService) callLLM(ctx context.Context, mc *model.ModelConfig, prompt string) (string, error) {
	provider, err := review.NewProvider(mc)
	if err != nil {
		return "", err
	}
	resp, err := provider.Completion(ctx, anyllm.CompletionParams{
		Model: mc.Model,
		Messages: []anyllm.Message{
			{Role: "user", Content: prompt},
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
	return content, nil
}

func parseRiskLevel(text string) (level string, hasRisk bool) {
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
func (s *ScanService) saveCheckpoints(sched *model.ScanSchedule, checkpoints map[string]string) {
	b, err := json.Marshal(checkpoints)
	if err != nil {
		return
	}
	sched.BranchCheckpoints = string(b)
	_ = s.schedules.Update(sched)
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

func injectCred(repoURL, username, password string) string {
	if !strings.HasPrefix(repoURL, "https://") && !strings.HasPrefix(repoURL, "http://") {
		return repoURL
	}
	prefix := "https://"
	if strings.HasPrefix(repoURL, "http://") {
		prefix = "http://"
	}
	rest := repoURL[len(prefix):]
	if atIdx := strings.Index(rest, "@"); atIdx != -1 {
		rest = rest[atIdx+1:]
	}
	return prefix + username + ":" + password + "@" + rest
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
