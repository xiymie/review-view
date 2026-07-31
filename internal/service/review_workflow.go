package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"review-view/internal/model"
	"review-view/internal/review"
)

// ReviewInput is the minimal input needed to execute a code review task.
// More execution context will be introduced in later refactor steps without
// changing Scheduler's queueing logic.
type ReviewInput struct {
	TaskID int64
}

// ReviewContext carries the loaded execution context for one review task.
//
// It is intentionally small for now: feature point 02 only makes the current
// ExecuteTask data flow explicit. Later Eino graph nodes can extend this as the
// shared state passed between prepare/run/finish stages.
type ReviewContext struct {
	Task           *model.Task
	Project        *model.Project
	ModelConfig    *model.ModelConfig
	RepoDir        string
	Prompt         string
	SkillPrompt    string
	StartedAt      time.Time
	TimeoutMinutes int
	RunCtx         context.Context
	Cancel         context.CancelFunc
	ReviewResult   *review.ReviewResult
	ReviewOutput   ReviewOutput
}

// ReviewOutput captures the final review execution result in a workflow-friendly
// shape. The legacy path still persists through model.Task directly, but future
// Eino workflows can return this structure between nodes.
type ReviewOutput struct {
	TaskID              int64
	Result              string
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

// ReviewWorkflow is the execution boundary for a code review task.
//
// Scheduler owns queueing, claiming, concurrency, and timing. ReviewWorkflow owns
// the actual task execution. Keeping this boundary explicit lets future Eino
// workflows replace the legacy implementation without changing scheduler logic.
type ReviewWorkflow interface {
	Run(ctx context.Context, input ReviewInput) error
}

// ReviewWorkflowFunc adapts a function to ReviewWorkflow. It is useful for tests
// and for small transitional workflow implementations.
type ReviewWorkflowFunc func(ctx context.Context, input ReviewInput) error

// Run executes the wrapped workflow function.
func (f ReviewWorkflowFunc) Run(ctx context.Context, input ReviewInput) error {
	return f(ctx, input)
}

type legacyReviewWorkflow struct {
	scheduler *Scheduler
}

func newLegacyReviewWorkflow(scheduler *Scheduler) *legacyReviewWorkflow {
	return &legacyReviewWorkflow{scheduler: scheduler}
}

func (w *legacyReviewWorkflow) Run(ctx context.Context, input ReviewInput) error {
	reviewCtx, err := w.loadReviewContext(input.TaskID)
	if err != nil {
		return err
	}
	return w.executeReviewContext(ctx, reviewCtx)
}

func (w *legacyReviewWorkflow) loadReviewContext(taskID int64) (*ReviewContext, error) {
	s := w.scheduler
	task, err := s.tasks.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	project, err := s.projects.GetByID(task.ProjectID)
	if err != nil {
		return nil, err
	}

	modelConfig, err := s.modelConfigs.GetByID(project.ModelConfigID)
	if err != nil {
		return nil, err
	}

	return &ReviewContext{
		Task:           task,
		Project:        project,
		ModelConfig:    modelConfig,
		TimeoutMinutes: s.getTaskTimeoutMinutes(project),
	}, nil
}

func (w *legacyReviewWorkflow) beginReviewContext(ctx context.Context, reviewCtx *ReviewContext) error {
	s := w.scheduler
	task := reviewCtx.Task

	reviewCtx.RunCtx, reviewCtx.Cancel = context.WithTimeout(ctx, time.Duration(reviewCtx.TimeoutMinutes)*time.Minute)
	s.RegisterCancel(task.ID, reviewCtx.Cancel)

	reviewCtx.StartedAt = time.Now()
	task.Status = model.TaskStatusRunning
	task.StartedAt = &reviewCtx.StartedAt
	if err := s.tasks.Update(task); err != nil {
		return err
	}
	s.appendLog(task.ID, model.TaskLogLevelInfo, "任务开始执行")
	return nil
}

func (w *legacyReviewWorkflow) cleanupReviewContext(reviewCtx *ReviewContext) {
	if reviewCtx == nil || reviewCtx.Task == nil {
		return
	}
	if reviewCtx.Cancel != nil {
		reviewCtx.Cancel()
	}
	w.scheduler.cancels.Delete(reviewCtx.Task.ID)
}

func (w *legacyReviewWorkflow) syncRepo(reviewCtx *ReviewContext) error {
	s := w.scheduler
	task := reviewCtx.Task
	project := reviewCtx.Project

	var cred *model.RepoCredential
	if project.RepoCredentialID != nil {
		cred, _ = s.credentials.GetByID(*project.RepoCredentialID)
	}

	repoDir, err := s.repoManager.EnsureRepo(reviewCtx.RunCtx, project.ID, project.RepoURL, project.Branch, cred)
	if err != nil {
		s.appendLog(task.ID, model.TaskLogLevelError, "代码仓库同步失败: "+err.Error())
		return s.failTask(task, err)
	}
	reviewCtx.RepoDir = repoDir
	s.appendLog(task.ID, model.TaskLogLevelInfo, "代码仓库同步完成")
	return nil
}

func (w *legacyReviewWorkflow) checkoutRepo(reviewCtx *ReviewContext) error {
	s := w.scheduler
	task := reviewCtx.Task

	// checkout 到目标 commit，使工作目录包含该 commit 的完整代码
	if err := s.repoManager.Checkout(reviewCtx.RunCtx, reviewCtx.RepoDir, task.ToCommit); err != nil {
		s.appendLog(task.ID, model.TaskLogLevelError, "代码迁出失败: "+err.Error())
		return s.failTask(task, err)
	}
	s.appendLog(task.ID, model.TaskLogLevelInfo, "已迁出到 commit "+task.ToCommit)
	return nil
}

func (w *legacyReviewWorkflow) loadTaskSkillPrompt(reviewCtx *ReviewContext) error {
	if reviewCtx == nil || reviewCtx.Task == nil {
		return nil
	}
	task := reviewCtx.Task
	skillIDs := decodeReviewSkillIDs(task.ReviewSkillIDs)
	if len(skillIDs) == 0 {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, "引入 Skill：空（本次审核未选择 Skill）")
		return nil
	}
	if w.scheduler.reviewSkills == nil {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelWarn, fmt.Sprintf("引入 Skill：已选择 %d 个，但 Skill 服务未初始化", len(skillIDs)))
		return nil
	}

	var names []string
	var unavailable []string
	for _, id := range skillIDs {
		skill, err := w.scheduler.reviewSkills.Get(id)
		if err != nil {
			unavailable = append(unavailable, fmt.Sprintf("#%d(不存在)", id))
			continue
		}
		if !skill.Enabled {
			unavailable = append(unavailable, fmt.Sprintf("%s#%d(已停用)", skill.Name, id))
			continue
		}
		names = append(names, fmt.Sprintf("%s#%d", skill.Name, id))
	}
	skillPrompt, err := w.scheduler.reviewSkills.BuildPromptForSkillIDs(skillIDs)
	if err != nil {
		return err
	}
	reviewCtx.SkillPrompt = skillPrompt
	if len(names) == 0 {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, "引入 Skill：空（所选 Skill 不可用或已停用）")
	} else {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("引入 Skill：%s（提示词 %d 字符）", strings.Join(names, "，"), len([]rune(skillPrompt))))
	}
	if len(unavailable) > 0 {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelWarn, "跳过不可用 Skill："+strings.Join(unavailable, "，"))
	}
	return nil
}

