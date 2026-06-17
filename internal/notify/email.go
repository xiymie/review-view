package notify

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"gopkg.in/gomail.v2"
	"review-view/internal/model"
)

// mdRenderer 启用 GFM（表格、删除线、自动链接等），与前端 marked.js 行为对齐
var mdRenderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	TLS      bool
}

// SMTPConfigProvider 每次发送时动态获取最新配置，避免启动时读取后失效
type SMTPConfigProvider func() SMTPConfig

type EmailNotifier struct {
	getConfig SMTPConfigProvider
}

func NewEmailNotifier(provider SMTPConfigProvider) *EmailNotifier {
	return &EmailNotifier{getConfig: provider}
}

func (n *EmailNotifier) Send(task *model.Task, project *model.Project, user *model.User) error {
	cfg := n.getConfig()
	if cfg.Host == "" || cfg.From == "" {
		return nil
	}
	if user.NotifyEmails == "" {
		return nil
	}

	recipients := splitEmails(user.NotifyEmails)
	if len(recipients) == 0 {
		return nil
	}

	subject := buildEmailSubject(task, project)
	htmlBody, err := mdToHTML(task.Result)
	if err != nil {
		htmlBody = "<pre>" + escapeHTML(task.Result) + "</pre>"
	}

	m := gomail.NewMessage()
	if cfg.FromName != "" {
		m.SetAddressHeader("From", cfg.From, cfg.FromName)
	} else {
		m.SetHeader("From", cfg.From)
	}
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", wrapHTML(subject, buildMeta(task, project, user), htmlBody))

	if task.Result != "" {
		filename := fmt.Sprintf("review-%s.md", safeCommit(task.ToCommit))
		content := []byte(task.Result)
		m.Attach(
			filename,
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(content)
				return err
			}),
			gomail.SetHeader(map[string][]string{
				"Content-Type": {"text/markdown; charset=UTF-8"},
			}),
		)
	}

	port := cfg.Port
	if port == 0 {
		if cfg.TLS {
			port = 465
		} else {
			port = 587
		}
	}
	d := gomail.NewDialer(cfg.Host, port, cfg.Username, cfg.Password)
	d.SSL = cfg.TLS
	return d.DialAndSend(m)
}

func buildMeta(task *model.Task, project *model.Project, user *model.User) string {
	finishedAt := time.Now().Format("2006-01-02 15:04:05")
	if task.FinishedAt != nil {
		finishedAt = task.FinishedAt.Format("2006-01-02 15:04:05")
	}

	statusText := map[model.TaskStatus]string{
		model.TaskStatusCompleted: "完成",
		model.TaskStatusFailed:   "失败",
	}
	status := statusText[task.Status]
	if status == "" {
		status = string(task.Status)
	}

	from := safeCommit(task.FromCommit)
	if from == "" {
		from = "—"
	}

	return fmt.Sprintf(`<table style="border-collapse:collapse;width:100%%;margin-bottom:24px;font-size:13px">
<tbody>
<tr><td style="padding:6px 12px;background:#f6f8fa;border:1px solid #e1e4e8;font-weight:600;width:120px;white-space:nowrap">审计完成时间</td><td style="padding:6px 12px;border:1px solid #e1e4e8">%s</td></tr>
<tr><td style="padding:6px 12px;background:#f6f8fa;border:1px solid #e1e4e8;font-weight:600">审计人</td><td style="padding:6px 12px;border:1px solid #e1e4e8">%s</td></tr>
<tr><td style="padding:6px 12px;background:#f6f8fa;border:1px solid #e1e4e8;font-weight:600">项目</td><td style="padding:6px 12px;border:1px solid #e1e4e8">%s</td></tr>
<tr><td style="padding:6px 12px;background:#f6f8fa;border:1px solid #e1e4e8;font-weight:600">Commit 范围</td><td style="padding:6px 12px;border:1px solid #e1e4e8;font-family:monospace">%s → %s</td></tr>
<tr><td style="padding:6px 12px;background:#f6f8fa;border:1px solid #e1e4e8;font-weight:600">审计状态</td><td style="padding:6px 12px;border:1px solid #e1e4e8">%s</td></tr>
</tbody></table>
<hr style="border:none;border-top:1px solid #eaecef;margin:0 0 20px">`,
		escapeHTML(finishedAt),
		escapeHTML(user.Username),
		escapeHTML(project.Name),
		escapeHTML(from), escapeHTML(safeCommit(task.ToCommit)),
		escapeHTML(status),
	)
}

func splitEmails(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildEmailSubject(task *model.Task, project *model.Project) string {
	status := "完成"
	if task.Status == model.TaskStatusFailed {
		status = "失败"
	}
	return fmt.Sprintf("[Code Review %s] %s - %s", status, project.Name, safeCommit(task.ToCommit))
}

func mdToHTML(md string) (string, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func wrapHTML(title, meta, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<title>%s</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;max-width:860px;margin:0 auto;padding:24px;color:#24292e}
pre,code{background:#f6f8fa;border-radius:4px;padding:2px 6px;font-size:0.9em}
pre{padding:12px;overflow-x:auto}
h1,h2,h3{border-bottom:1px solid #eaecef;padding-bottom:.3em}
table{border-collapse:collapse;width:100%%;margin:16px 0;font-size:13px;display:block;overflow-x:auto}
table th,table td{border:1px solid #d0d7de;padding:6px 12px;text-align:left;vertical-align:top}
table th{background:#f6f8fa;font-weight:600;white-space:nowrap}
table tr:nth-child(2n){background:#f6f8fa}
</style>
</head><body>%s%s</body></html>`, escapeHTML(title), meta, body)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func safeCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}

// ParseSMTPConfig 从原始字符串构建 SMTPConfig
func ParseSMTPConfig(host, portStr, username, password, from, fromName, tlsStr string) SMTPConfig {
	port, _ := strconv.Atoi(portStr)
	return SMTPConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		FromName: fromName,
		TLS:      tlsStr == "true" || tlsStr == "1",
	}
}

// SendTestEmail 使用给定配置发送一封测试邮件
func SendTestEmail(cfg SMTPConfig, to ...string) error {
	if cfg.Host == "" {
		return fmt.Errorf("SMTP Host 未配置")
	}
	if cfg.From == "" {
		return fmt.Errorf("发件人地址（From）未配置")
	}
	if len(to) == 0 {
		return fmt.Errorf("收件地址不能为空")
	}

	m := gomail.NewMessage()
	if cfg.FromName != "" {
		m.SetAddressHeader("From", cfg.From, cfg.FromName)
	} else {
		m.SetHeader("From", cfg.From)
	}
	m.SetHeader("To", to...)
	subject := "Review View 邮件推送配置确认"
	m.SetHeader("Subject", subject)
	body := `<p>您好，</p>
<p>这是一封来自 Review View 的配置确认邮件。如果您收到本邮件，说明邮件推送功能已正确配置，后续的代码审查结果将通过此通道发送。</p>
<p>无需回复本邮件。</p>`
	m.SetBody("text/html", wrapHTML(subject, "", body))

	port := cfg.Port
	if port == 0 {
		if cfg.TLS {
			port = 465
		} else {
			port = 587
		}
	}
	d := gomail.NewDialer(cfg.Host, port, cfg.Username, cfg.Password)
	// TLS=true: SSL/TLS（465）; TLS=false: STARTTLS（587）
	d.SSL = cfg.TLS
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("SMTP %s:%d → %w", cfg.Host, port, err)
	}
	return nil
}
