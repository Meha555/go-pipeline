package pipeline

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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
		Name:        "PIPELINE_SHELL",
		Description: "Current Pipeline shell",
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
	{
		Name:        "OS",
		Description: "Current OS",
	},
	{
		Name:        "ARCH",
		Description: "Current Architecture",
	},
	{
		Name:        "CPU_NUM",
		Description: "Number of CPU Core(s)",
	},
	{
		Name:        "TEMP_DIR",
		Description: "Temporary directory",
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
			Name:        "PIPELINE_SHELL",
			Value:       p.Shell[0],
			Description: "Current Pipeline shell",
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
		{
			Name:        "OS",
			Value:       runtime.GOOS,
			Description: "Current OS",
		},
		{
			Name:        "ARCH",
			Value:       runtime.GOARCH,
			Description: "Current Architecture",
		},
		{
			Name:        "CPU_NUM",
			Value:       strconv.Itoa(runtime.NumCPU()),
			Description: "Number of logical CPU(s)",
		},
		{
			Name:        "TEMP_DIR",
			Value:       os.TempDir(),
			Description: "Temporary directory",
		},
	}
	for _, env := range Builtins {
		// 必须用 Prepand ，这样才能确保内置变量放在p.Envs最前头，
		// 从而使得自定义变量可以引用内置变量
		p.Envs.Prepand(env.Name, env.Value)
	}
}

// 处理环境变量
func resolveEnvList(shell [2]string, envs EnvList, bases ...EnvList) EnvList {
	resolved := EnvList{}
	for _, env := range envs {
		key := env.Key
		value := env.Value
		// 1. 执行value中可能包含的$变量以及`命令`
		cmds := findInlineCmd(value, shell)
		for _, cmd := range cmds {
			output, err := cmd.cmd.CombinedOutput()
			if err != nil {
				slog.Error(fmt.Sprintf("failed to expr %s: %v", cmd.cmd.String(), err), "error", err, "expr", cmd.cmd.String())
				continue
			}
			value = value[:cmd.startPos] + strings.TrimSuffix(string(output), "\n") + value[cmd.endPos+1:]
		}
		// 2. 将命令执行结果展开
		value = os.Expand(value, func(v string) string {
			// envs比bases的查找优先级更高
			if val, ok := resolved.Find(v); ok {
				return val
			}
			for _, base := range bases {
				if val, ok := base.Find(v); ok {
					return val
				}
			}
			// 最后再回退到环境变量
			if val := os.Getenv(v); val != "" {
				return val
			}
			return ""
		})
		resolved.Append(key, value)
	}
	return resolved
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
func findInlineCmdImplBackQuote(offset int, str string, shell [2]string) (cmd *inlineCmd, nextPos int) {
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
					cmd:      exec.Command(shell[0], shell[1], cmdLine),
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
func findInlineCmdImplDollar(offset int, str string, shell [2]string) (cmd *inlineCmd, nextPos int) {
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
					cmd:      exec.Command(shell[0], shell[1], cmdLine),
					startPos: dollarPos + offset,
					endPos:   rightBracketPos + offset,
				}, rightBracketPos + offset
			}
		}
	}
	return nil, lenValue + offset + 1
}

func findInlineCmd(str string, shell [2]string) []*inlineCmd {
	var cmds []*inlineCmd
	lenValue := len(str)
	finder := func(f func(offset int, str string, shell [2]string) (cmd *inlineCmd, nextPos int)) {
		for i := 0; i < lenValue; i++ {
			cmd, nextPos := f(i, str, shell)
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
