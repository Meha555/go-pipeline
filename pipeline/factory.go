package pipeline

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/Meha555/go-pipeline/parser"
)

func isSkipped(config *parser.PipelineConf, item string) bool {
	return slices.Contains(config.Skips, item)
}

// MakePipeline 根据配置信息创建流水线
func MakePipeline(config *parser.PipelineConf) *Pipeline {
	// 创建流水线
	pipeObj := NewPipeline(config.Name, config.Version, WithShell(config.Shell), WithCron(config.Cron), WithEnvs(config.Envs), WithWorkdir(config.Workdir))

	// 为每个阶段创建 Stage 对象
	stageMap := make(map[string]*Stage)
	for _, stageName := range config.Stages {
		if isSkipped(config, stageName) {
			continue
		}
		stageObj, exists := stageMap[stageName]
		if exists {
			pipeObj.logger.Error(fmt.Sprintf("duplicate stage %q", stageName), "stage", stageName)
			os.Exit(1)
		}
		stageObj = NewStage(stageName, pipeObj)
		stageMap[stageName] = stageObj
		pipeObj.AddStage(stageObj)
	}

	// 处理Job
	for jobName, jobDef := range config.Jobs {
		if parser.IsKeyword(jobName) || isSkipped(config, jobName) || isSkipped(config, jobDef.Stage) {
			continue
		}

		// 检查对应的Stage是否存在
		stageObj, exists := stageMap[jobDef.Stage]
		if !exists {
			// 如果Stage不存在，丢弃Job
			slog.Warn(fmt.Sprintf("job %s belong to undefined stage %s, ignored it", jobName, jobDef.Stage), "job", jobName, "stage", jobDef.Stage)
			continue
		}

		// 创建Job并添加到Stage
		// 1. 创建Actions并添加到Job
		actions := makeActions(pipeObj.Shell, jobDef.Actions)
		// 2. 创建Hooks并添加到Job
		hooks := &Hooks{
			Before: makeActions(pipeObj.Shell, jobDef.Hooks.Before),
			After:  makeActions(pipeObj.Shell, jobDef.Hooks.After),
		}
		jobObj := NewJob(jobName, actions, stageObj, WithAllowFailure(jobDef.AllowFailure), WithJobEnvs(jobDef.Envs), WithRules(jobDef.Rules), WithExports(jobDef.Exports), WithHooks(hooks))
		if jobDef.Timeout != "" {
			if jobTimeout, err := time.ParseDuration(jobDef.Timeout); err == nil {
				jobObj.Timeout = jobTimeout
			}
		}
		stageObj.AddJob(jobObj)
	}

	return pipeObj
}

func makeActions(shell [2]string, actionLines []string) (actions []*Action) {
	for _, actionLine := range actionLines {
		var action *Action
		if shell[0] == "cmd" {
			actionArgs, err := makeSafeCmdline(actionLine)
			if err != nil {
				slog.Error(fmt.Sprintf("invalid action format: %s (error: %v)", actionLine, err), "error", err, "action", actionLine)
				os.Exit(-1)
			}
			if len(actionArgs) > 1 {
				action = NewAction(shell, actionArgs[0], actionArgs[1:]...)
			} else {
				action = NewAction(shell, actionArgs[0])
			}
		} else {
			action = NewAction(shell, actionLine)
		}
		actions = append(actions, action)
	}
	return
}

// 使用'单引号包裹一个带有空格的命令。
// 双引号 "：会解析内部的变量、转义符
// 单引号 '：原义传递所有内容，不会修改字符串
func makeSafeCmdline(rawline string) ([]string, error) {
	var (
		args        []string
		withinQuote bool = false
	)
	l := len(rawline)
	i, j := 0, 0
	for ; j < l && i <= j; j++ {
		c := rawline[j]
		if c == ' ' && !withinQuote {
			if j > i {
				args = append(args, rawline[i:j])
			}
			i = j + 1
		} else if c == '\'' {
			if withinQuote {
				withinQuote = false
				args = append(args, rawline[i:j])
			} else {
				withinQuote = true
			}
			i = j + 1
		}
	}
	if j > i && rawline[j-1] != ' ' {
		if withinQuote && rawline[j-1] == '"' {
			args = append(args, rawline[i:j-1])
		} else if !withinQuote {
			args = append(args, rawline[i:j])
		}
	}
	return args, nil
}
