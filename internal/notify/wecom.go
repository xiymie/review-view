package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"review-view/internal/model"
)

type WecomNotifier struct{}

func NewWecomNotifier() *WecomNotifier {
	return &WecomNotifier{}
}

func (n *WecomNotifier) Send(task *model.Task, project *model.Project, user *model.User) error {
	if user.NotifyWecomWebhook == "" {
		return nil
	}

	content := buildWecomContent(task, project)
	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(user.NotifyWecomWebhook, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wecom webhook returned %d", resp.StatusCode)
	}
	return nil
}

func buildWecomContent(task *model.Task, project *model.Project) string {
	status := "✅ 完成"
	if task.Status == model.TaskStatusFailed {
		status = "❌ 失败"
	}

	commit := task.ToCommit
	if len(commit) > 8 {
		commit = commit[:8]
	}

	msg := fmt.Sprintf("## Code Review %s\n\n**项目：** %s\n**Commit：** `%s`",
		status, project.Name, commit)

	if task.Status == model.TaskStatusFailed && task.ErrorMessage != "" {
		msg += fmt.Sprintf("\n**错误：** %s", task.ErrorMessage)
	} else if task.Result != "" {
		// 企业微信 markdown 内容有长度限制（4096），摘要前 500 字
		preview := task.Result
		if len(preview) > 500 {
			preview = preview[:500] + "\n\n...（完整结果请登录系统查看）"
		}
		msg += "\n\n" + preview
	}

	return msg
}
