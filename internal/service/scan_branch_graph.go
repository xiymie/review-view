package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"review-view/internal/model"
)

type scanBranchGraphInput struct {
	Config  *ScanConfig
	RepoDir string
	Branch  string
	Commits []commitEntry
	RepoURL string
	JobID   int64
}

type scanBranchCallbackJobIDKey struct{}
type scanBranchCallbackBranchKey struct{}

var scanBranchGraphNodeNames = map[string]struct{}{
	"prepare_commits": {},
	"load_diff":       {},
	"build_prompt":    {},
	"call_llm":        {},
	"build_result":    {},
}

type scanBranchGraphState struct {
	Input           scanBranchGraphInput
	CommitText      string
	CommitsJSON     string
	FromCommit      string
	ToCommit        string
	DiffText        string
	ProjectRules    string
	ReplacementHits []sensitiveReplacementHit
	Prompt          string
	LLMResult       string
}

type scanBranchGraph struct {
	service *ScanService
	runner  compose.Runnable[scanBranchGraphInput, *model.ScanBranchResult]
	initErr error
}

func newScanBranchGraph(service *ScanService) *scanBranchGraph {
	graph := &scanBranchGraph{service: service}
	graph.runner, graph.initErr = graph.build(context.Background())
	return graph
}

func (g *scanBranchGraph) Run(ctx context.Context, input scanBranchGraphInput) (*model.ScanBranchResult, error) {
	if g.initErr != nil {
		return nil, g.initErr
	}
	return g.runner.Invoke(ctx, input, compose.WithCallbacks(g.jobLogCallback()))
}

func (g *scanBranchGraph) build(ctx context.Context) (compose.Runnable[scanBranchGraphInput, *model.ScanBranchResult], error) {
	chain := compose.NewChain[scanBranchGraphInput, *model.ScanBranchResult]().
		AppendLambda(compose.InvokableLambda(g.prepareCommits), compose.WithNodeName("prepare_commits")).
		AppendLambda(compose.InvokableLambda(g.loadDiff), compose.WithNodeName("load_diff")).
		AppendLambda(compose.InvokableLambda(g.buildPrompt), compose.WithNodeName("build_prompt")).
		AppendLambda(compose.InvokableLambda(g.callLLM), compose.WithNodeName("call_llm")).
		AppendLambda(compose.InvokableLambda(g.buildResult), compose.WithNodeName("build_result"))

	return chain.Compile(ctx, compose.WithGraphName("scan_branch_graph_v1"))
}

func (g *scanBranchGraph) jobLogCallback() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if !isScanBranchGraphNode(info) {
				return ctx
			}
			jobID, branch := scanBranchIdentityFromCallbackInput(input)
			if jobID == 0 || branch == "" {
				return ctx
			}
			g.appendBranchLog(jobID, model.ScanJobLogLevelInfo, fmt.Sprintf("Scan 分支 %s 节点 %s 开始", branch, info.Name))
			ctx = context.WithValue(ctx, scanBranchCallbackJobIDKey{}, jobID)
			return context.WithValue(ctx, scanBranchCallbackBranchKey{}, branch)
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if !isScanBranchGraphNode(info) {
				return ctx
			}
			jobID, _ := ctx.Value(scanBranchCallbackJobIDKey{}).(int64)
			branch, _ := ctx.Value(scanBranchCallbackBranchKey{}).(string)
			if jobID == 0 || branch == "" {
				jobID, branch = scanBranchIdentityFromCallbackInput(output)
			}
			if jobID != 0 && branch != "" {
				g.appendBranchLog(jobID, model.ScanJobLogLevelInfo, fmt.Sprintf("Scan 分支 %s 节点 %s 完成", branch, info.Name))
			}
			ctx = context.WithValue(ctx, scanBranchCallbackJobIDKey{}, jobID)
			return context.WithValue(ctx, scanBranchCallbackBranchKey{}, branch)
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if !isScanBranchGraphNode(info) {
				return ctx
			}
			jobID, _ := ctx.Value(scanBranchCallbackJobIDKey{}).(int64)
			branch, _ := ctx.Value(scanBranchCallbackBranchKey{}).(string)
			if jobID != 0 && branch != "" {
				g.appendBranchLog(jobID, model.ScanJobLogLevelError, fmt.Sprintf("Scan 分支 %s 节点 %s 失败: %v", branch, info.Name, err))
			}
			return ctx
		}).
		Build()
}

