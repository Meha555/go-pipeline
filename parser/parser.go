package parser

import (
	"fmt"
	"go-pipeline/pipeline"
	"log"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// keywords
const (
	keywordName    = "name"
	keywordVersion = "version"
	keywordEnvs    = "envs"
	keywordWorkdir = "workdir"

	keywordStages = "stages"

	keywordJobs    = "jobs"
	keywordStage   = "stage"
	keywordActions = "actions"
)

var keywordMap = []string{
	keywordName,
	keywordVersion,
	keywordEnvs,
	keywordWorkdir,
	keywordStages,
	keywordJobs,
	keywordStage,
	keywordActions,
}

type PipelineConf struct {
	Name    string             `yaml:"name"`
	Version string             `yaml:"version"`
	Envs    []string           `yaml:"envs,omitempty"`
	Workdir string             `yaml:"workdir,omitempty"`
	Stages  []string           `yaml:"stages"`
	Jobs    map[string]JobConf `yaml:",inline"`
}

type JobConf struct {
	Stage   string   `yaml:"stage"`
	Actions []string `yaml:"actions"`
}

// ParseConfigFile 解析 YAML 配置文件并返回 Pipeline 对象
func ParseConfigFile(configPath string) (*pipeline.Pipeline, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}

	// 读取配置文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// 解析配置文件
	config := &PipelineConf{}
	if err := yaml.Unmarshal(content, config); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

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
	p := pipeline.NewPipeline(config.Name, config.Version, pipeline.WithEnvs(envs), pipeline.WithWorkdir(config.Workdir))

	// 为每个阶段创建 Stage 对象
	stageMap := make(map[string]*pipeline.Stage)
	for _, stageName := range config.Stages {
		stageObj := pipeline.NewStage(stageName)
		stageMap[stageName] = stageObj
		p.AddStage(stageObj)
	}

	// 处理Job
	for jobName, jobDef := range config.Jobs {
		// 跳过内置字段
		if slices.Contains(keywordMap, jobName) {
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
		// jobArgs := []interface{}{jobName, config.Workdir, jobDef.Actions, exportVars}
		var actions []*pipeline.Action
		for _, actionLine := range jobDef.Actions {
			actions = append(actions, pipeline.NewAction("sh", "-c", actionLine))
		}
		jobObj := pipeline.NewJob(jobName, actions...)
		stageObj.AddJob(jobObj)
	}

	return p, nil
}
