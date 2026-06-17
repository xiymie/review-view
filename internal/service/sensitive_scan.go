package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// sensitiveHit 一条敏感词命中记录
type sensitiveHit struct {
	Word    string
	File    string
	Line    int
	Snippet string
}

const maxSnippetLen = 200 // 单条片段最大字符数
const maxHitsPerWord = 50 // 单个词最多展示命中数，防止报告过长

// scanSensitiveWords 在 repoDir（已 checkout 到目标 commit）中扫描所有检测类敏感词，
// 通过 git grep 仅搜索已跟踪文件。
// 返回 (命中列表, 是否配置了检测词, error)：未配置检测词时 configured=false，调用方据此跳过拼接。
func (s *Scheduler) scanSensitiveWords(ctx context.Context, repoDir string) (hits []sensitiveHit, configured bool, err error) {
	if s.sensitiveWords == nil {
		return nil, false, nil
	}
	words, err := s.sensitiveWords.ListDetect()
	if err != nil {
		return nil, false, err
	}

	for _, w := range words {
		if w.Original == "" {
			continue
		}
		configured = true
		out, err := s.repoManager.GitGrep(ctx, repoDir, w.Original)
		if err != nil {
			return nil, configured, fmt.Errorf("git grep %q: %w", w.Original, err)
		}
		hits = append(hits, parseGrepOutput(w.Original, out)...)
	}
	return hits, configured, nil
}

// parseGrepOutput 解析 git grep -n 输出（file:line:content），按词聚合命中。
func parseGrepOutput(word, out string) []sensitiveHit {
	var hits []sensitiveHit
	count := 0
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		// 格式：path:lineno:content
		first := strings.IndexByte(line, ':')
		if first < 0 {
			continue
		}
		rest := line[first+1:]
		second := strings.IndexByte(rest, ':')
		if second < 0 {
			continue
		}
		lineNo, err := strconv.Atoi(rest[:second])
		if err != nil {
			continue
		}
		if count >= maxHitsPerWord {
			break
		}
		count++
		hits = append(hits, sensitiveHit{
			Word:    word,
			File:    line[:first],
			Line:    lineNo,
			Snippet: truncateSnippet(strings.TrimSpace(rest[second+1:])),
		})
	}
	return hits
}

func truncateSnippet(s string) string {
	r := []rune(s)
	if len(r) > maxSnippetLen {
		return string(r[:maxSnippetLen]) + "…"
	}
	return s
}

// buildSensitiveReport 把命中列表渲染为 Markdown 段落，拼接到审核报告头部。
func buildSensitiveReport(hits []sensitiveHit) string {
	var b strings.Builder
	b.WriteString("## 敏感词检测结果\n\n")
	if len(hits) == 0 {
		b.WriteString("✅ 未发现敏感词。\n")
		return b.String()
	}
	b.WriteString(fmt.Sprintf("⚠️ 共发现 %d 处敏感词命中：\n\n", len(hits)))
	b.WriteString("| 文件 | 行号 | 命中词 | 代码片段 |\n")
	b.WriteString("|------|------|--------|----------|\n")
	for _, h := range hits {
		b.WriteString(fmt.Sprintf("| %s | %d | %s | `%s` |\n",
			escapeMarkdownCell(h.File),
			h.Line,
			escapeMarkdownCell(h.Word),
			escapeMarkdownCode(h.Snippet),
		))
	}
	b.WriteString("\n")
	return b.String()
}

// escapeMarkdownCell 转义会破坏表格的字符
func escapeMarkdownCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// escapeMarkdownCode 用于行内代码单元格：换行压平、转义 | 和反引号
func escapeMarkdownCode(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "`", "'")
	return s
}
