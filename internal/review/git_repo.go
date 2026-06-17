package review

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"review-view/internal/model"
)

type GitRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
}

type RepositoryManager struct {
	baseDir string
	runner  GitRunner
}

func NewRepositoryManager(baseDir string, runner GitRunner) *RepositoryManager {
	if runner == nil {
		runner = execGitRunner{}
	}
	return &RepositoryManager{
		baseDir: baseDir,
		runner:  runner,
	}
}

// BaseDir 返回仓库存储根目录
func (m *RepositoryManager) BaseDir() string {
	return m.baseDir
}

func (m *RepositoryManager) EnsureRepo(ctx context.Context, projectID int64, repoURL, branch string, cred *model.RepoCredential) (string, error) {
	cloneURL := repoURL
	if cred != nil {
		cloneURL = injectCredentials(repoURL, cred.Username, cred.Password)
	}

	repoDir := filepath.Join(m.baseDir, fmt.Sprintf("%d", projectID))
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		if _, err := m.runner.Run(ctx, "", "git", "clone", "--branch", branch, cloneURL, repoDir); err != nil {
			return "", err
		}
		return repoDir, nil
	}

	// 已有仓库：更新 remote URL（凭据可能变更）
	if cred != nil {
		if _, err := m.runner.Run(ctx, repoDir, "git", "remote", "set-url", "origin", cloneURL); err != nil {
			return "", err
		}
	}

	if _, err := m.runner.Run(ctx, repoDir, "git", "fetch", "origin", branch); err != nil {
		return "", err
	}
	return repoDir, nil
}

// Checkout 将仓库工作目录切换到指定 commit，使 cat/grep 等工具能读到目标版本的完整代码。
func (m *RepositoryManager) Checkout(ctx context.Context, repoDir, commit string) error {
	_, err := m.runner.Run(ctx, repoDir, "git", "checkout", commit)
	return err
}

func (m *RepositoryManager) ResolveHeadCommit(ctx context.Context, repoDir, branch string) (string, error) {
	output, err := m.runner.Run(ctx, repoDir, "git", "rev-parse", "origin/"+branch)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (m *RepositoryManager) BuildDiff(ctx context.Context, repoDir, fromCommit, toCommit string) (string, error) {
	if fromCommit == "" {
		return m.runner.Run(ctx, repoDir, "git", "show", toCommit)
	}
	return m.runner.Run(ctx, repoDir, "git", "diff", fromCommit+".."+toCommit)
}

// GitGrep 在仓库已跟踪文件中固定字符串匹配 word，返回 "file:line:content" 行。
// 仅搜索已跟踪文件，天然排除 .git 及 .gitignore 忽略的内容（node_modules/vendor 等）。
// git grep 无匹配时退出码为 1，此处归一化为「空结果、无错误」。
func (m *RepositoryManager) GitGrep(ctx context.Context, repoDir, word string) (string, error) {
	out, err := m.runner.Run(ctx, repoDir, "git", "grep", "-n", "-F", "-e", word, "--", ".")
	if err != nil {
		// 无匹配（退出码 1）时输出为空，视为正常
		if strings.TrimSpace(out) == "" {
			return "", nil
		}
		return out, err
	}
	return out, nil
}

// BuildDiffNameStatus 获取 commit 范围内的变更文件列表（name-status 格式）。
// fromCommit 为空时使用 git show --name-status 仅展示 toCommit 的变更。
func (m *RepositoryManager) BuildDiffNameStatus(ctx context.Context, repoDir, fromCommit, toCommit string) (string, error) {
	if fromCommit == "" {
		return m.runner.Run(ctx, repoDir, "git", "show", "--name-status", "--pretty=", toCommit)
	}
	return m.runner.Run(ctx, repoDir, "git", "diff", "--name-status", fromCommit+".."+toCommit)
}

// BuildCommitLog 获取 commit 范围内的 commit 记录（oneline 格式）。
// fromCommit 为空时仅展示 toCommit 这一条 commit。
func (m *RepositoryManager) BuildCommitLog(ctx context.Context, repoDir, fromCommit, toCommit string) (string, error) {
	if fromCommit == "" {
		return m.runner.Run(ctx, repoDir, "git", "log", "--oneline", "-1", toCommit)
	}
	return m.runner.Run(ctx, repoDir, "git", "log", "--oneline", fromCommit+".."+toCommit)
}

// CommitSubject 获取单个 commit 的提交说明（subject，即首行）。
func (m *RepositoryManager) CommitSubject(ctx context.Context, repoDir, commit string) (string, error) {
	if commit == "" {
		return "", nil
	}
	out, err := m.runner.Run(ctx, repoDir, "git", "log", "--format=%s", "-1", commit)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// CommitInfo 表示一条 commit 记录的摘要信息
type CommitInfo struct {
	Hash    string    `json:"sha"`
	Message string    `json:"message"`
	Date    time.Time `json:"date"`
	Author  string    `json:"author"`
}

// ListCommits 获取指定分支最近 limit 条 commit 记录
func (m *RepositoryManager) ListCommits(ctx context.Context, repoDir, branch string, limit int) ([]CommitInfo, error) {
	output, err := m.runner.Run(ctx, repoDir, "git", "log", "--pretty=format:%H|%s|%ai|%an", "-n", fmt.Sprintf("%d", limit), "origin/"+branch)
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		date, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[2])
		commits = append(commits, CommitInfo{
			Hash:    parts[0],
			Message: parts[1],
			Date:    date,
			Author:  parts[3],
		})
	}
	return commits, nil
}

// RemoveRepo 删除项目对应的本地仓库目录。目录不存在时不报错。
func (m *RepositoryManager) RemoveRepo(projectID int64) error {
	repoDir := filepath.Join(m.baseDir, fmt.Sprintf("%d", projectID))
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(repoDir)
}

// injectCredentials 将用户名密码注入 HTTPS URL 的 userinfo 部分。
// SSH URL 或空凭据时原样返回。
func injectCredentials(repoURL, username, password string) string {
	if username == "" && password == "" {
		return repoURL
	}
	if !strings.HasPrefix(repoURL, "https://") && !strings.HasPrefix(repoURL, "http://") {
		return repoURL
	}

	prefix := "https://"
	if strings.HasPrefix(repoURL, "http://") {
		prefix = "http://"
	}
	rest := repoURL[len(prefix):]

	// 去掉已有的 userinfo
	if atIdx := strings.Index(rest, "@"); atIdx != -1 {
		rest = rest[atIdx+1:]
	}

	return prefix + url.PathEscape(username) + ":" + password + "@" + rest
}

type execGitRunner struct{}

func (execGitRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}