func (w *legacyReviewWorkflow) buildReviewPrompt(reviewCtx *ReviewContext) string {
	task := reviewCtx.Task
	sections := make([]string, 0, 4)
	if strings.TrimSpace(reviewCtx.SkillPrompt) != "" {
		sections = append(sections, strings.TrimSpace(reviewCtx.SkillPrompt))
	}
	if strings.TrimSpace(reviewCtx.ModelConfig.Prompt) != "" {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("读取模型全局提示词：有（%d 字符）", len([]rune(strings.TrimSpace(reviewCtx.ModelConfig.Prompt)))))
		sections = append(sections, strings.TrimSpace(reviewCtx.ModelConfig.Prompt))
	} else {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, "读取模型全局提示词：空")
	}
	if strings.TrimSpace(reviewCtx.Project.CustomPrompt) != "" {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("读取项目提示词：有（%d 字符）", len([]rune(strings.TrimSpace(reviewCtx.Project.CustomPrompt)))))
		sections = append(sections, "## 项目补充说明\n\n"+strings.TrimSpace(reviewCtx.Project.CustomPrompt))
	} else {
		w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, "读取项目提示词：空")
	}
	claudeMD, claudeStatus := loadReviewClaudeMD(reviewCtx.RepoDir)
	w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, claudeStatus)
	if claudeMD != "" {
		sections = append(sections, "## 项目 CLAUDE.md 细则\n\n"+claudeMD)
	}
	prompt := strings.Join(sections, "\n\n")
	if reviewCtx.ModelConfig.Type == model.ModelTypeClaudeCLI {
		prompt = review.BuildCLIPrompt(prompt, reviewCtx.Task.FromCommit, reviewCtx.Task.ToCommit)
	}
	w.scheduler.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("构建审核提示词完成：%d 字符", len([]rune(prompt))))
	return prompt
}

