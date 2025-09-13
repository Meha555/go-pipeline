package pipeline

import (
	"errors"
	"log"
	"strings"

	"github.com/Meha555/go-pipeline/parser"
)

func MakePipeline(config *parser.PipelineConf) *Pipeline {
	// 处理环境变量
	envs := make(map[string]string)
	for _, envLine := range config.Envs {
		if parts := strings.SplitN(envLine, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			envs[key] = value
		} else {
			log.Printf("invalid env format: %s (expected key=value)", envLine)
		}
	}

	// 创建流水线
	pipeObj := NewPipeline(config.Name, config.Version, WithEnvs(envs), WithWorkdir(config.Workdir))

	// 为每个阶段创建 Stage 对象
	stageMap := make(map[string]*Stage)
	for _, stageName := range config.Stages {
		stageObj := NewStage(stageName)
		stageMap[stageName] = stageObj
		pipeObj.AddStage(stageObj)
	}

	// 处理Job
	for jobName, jobDef := range config.Jobs {
		// 跳过内置字段
		if parser.IsKeyword(jobName) {
			continue
		}

		// 检查对应的Stage是否存在
		stageObj, exists := stageMap[jobDef.Stage]
		if !exists {
			// 如果Stage不存在，丢弃Job
			log.Printf("job %s belong to undefined stage %s, ignored it", jobName, jobDef.Stage)
			continue
		}

		// 创建任务并添加到阶段
		var actions []*Action
		for _, actionLine := range jobDef.Actions {
			actions = append(actions, NewAction("sh", "-c", actionLine))
		}
		jobObj := NewJob(jobName, actions, WithAllowFailure(jobDef.AllowFailure))
		if jobTimeout, err := parser.ParseDuration(jobDef.Timeout); err != nil {
			if !errors.Is(err, parser.ErrTimeoutIsEmpty) {
				log.Printf("job %s timeout parse failed: %v, set to +inf", jobName, err)
			}
		} else {
			jobObj.Timeout = jobTimeout
		}
		stageObj.AddJob(jobObj)
	}

	return pipeObj
}
