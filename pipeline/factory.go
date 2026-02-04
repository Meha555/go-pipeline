package pipeline

import (
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Meha555/go-pipeline/parser"
)

func isSkipped(config *parser.PipelineConf, item string) bool {
	return slices.Contains(config.Skips, item)
}

// MakePipeline 根据配置信息创建流水线
func MakePipeline(config *parser.PipelineConf) *Pipeline {
	// 处理环境变量
	var envs EnvList
	for _, envLine := range config.Envs {
		if parts := strings.SplitN(envLine, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// envs[key] = os.Expand(value, func(v string) string {
			// 	if val := os.Getenv(v); val != "" {
			// 		return val
			// 	}
			// 	return envs[v]
			// })
			// 注意此时value中可能包含$变量以及命令需要执行，需要在后续展开。选择在后续展开是因为builtin环境变量的初始化在后面
			envs.Append(key, value)
		} else {
			logger.Printf("invalid env format: %s (expected key=value)", envLine)
		}
	}

	// 创建流水线
	pipeObj := NewPipeline(config.Name, config.Version, WithCron(config.Cron), WithEnvs(envs), WithWorkdir(config.Workdir))

	// 为每个阶段创建 Stage 对象
	stageMap := make(map[string]*Stage)
	for _, stageName := range config.Stages {
		if isSkipped(config, stageName) {
			continue
		}
		stageObj, exists := stageMap[stageName]
		if !exists {
			stageObj = NewStage(stageName, pipeObj)
			stageMap[stageName] = stageObj
		}
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
			logger.Printf("job %s belong to undefined stage %s, ignored it", jobName, jobDef.Stage)
			continue
		}

		// 创建Job并添加到Stage
		// 1. 创建Actions并添加到Job
		actions := makeActions(jobDef.Actions)
		// 2. 创建Hooks并添加到Job
		hooks := &Hooks{
			Before: makeActions(jobDef.Hooks.Before),
			After:  makeActions(jobDef.Hooks.After),
		}
		jobObj := NewJob(jobName, actions, stageObj, WithAllowFailure(jobDef.AllowFailure), WithHooks(hooks))
		if jobDef.Timeout != "" {
			if jobTimeout, err := time.ParseDuration(jobDef.Timeout); err == nil {
				jobObj.Timeout = jobTimeout
			}
		}
		stageObj.AddJob(jobObj)
	}

	return pipeObj
}

func makeActions(actionLines []string) (actions []*Action) {
	for _, actionLine := range actionLines {
		actionArgs, err := makeSafeCmdline(actionLine)
		if err != nil {
			logger.Printf("invalid action format: %s (error: %v)", actionLine, err)
			os.Exit(-1)
		}
		var action *Action
		if len(actionArgs) > 1 {
			action = NewAction(actionArgs[0], actionArgs[1:]...)
		} else {
			action = NewAction(actionArgs[0])
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

var logger *log.Logger

func init() {
	var logFlags int
	if os.Getenv("PIPELINE_LOG_TIMESTAMP") == "1" {
		logFlags |= log.LstdFlags | log.Lmicroseconds
	}
	logger = log.New(os.Stderr, "", logFlags)
}