func loadReviewClaudeMD(repoDir string) (string, string) {
	if strings.TrimSpace(repoDir) == "" {
		return "", "读取 CLAUDE.md：空（仓库目录为空）"
	}
	content, err := os.ReadFile(filepath.Join(repoDir, "CLAUDE.md"))
	if err != nil {
		return "", "读取 CLAUDE.md：未找到"
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		return "", "读取 CLAUDE.md：存在但为空"
	}
	return text, fmt.Sprintf("读取 CLAUDE.md：有（%d 字符，已完整注入）", len([]rune(text)))
}

func (w *legacyReviewWorkflow) executeReviewContext(ctx context.Context, reviewCtx *ReviewContext) error {
	if err := w.beginReviewContext(ctx, reviewCtx); err != nil {
		return err
	}
	defer w.cleanupReviewContext(reviewCtx)

	if err := w.syncRepo(reviewCtx); err != nil {
		return err
	}
	if err := w.checkoutRepo(reviewCtx); err != nil {
		return err
	}
	if err := w.loadTaskSkillPrompt(reviewCtx); err != nil {
		return err
	}
	w.scheduler.appendLog(reviewCtx.Task.ID, model.TaskLogLevelInfo, fmt.Sprintf("读取 commit 信息：%d 字符", len([]rune(reviewCtx.Task.CommitMessages))))
	w.scheduler.appendLog(reviewCtx.Task.ID, model.TaskLogLevelInfo, fmt.Sprintf("读取 diff 代码：%d 字符", len([]rune(reviewCtx.Task.DiffContent))))
	if reviewCtx.Prompt == "" {
		reviewCtx.Prompt = w.buildReviewPrompt(reviewCtx)
	}
	return w.executeModelAndPersist(reviewCtx)
}

func (w *legacyReviewWorkflow) executeModelAndPersist(reviewCtx *ReviewContext) error {
	if err := w.executeModel(reviewCtx); err != nil {
		return err
	}
	if reviewCtx.ReviewResult == nil {
		return nil
	}
	if err := w.scanSensitiveWords(reviewCtx); err != nil {
		return err
	}
	return w.persistReviewResult(reviewCtx)
}

func (w *legacyReviewWorkflow) executeModel(reviewCtx *ReviewContext) error {
	s := w.scheduler
	task := reviewCtx.Task
	modelConfig := reviewCtx.ModelConfig
	timeoutMinutes := reviewCtx.TimeoutMinutes

	// Agent 路径：变更信息已在 Task 创建时填充，直接使用
	s.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("变更文件: %d 字符, commit 记录: %d 字符", len(task.DiffContent), len(task.CommitMessages)))

	s.appendLog(task.ID, model.TaskLogLevelInfo, "开始调用 "+string(modelConfig.Type))

	var outputChars int64
	onChunk := func(text string) {
		s.cache.AppendResultChunk(task.ID, text)
		n := atomic.AddInt64(&outputChars, int64(len([]rune(text))))
		// 流式过程中按字符数估算 output token（中英混合约 3 字符/token）
		s.cache.UpdateTokens(task.ID, 0, n/3)
	}
	onLog := func(level, msg string) {
		s.appendLog(task.ID, model.TaskLogLevelInfo, msg)
	}

	s.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("提交 AI 审核：model=%s, prompt=%d 字符, diff=%d 字符, commits=%d 字符", modelConfig.Name, len([]rune(reviewCtx.Prompt)), len([]rune(task.DiffContent)), len([]rune(task.CommitMessages))))
	result, err := s.reviewerFactory(modelConfig).Review(reviewCtx.RunCtx, review.ReviewParams{
		Prompt:         reviewCtx.Prompt,
		WorkDir:        reviewCtx.RepoDir,
		FromCommit:     task.FromCommit,
		ToCommit:       task.ToCommit,
		DiffContent:    task.DiffContent,
		CommitMessages: task.CommitMessages,
		ModelConfig:    modelConfig,
		OnChunk:        onChunk,
		OnLog:          onLog,
		Replace:        s.sensitiveWordReplacer(),
		Restore:        s.sensitiveWordRestorer(),
	})
	if err != nil {
		switch reviewCtx.RunCtx.Err() {
		case context.DeadlineExceeded:
			s.appendLog(task.ID, model.TaskLogLevelError, fmt.Sprintf("任务超时 (%d 分钟)", timeoutMinutes))
			return s.failTask(task, fmt.Errorf("任务超时"))
		case context.Canceled:
			s.appendLog(task.ID, model.TaskLogLevelInfo, "任务被取消")
			return s.cancelTaskRecord(task)
		default:
			s.appendLog(task.ID, model.TaskLogLevelError, "Review 调用失败: "+err.Error())
			return s.failTask(task, err)
		}
	}

	s.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("AI 返回审核结果：结果 %d 字符，耗时 %dms", len([]rune(result.Content)), result.DurationMs))
	s.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("Review 完成，耗时 %dms", result.DurationMs))

	task.Status = model.TaskStatusCompleted
	task.Result = result.Content
	reviewCtx.ReviewResult = result
	return nil
}

