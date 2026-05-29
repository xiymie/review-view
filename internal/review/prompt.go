package review

import "fmt"

func BuildCLIPrompt(basePrompt, fromCommit, toCommit string) string {
	if fromCommit == "" {
		return fmt.Sprintf(
			"%s\n\n请审查此仓库中最新一次提交 (%s) 的代码变更。使用 git show %s 查看变更内容。",
			basePrompt,
			toCommit,
			toCommit,
		)
	}

	return fmt.Sprintf(
		"%s\n\n请审查此仓库中 %s 到 %s 之间的代码变更。使用 git diff %s..%s 查看变更内容。",
		basePrompt,
		fromCommit,
		toCommit,
		fromCommit,
		toCommit,
	)
}