func isScanBranchGraphNode(info *callbacks.RunInfo) bool {
	if info == nil {
		return false
	}
	_, ok := scanBranchGraphNodeNames[info.Name]
	return ok
}

func scanBranchIdentityFromCallbackInput(input callbacks.CallbackInput) (int64, string) {
	switch v := input.(type) {
	case scanBranchGraphInput:
		return v.JobID, v.Branch
	case *scanBranchGraphState:
		if v != nil {
			return v.Input.JobID, v.Input.Branch
		}
	case scanBranchGraphState:
		return v.Input.JobID, v.Input.Branch
	case *model.ScanBranchResult:
		if v != nil {
			return v.JobID, v.BranchName
		}
	case model.ScanBranchResult:
		return v.JobID, v.BranchName
	}
	return 0, ""
}

func (g *scanBranchGraph) appendBranchLog(jobID int64, level model.ScanJobLogLevel, message string) {
	if jobID == 0 || g.service == nil || g.service.jobs == nil {
		return
	}
	_ = g.service.jobs.AppendJobLog(jobID, level, message)
}

func (g *scanBranchGraph) prepareCommits(_ context.Context, input scanBranchGraphInput) (*scanBranchGraphState, error) {
	state := &scanBranchGraphState{Input: input}

	var sb strings.Builder
	for i, c := range input.Commits {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (%s, %s)\n", i+1, c.Hash, c.Message, c.Author, c.Time))
	}
	state.CommitText = sb.String()

	commitsJSON, _ := json.Marshal(input.Commits)
	state.CommitsJSON = string(commitsJSON)

	if len(input.Commits) > 0 {
		state.ToCommit = input.Commits[0].Hash
		state.FromCommit = input.Commits[len(input.Commits)-1].Hash
	}
	return state, nil
}

func (g *scanBranchGraph) loadDiff(ctx context.Context, state *scanBranchGraphState) (*scanBranchGraphState, error) {
	if state == nil {
		return state, fmt.Errorf("scan branch graph state is nil")
	}
	diffText, err := g.service.loadBranchDiff(ctx, state.Input.RepoDir, state.Input.Branch, state.FromCommit)
	if err != nil {
		return state, err
	}
	state.DiffText = diffText
	g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
		fmt.Sprintf("[load_diff] 分支 %s diff 长度: %d 字符", state.Input.Branch, len(diffText)))
	g.scanAndLogSensitiveReplacements(ctx, state)
	return state, nil
}

func (g *scanBranchGraph) scanAndLogSensitiveReplacements(ctx context.Context, state *scanBranchGraphState) {
	if state == nil {
		return
	}
	if g.service == nil || g.service.sensitiveWords == nil {
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[sensitive_replace] 分支 %s 未配置敏感词替换服务", state.Input.Branch))
		return
	}
	hits, configured, err := g.service.scanSensitiveReplacements(ctx, state.Input.RepoDir)
	if err != nil {
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelError,
			fmt.Sprintf("[sensitive_replace] 分支 %s 替换词扫描失败: %v", state.Input.Branch, err))
		return
	}
	if !configured {
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[sensitive_replace] 分支 %s 未配置替换类敏感词", state.Input.Branch))
		return
	}
	state.ReplacementHits = hits
	if len(hits) == 0 {
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[sensitive_replace] 分支 %s 已启用替换类敏感词，未命中", state.Input.Branch))
		return
	}
	g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelWarn,
		fmt.Sprintf("[sensitive_replace] 分支 %s 命中 %d 处替换类敏感词，提交 AI 前会替换为配置的替换词", state.Input.Branch, len(hits)))
	for i, hit := range hits {
		if i >= 20 {
			g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelWarn,
				fmt.Sprintf("[sensitive_replace] 分支 %s 命中日志超过 20 条，其余 %d 条省略", state.Input.Branch, len(hits)-20))
			break
		}
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelWarn,
			fmt.Sprintf("[sensitive_replace] %s:%d original=%q replacement=%q snippet=%q", hit.File, hit.Line, hit.Original, hit.Replacement, hit.Snippet))
	}
}