func (w *legacyReviewWorkflow) scanSensitiveWords(reviewCtx *ReviewContext) error {
	s := w.scheduler
	task := reviewCtx.Task

	// 敏感词检测：扫描已 checkout 的工作区，命中结果前置拼接到报告
	if hits, configured, scanErr := s.scanSensitiveWords(reviewCtx.RunCtx, reviewCtx.RepoDir); scanErr != nil {
		s.appendLog(task.ID, model.TaskLogLevelError, "敏感词检测失败: "+scanErr.Error())
	} else if configured {
		s.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("敏感词检测完成，命中 %d 处", len(hits)))
		for i, hit := range hits {
			if i >= 20 {
				s.appendLog(task.ID, model.TaskLogLevelWarn, fmt.Sprintf("敏感词命中日志超过 20 条，其余 %d 条已省略，完整信息见报告", len(hits)-20))
				break
			}
			s.appendLog(task.ID, model.TaskLogLevelWarn, fmt.Sprintf("敏感词命中：%s:%d word=%q snippet=%q", hit.File, hit.Line, hit.Word, hit.Snippet))
		}
		task.Result = buildSensitiveReport(hits) + "\n" + task.Result
	}
	return nil
}

func (w *legacyReviewWorkflow) persistReviewResult(reviewCtx *ReviewContext) error {
	s := w.scheduler
	task := reviewCtx.Task
	project := reviewCtx.Project
	result := reviewCtx.ReviewResult
	if result == nil {
		return fmt.Errorf("review result is nil")
	}

	finishedAt := time.Now()
	reviewOutput := ReviewOutput{
		TaskID:              task.ID,
		Result:              task.Result,
		InputTokens:         result.InputTokens,
		OutputTokens:        result.OutputTokens,
		CacheCreationTokens: result.CacheCreationTokens,
		CacheReadTokens:     result.CacheReadTokens,
	}
	reviewCtx.ReviewOutput = reviewOutput
	task.InputTokens = reviewOutput.InputTokens
	task.OutputTokens = reviewOutput.OutputTokens
	task.CacheCreationTokens = reviewOutput.CacheCreationTokens
	task.CacheReadTokens = reviewOutput.CacheReadTokens
	task.FinishedAt = &finishedAt
	s.writeReviewFile(task.ProjectID, task)
	if err := s.tasks.Update(task); err != nil {
		return err
	}
	// 推送最终精确 token 数，显式触发 SSE handler 的 done 检测
	s.cache.UpdateTokens(task.ID, task.InputTokens, task.OutputTokens)
	s.cache.SendNotify(task.ID)
	s.cache.RemoveResult(task.ID)
	s.flushLogs(task.ID)

	s.dispatchNotify(task, project)

	// 只有增量 review（from 等于 LastReviewedCommit）才更新 LastReviewedCommit
	if task.FromCommit == project.LastReviewedCommit {
		project.LastReviewedCommit = task.ToCommit
		return s.projects.Update(project)
	}
	return nil
}
