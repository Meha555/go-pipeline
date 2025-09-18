package pipeline

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Env 表示Pipeline内置的环境变量（不是系统环境变量，类似Jenkins的内置环境变量：${YOUR_JENKINS_HOST}/env-vars.html）
// 主要用途是获取当前Pipeline运行时的一些信息（只是获取，不能修改）
type Env struct {
	Name        string
	Value       string
	Description string
}

func (e *Env) String() string {
	return fmt.Sprintf("%s=%s", e.Name, e.Value)
}

var Builtins = []*Env{
	{
		Name:        "PIPELINE_NAME",
		Description: "Current Pipeline name",
	},
	{
		Name:        "PIPELINE_VERSION",
		Description: "Current Pipeline version",
	},
	{
		Name:        "PIPELINE_TIMESTAMP",
		Description: "Current Pipeline starting timestamp in format 'YYYYMMDDHHMMSS'",
	},
	{
		Name:        "PIPELINE_WORKDIR",
		Description: "Current Pipeline working directory",
	},
	{
		Name:        "STAGE_NAME",
		Description: "Current Stage name",
	},
	{
		Name:        "JOB_NAME",
		Description: "Current Job name",
	},
}

func setupBuiltins(p *Pipeline) {
	Builtins = []*Env{
		{
			Name:        "PIPELINE_NAME",
			Value:       p.Name,
			Description: "Current Pipeline name",
		},
		{
			Name:        "PIPELINE_VERSION",
			Value:       p.Version,
			Description: "Current Pipeline version",
		},
		{
			Name:        "PIPELINE_TIMESTAMP",
			Value:       time.Now().Format("20060102150405"),
			Description: "Current Pipeline starting timestamp in format 'YYYYMMDDHHMMSS'",
		},
		{
			Name:        "PIPELINE_WORKDIR",
			Value:       p.Workdir,
			Description: "Current Pipeline working directory",
		},
		{
			Name:        "STAGE_NAME",
			Value:       "",
			Description: "Current Stage name",
		},
		{
			Name:        "JOB_NAME",
			Value:       "",
			Description: "Current Job name",
		},
	}
	for _, env := range Builtins {
		p.Envs.Append(env.Name, env.Value)
	}
}

type inlineCmd struct {
	cmd      *exec.Cmd
	startPos int // 包括`或者$(的起始位置
	endPos   int // 包括`或者)的结束位置
}

// findInlineCmdImplBackQuote 查找内联命令`cmd`
// offset 表示当前查找的起始位置下标
// str 表示当前查找的字符串
// cmd 表示找到的内联命令
// nextPos 表示下一次应该查找的起始位置下标
func findInlineCmdImplBackQuote(offset int, str string) (cmd *inlineCmd, nextPos int) {
	lenValue := len(str)
	leftBracketPos, rightBracketPos := -1, -1
	if lenValue < 3 {
		return nil, lenValue + offset
	}
	leftBracketPos = strings.IndexByte(str[offset:], '`')
	if leftBracketPos != -1 {
		rightBracketPos = strings.IndexByte(str[offset+leftBracketPos+1:], '`')
		if rightBracketPos != -1 {
			rightBracketPos += leftBracketPos + 1
			cmdLine := str[offset+leftBracketPos+1 : rightBracketPos]
			// found a cmd
			if len(cmdLine) > 0 {
				return &inlineCmd{
					cmd:      exec.Command("sh", "-c", cmdLine),
					startPos: leftBracketPos + offset,
					endPos:   rightBracketPos + offset,
				}, rightBracketPos + offset
			}
		}
	}
	return nil, lenValue + offset + 1
}

// findInlineCmdImplDollar 查找内联命令$(cmd)
// offset 表示当前查找的起始位置下标
// str 表示当前查找的字符串
// cmd 表示找到的内联命令
// nextPos 表示下一次应该查找的起始位置下标
func findInlineCmdImplDollar(offset int, str string) (cmd *inlineCmd, nextPos int) {
	lenValue := len(str)
	leftBracketPos, rightBracketPos := -1, -1
	if lenValue < 4 {
		return nil, lenValue + offset + 1
	}
	dollarPos := strings.IndexByte(str[offset:], '$')
	if dollarPos != -1 && dollarPos < lenValue-2 && str[offset+dollarPos+1] == '(' {
		leftBracketPos = dollarPos + 1
		rightBracketPos = strings.IndexByte(str[offset+leftBracketPos+1:], ')')
		if rightBracketPos != -1 {
			rightBracketPos += leftBracketPos + 1
			cmdLine := str[offset+leftBracketPos+1 : rightBracketPos]
			// found a cmd
			if len(cmdLine) > 0 {
				return &inlineCmd{
					cmd:      exec.Command("sh", "-c", cmdLine),
					startPos: dollarPos + offset,
					endPos:   rightBracketPos + offset,
				}, rightBracketPos + offset
			}
		}
	}
	return nil, lenValue + offset + 1
}

func findInlineCmd(str string) []*inlineCmd {
	var cmds []*inlineCmd
	lenValue := len(str)
	finder := func(f func(offset int, str string) (cmd *inlineCmd, nextPos int)) {
		for i := 0; i < lenValue; i++ {
			cmd, nextPos := f(i, str)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			i = nextPos
		}
	}
	finder(findInlineCmdImplBackQuote)
	finder(findInlineCmdImplDollar)
	return cmds
}