func (g *scanBranchGraph) buildPrompt(_ context.Context, state *scanBranchGraphState) (*scanBranchGraphState, error) {
	if state == nil {
		return state, fmt.Errorf("scan branch graph state is nil")
	}
	promptBase := state.Input.Config.Prompt
	if promptBase == "" {
		promptBase = defaultScanPrompt
	}
	state.ProjectRules = loadScanProjectRules(state.Input.RepoDir)

	// log CLAUDE.md status
	claudePath := filepath.Join(state.Input.RepoDir, "CLAUDE.md")
	if rawContent, err := os.ReadFile(claudePath); err != nil {
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[build_prompt] 分支 %s CLAUDE.md: 不存在", state.Input.Branch))
	} else {
		rawLen := len([]rune(strings.TrimSpace(string(rawContent))))
		if rawLen == 0 {
			g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
				fmt.Sprintf("[build_prompt] 分支 %s CLAUDE.md: 存在但为空", state.Input.Branch))
		} else {
			g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
				fmt.Sprintf("[build_prompt] 分支 %s CLAUDE.md: 存在 (%d 字符，已完整注入)", state.Input.Branch, rawLen))
		}
	}

	today := time.Now().Format("2006-01-02")
	basePrompt := fmt.Sprintf(promptBase, state.Input.RepoURL, state.Input.Branch, today, len(state.Input.Commits), state.CommitText, state.DiffText)
	sections := make([]string, 0, 5)
	if strings.TrimSpace(state.Input.Config.SkillPrompt) != "" {
		sections = append(sections, strings.TrimSpace(state.Input.Config.SkillPrompt))
	}
	sections = append(sections, strings.TrimSpace(basePrompt))
	if state.ProjectRules != "" {
		sections = append(sections, "## 项目 CLAUDE.md 细则\n\n"+state.ProjectRules)
	}
	sections = append(sections, "## Commit 信息\n\n"+strings.TrimSpace(state.CommitText))
	sections = append(sections, "## Diff 代码\n\n"+strings.TrimSpace(state.DiffText))
	state.Prompt = strings.Join(sections, "\n\n")
	return state, nil
}

func loadScanProjectRules(repoDir string) string {
	if strings.TrimSpace(repoDir) == "" {
		return ""
	}
	content, err := os.ReadFile(filepath.Join(repoDir, "CLAUDE.md"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func (g *scanBranchGraph) callLLM(ctx context.Context, state *scanBranchGraphState) (*scanBranchGraphState, error) {
	if state == nil {
		return state, fmt.Errorf("scan branch graph state is nil")
	}
	g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
		fmt.Sprintf("[call_llm] 分支 %s 提交 LLM，提示词长度: %d 字符", state.Input.Branch, len(state.Prompt)))
	result, err := g.service.callLLM(ctx, state.Input.Config.ModelConfig, state.Prompt)
	if err != nil {
		return state, fmt.Errorf("LLM call: %w", err)
	}
	state.LLMResult = result
	g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
		fmt.Sprintf("[call_llm] 分支 %s LLM 返回，响应长度: %d 字符", state.Input.Branch, len(result)))
	return state, nil
}

func (g *scanBranchGraph) buildResult(_ context.Context, state *scanBranchGraphState) (*model.ScanBranchResult, error) {
	if state == nil {
		return nil, fmt.Errorf("scan branch graph state is nil")
	}
	riskLevel, hasRisk := parseRiskLevel(state.LLMResult)
	if riskLevel == "high" {
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[build_result] 分支 %s 风险等级: %s (高风险，请人工复核)", state.Input.Branch, riskLevel))
	} else {
		g.appendBranchLog(state.Input.JobID, model.ScanJobLogLevelInfo,
			fmt.Sprintf("[build_result] 分支 %s 风险等级: %s", state.Input.Branch, riskLevel))
	}
	return &model.ScanBranchResult{
		JobID:          state.Input.JobID,
		BranchName:     state.Input.Branch,
		CommitCount:    len(state.Input.Commits),
		CommitMessages: state.CommitsJSON,
		FromCommit:     state.FromCommit,
		ToCommit:       state.ToCommit,
		AnalysisStage:  model.ScanAnalysisStageWithDiff,
		Result:         state.LLMResult,
		HasRisk:        hasRisk,
		RiskLevel:      riskLevel,
	}, nil
}
