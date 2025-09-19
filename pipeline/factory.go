package pipeline

import (
	"errors"
	"log"
	"os"
	"slices"
	"strings"

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
	pipeObj := NewPipeline(config.Name, config.Version, WithEnvs(envs), WithWorkdir(config.Workdir))

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
		if jobTimeout, err := parser.ParseDuration(jobDef.Timeout); err != nil {
			if !errors.Is(err, parser.ErrTimeoutIsEmpty) {
				logger.Printf("job %s timeout parse failed: %v, set to +inf", jobName, err)
			}
		} else {
			jobObj.Timeout = jobTimeout
		}
		stageObj.AddJob(jobObj)
	}

	return pipeObj
}

func makeActions(actionLines []string) (actions []*Action) {
	for _, actionLine := range actionLines {
		actions = append(actions, NewAction("sh", "-c", actionLine))
	}
	return
}

var logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
