package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"review-view/internal/model"
)

type scanScheduleWorkflowInput struct {
	ScheduleID int64
}

type scanScheduleCallbackJobIDKey struct{}

var scanScheduleWorkflowNodeNames = map[string]struct{}{
	"create_job":       {},
	"build_config":     {},
	"prepare_repo":     {},
	"list_branches":    {},
	"process_branches": {},
	"finalize":         {},
}

type scanScheduleWorkflowState struct {
	Schedule       *model.ScanSchedule
	Job            *model.ScanJob
	Config         *ScanConfig
	RepoDir        string
	Branches       []string
	Checkpoints    map[string]string
	ChangedCount   int
	RiskCount      int
	ReportSections []string
	ReportPath     string
}

type scanScheduleWorkflow struct {
	service *ScanService
	runner  compose.Runnable[scanScheduleWorkflowInput, *model.ScanJob]
	initErr error
}

func newScanScheduleWorkflow(service *ScanService) *scanScheduleWorkflow {
	workflow := &scanScheduleWorkflow{service: service}
	workflow.runner, workflow.initErr = workflow.build(context.Background())
	return workflow
}

func (w *scanScheduleWorkflow) Run(ctx context.Context, scheduleID int64) error {
	if w.initErr != nil {
		return w.initErr
	}
	_, err := w.runner.Invoke(ctx, scanScheduleWorkflowInput{ScheduleID: scheduleID}, compose.WithCallbacks(w.jobLogCallback()))
	return err
}

func (w *scanScheduleWorkflow) build(ctx context.Context) (compose.Runnable[scanScheduleWorkflowInput, *model.ScanJob], error) {
	chain := compose.NewChain[scanScheduleWorkflowInput, *model.ScanJob]().
		AppendLambda(compose.InvokableLambda(w.createJob), compose.WithNodeName("create_job")).
		AppendLambda(compose.InvokableLambda(w.buildConfig), compose.WithNodeName("build_config")).
		AppendLambda(compose.InvokableLambda(w.prepareRepo), compose.WithNodeName("prepare_repo")).
		AppendLambda(compose.InvokableLambda(w.listBranches), compose.WithNodeName("list_branches")).
		AppendLambda(compose.InvokableLambda(w.processBranches), compose.WithNodeName("process_branches")).
		AppendLambda(compose.InvokableLambda(w.finalize), compose.WithNodeName("finalize"))

	return chain.Compile(ctx, compose.WithGraphName("scan_schedule_workflow_v1"))
}

func (w *scanScheduleWorkflow) jobLogCallback() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if !isScanScheduleWorkflowNode(info) {
				return ctx
			}
			jobID := scanJobIDFromCallbackInput(input)
			if jobID == 0 {
				return ctx
			}
			w.appendJobLog(jobID, model.ScanJobLogLevelInfo, fmt.Sprintf("Scan 节点 %s 开始", info.Name))
			return context.WithValue(ctx, scanScheduleCallbackJobIDKey{}, jobID)
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if !isScanScheduleWorkflowNode(info) {
				return ctx
			}
			jobID, _ := ctx.Value(scanScheduleCallbackJobIDKey{}).(int64)
			if jobID == 0 {
				jobID = scanJobIDFromCallbackInput(output)
			}
			if jobID != 0 {
				w.appendJobLog(jobID, model.ScanJobLogLevelInfo, fmt.Sprintf("Scan 节点 %s 完成", info.Name))
			}
			return context.WithValue(ctx, scanScheduleCallbackJobIDKey{}, jobID)
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if !isScanScheduleWorkflowNode(info) {
				return ctx
			}
			if jobID, ok := ctx.Value(scanScheduleCallbackJobIDKey{}).(int64); ok {
				w.appendJobLog(jobID, model.ScanJobLogLevelError, fmt.Sprintf("Scan 节点 %s 失败: %v", info.Name, err))
			}
			return ctx
		}).
		Build()
}

func isScanScheduleWorkflowNode(info *callbacks.RunInfo) bool {
	if info == nil {
		return false
	}
	_, ok := scanScheduleWorkflowNodeNames[info.Name]
	return ok
}

func scanJobIDFromCallbackInput(input callbacks.CallbackInput) int64 {
	switch v := input.(type) {
	case *scanScheduleWorkflowState:
		if v != nil && v.Job != nil {
			return v.Job.ID
		}
	case scanScheduleWorkflowState:
		if v.Job != nil {
			return v.Job.ID
		}
	case *model.ScanJob:
		if v != nil {
			return v.ID
		}
	case model.ScanJob:
		return v.ID
	}
	return 0
}

