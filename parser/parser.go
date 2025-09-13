package parser

import (
	"errors"
	"fmt"
	"go-pipeline/pipeline"
	"log"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// keywords
const (
	keywordName    = "name"
	keywordVersion = "version"
	keywordEnvs    = "envs"
	keywordWorkdir = "workdir"

	keywordStages = "stages"

	keywordJobs         = "jobs"
	keywordStage        = "stage"
	keywordActions      = "actions"
	keywordTimeout      = "timeout"
	keywordAllowFailure = "allow_failure"
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
	keywordTimeout,
	keywordAllowFailure,
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
	Stage        string   `yaml:"stage"`
	Actions      []string `yaml:"actions"`
	Timeout      string   `yaml:"timeout,omitempty"`
	AllowFailure bool     `yaml:"allow_failure,omitempty"`
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
		var actions []*pipeline.Action
		for _, actionLine := range jobDef.Actions {
			actions = append(actions, pipeline.NewAction("sh", "-c", actionLine))
		}
		jobObj := pipeline.NewJob(jobName, actions, pipeline.WithAllowFailure(jobDef.AllowFailure))
		if jobTimeout, err := parseDuration(jobDef.Timeout); err != nil {
			if !errors.Is(err, ErrTimeoutIsEmpty) {
				log.Printf("job %s timeout parse failed: %v, set to +inf", jobName, err)
			}
		} else {
			jobObj.Timeout = jobTimeout
		}
		stageObj.AddJob(jobObj)
	}

	return p, nil
}

var (
	ErrTimeoutIsEmpty = errors.New("timeout is empty")
	ErrTimeoutNoUnit  = errors.New("timeout has no unit")
	ErrTimeoutValue   = errors.New("invalid timeout value")
	ErrTimeoutUnit    = errors.New("invalid timeout unit, supported units are ms, s, m, h, d")
)

// parseDuration 解析带单位的时间字符串为time.Duration
func parseDuration(duration string) (time.Duration, error) {
	if duration == "" {
		return time.Duration(math.MaxInt64), ErrTimeoutIsEmpty
	}

	unitIndex := -1
	for i := 0; i < len(duration); i++ {
		if duration[i] < '0' || duration[i] > '9' {
			unitIndex = i
			break
		}
	}
	if unitIndex == -1 {
		return time.Duration(math.MaxInt64), ErrTimeoutNoUnit
	}

	// 分割数值和单位
	valueStr := duration[:unitIndex]
	unit := strings.ToLower(duration[unitIndex:])

	// 解析数值
	value, err := strconv.Atoi(valueStr)
	if err != nil || value < 0 {
		return time.Duration(math.MaxInt64), fmt.Errorf("%w: %v", ErrTimeoutValue, err)
	}

	// 根据单位转换为对应的Duration
	switch unit {
	case "ms":
		return time.Duration(value) * time.Millisecond, nil
	case "s":
		return time.Duration(value) * time.Second, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrTimeoutUnit, unit)
	}
}