func (w *scanScheduleWorkflow) appendJobLog(jobID int64, level model.ScanJobLogLevel, message string) {
	if jobID == 0 {
		return
	}
	_ = w.service.jobs.AppendJobLog(jobID, level, message)
}

func (w *scanScheduleWorkflow) createJob(_ context.Context, input scanScheduleWorkflowInput) (*scanScheduleWorkflowState, error) {
	svc := w.service
	sched, err := svc.schedules.GetByID(input.ScheduleID)
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	job := &model.ScanJob{
		ScheduleID:  input.ScheduleID,
		Status:      model.ScanJobStatusRunning,
		TriggeredAt: time.Now(),
	}
	if err := svc.jobs.Create(job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	w.appendJobLog(job.ID, model.ScanJobLogLevelInfo, "Scan 节点 create_job 开始")

	return &scanScheduleWorkflowState{
		Schedule: sched,
		Job:      job,
	}, nil
}

func (w *scanScheduleWorkflow) buildConfig(_ context.Context, state *scanScheduleWorkflowState) (*scanScheduleWorkflowState, error) {
	if state == nil {
		return state, fmt.Errorf("scan schedule workflow state is nil")
	}
	cfg, err := w.service.buildConfig(state.Schedule)
	if err != nil {
		return state, w.service.failJob(state.Job, fmt.Sprintf("build config: %v", err))
	}
	state.Config = cfg

	// --- prompt source diagnostics ---
	jobID := state.Job.ID
	if strings.TrimSpace(cfg.SkillPrompt) != "" {
		w.appendJobLog(jobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[build_config] 项目技能提示词: 有 (%d 字符)", len(cfg.SkillPrompt)))
	} else {
		w.appendJobLog(jobID, model.ScanJobLogLevelInfo, "[build_config] 项目技能提示词: 无")
	}
	globalPrompt, _ := w.service.settings.GetRaw(model.GlobalConfigKeyScanPrompt)
	if strings.TrimSpace(globalPrompt) != "" {
		w.appendJobLog(jobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[build_config] 全局自定义提示词: 有 (%d 字符)", len(strings.TrimSpace(globalPrompt))))
	} else {
		w.appendJobLog(jobID, model.ScanJobLogLevelInfo, "[build_config] 全局/默认提示词: 使用内置默认")
	}
	if w.service.projectStore != nil {
		if projects, err2 := w.service.projectStore.List(); err2 == nil {
			var projPrompt string
			repoURL := strings.TrimSpace(state.Schedule.RepoURL)
			for i := range projects {
				if strings.TrimSpace(projects[i].RepoURL) == repoURL {
					projPrompt = projects[i].CustomPrompt
					break
				}
			}
			if strings.TrimSpace(projPrompt) != "" {
				w.appendJobLog(jobID, model.ScanJobLogLevelInfo,
					fmt.Sprintf("[build_config] 项目自定义提示词: 有 (%d 字符)", len(strings.TrimSpace(projPrompt))))
			} else {
				w.appendJobLog(jobID, model.ScanJobLogLevelInfo, "[build_config] 项目自定义提示词: 无")
			}
		}
	}
	if strings.TrimSpace(state.Schedule.CustomPrompt) != "" {
		w.appendJobLog(jobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[build_config] 巡检自定义提示词: 有 (%d 字符)", len(strings.TrimSpace(state.Schedule.CustomPrompt))))
	} else {
		w.appendJobLog(jobID, model.ScanJobLogLevelInfo, "[build_config] 巡检自定义提示词: 无")
	}

	return state, nil
}

func (w *scanScheduleWorkflow) prepareRepo(ctx context.Context, state *scanScheduleWorkflowState) (*scanScheduleWorkflowState, error) {
	if state == nil {
		return state, fmt.Errorf("scan schedule workflow state is nil")
	}
	repoKey := fmt.Sprintf("scan/%d", state.Schedule.ID)
	state.RepoDir = filepath.Join(w.service.repoMgr.BaseDir(), repoKey)
	w.appendJobLog(state.Job.ID, model.ScanJobLogLevelInfo,
		fmt.Sprintf("[prepare_repo] 仓库目录: %s", state.RepoDir))
	if err := w.service.prepareScanRepo(ctx, state.RepoDir, state.Config); err != nil {
		return state, w.service.failJob(state.Job, fmt.Sprintf("ensure repo: %v", err))
	}
	return state, nil
}

func (w *scanScheduleWorkflow) listBranches(ctx context.Context, state *scanScheduleWorkflowState) (*scanScheduleWorkflowState, error) {
	if state == nil {
		return state, fmt.Errorf("scan schedule workflow state is nil")
	}
	branches, err := w.service.loadRemoteBranches(ctx, state.RepoDir)
	if err != nil {
		return state, w.service.failJob(state.Job, fmt.Sprintf("list branches: %v", err))
	}
	state.Branches = branches
	state.Job.BranchCount = len(branches)
	state.Checkpoints = w.service.loadCheckpoints(state.Schedule)
	w.appendJobLog(state.Job.ID, model.ScanJobLogLevelInfo,
		fmt.Sprintf("[list_branches] 发现分支数: %d", len(branches)))
	return state, nil
}

func (w *scanScheduleWorkflow) processBranches(ctx context.Context, state *scanScheduleWorkflowState) (*scanScheduleWorkflowState, error) {
	if state == nil {
		return state, fmt.Errorf("scan schedule workflow state is nil")
	}
	checkpoints := state.Checkpoints
	for _, branch := range state.Branches {
		headCommit, err := w.service.loadBranchHead(ctx, state.RepoDir, branch)
		if err != nil || headCommit == "" {
			continue
		}

		lastCommit := checkpoints[branch]
		var commits []commitEntry
		if lastCommit == "" {
			commits, err = w.service.loadRecentCommits(ctx, state.RepoDir, branch, 3)
		} else if lastCommit == headCommit {
			continue
		} else {
			commits, err = w.service.loadCommitsBetween(ctx, state.RepoDir, branch, lastCommit, headCommit)
		}
		if err != nil || len(commits) == 0 {
			if lastCommit == "" && headCommit != "" {
				checkpoints[branch] = headCommit
			}
			continue
		}
		state.ChangedCount++

		fromCommit := ""
		toCommit := ""
		if len(commits) > 0 {
			toCommit = commits[0].Hash
			fromCommit = commits[len(commits)-1].Hash
		}
		w.appendJobLog(state.Job.ID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[process_branches] 分支 %s: commit 数=%d, from=%s, to=%s",
				branch, len(commits), fromCommit, toCommit))

		result, err := w.service.analyzeBranch(ctx, state.Config, state.RepoDir, branch, commits, state.Schedule.RepoURL, state.Job.ID)
		if err != nil {
			result = &model.ScanBranchResult{
				JobID:         state.Job.ID,
				BranchName:    branch,
				CommitCount:   len(commits),
				AnalysisStage: model.ScanAnalysisStageMessageOnly,
				Result:        fmt.Sprintf("分析失败: %v", err),
				HasRisk:       false,
				RiskLevel:     "unknown",
			}
		}
		if err2 := w.service.jobs.CreateBranchResult(result); err2 != nil {
			_ = err2
		}
		if result.HasRisk {
			state.RiskCount++
		}
		state.ReportSections = append(state.ReportSections, w.service.formatBranchSection(branch, result, commits))
		checkpoints[branch] = headCommit
	}
	return state, nil
}

func (w *scanScheduleWorkflow) finalize(ctx context.Context, state *scanScheduleWorkflowState) (*model.ScanJob, error) {
	if state == nil {
		return nil, fmt.Errorf("scan schedule workflow state is nil")
	}
	if err := w.service.saveCheckpoints(state.Schedule, state.Checkpoints); err != nil {
		return nil, w.service.failJob(state.Job, fmt.Sprintf("save checkpoints: %v", err))
	}
	state.Job.ChangedBranchCount = state.ChangedCount
	state.Job.RiskCount = state.RiskCount

	if len(state.ReportSections) > 0 {
		reportContent := w.service.buildReport(state.Schedule.Name, state.Schedule.RepoURL, state.ReportSections)
		reportPath, err := w.service.uploadScanReport(state.Config, state.Schedule.Name, reportContent)
		if err != nil {
			state.Job.ErrorMessage = fmt.Sprintf("NAS upload failed: %v", err)
		} else {
			state.ReportPath = reportPath
		}
	}

	finishedAt := time.Now()
	state.Job.Status = model.ScanJobStatusCompleted
	state.Job.FinishedAt = &finishedAt
	state.Job.ReportPath = state.ReportPath
	if err := w.service.jobs.Update(state.Job); err != nil {
		return nil, err
	}

	w.service.cleanOldScanReports(ctx, state.Schedule.ID, state.Config)
	return state.Job, nil
}
