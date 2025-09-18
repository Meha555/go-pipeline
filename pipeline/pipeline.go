package pipeline

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/Meha555/go-pipeline/parser"
)

type EnvList = parser.DictList[string, string]

// Pipeline 定义流水线结构体
type Pipeline struct {
	Name    string
	Version string
	Envs    EnvList // 为了确保环境变量初始化时按照conf.Envs中切片中的顺序，这里不能采用map
	Workdir string
	Stages  []*Stage

	timer      *internal.Timer
	succeedCnt int
}

type PipelineOptions func(*Pipeline)

func WithEnvs(envs EnvList) PipelineOptions {
	return func(p *Pipeline) {
		p.Envs = envs
	}
}

func WithWorkdir(workdir string) PipelineOptions {
	return func(p *Pipeline) {
		p.Workdir = workdir
	}
}

func NewPipeline(name, version string, opts ...PipelineOptions) *Pipeline {
	p := &Pipeline{
		Name:    name,
		Version: version,
		Envs:    EnvList{},
		Stages:  []*Stage{},
		timer:   &internal.Timer{},
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.Workdir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			logger.Printf("get current workdir failed: %v", err)
		}
		p.Workdir = pwd
	}

	return p
}

func (p *Pipeline) AddStage(stage *Stage) *Pipeline {
	p.Stages = append(p.Stages, stage)
	return p
}

// NOTE 不需要恢复工作目录和环境变量，因为程序执行完就退出了
func (p *Pipeline) Run(ctx context.Context) (status Status) {
	defer logger.SetPrefix(logger.Prefix())
	logger.SetPrefix(fmt.Sprintf("pipeline[%s@%s] ", p.Name, p.Version))
	status = Success
	if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
		p.timer.Start()
		defer func() {
			p.timer.Elapsed()
			logger.Printf("Pipeline %s@%s cost %v", p.Name, p.Version, p.timer.Elapsed())
		}()
	}

	defer func() {
		statistics := fmt.Sprintf("(%d succeed/%d total)", p.succeedCnt, len(p.Stages))
		if status == Failed {
			logger.Printf("Pipeline %s@%s failed %s", p.Name, p.Version, statistics)
		} else {
			logger.Printf("Pipeline %s@%s success %s", p.Name, p.Version, statistics)
		}
	}()
	// 初始化内置环境变量
	setupBuiltins(p)
	// 初始化定制环境变量
	for i := range p.Envs {
		key := p.Envs[i].Key
		value := p.Envs[i].Value
		// 继续处理value中可能存在的'$'进行变量展开，以及命令的执行
		// 1. 先执行命令
		cmds := findInlineCmd(value)
		for _, cmd := range cmds {
			output, err := cmd.cmd.CombinedOutput()
			if err != nil {
				logger.Printf("failed to expr %s: %v", cmd.cmd.String(), err)
				continue
			}
			// 替换value中cmd.startPos到cmd.endPos的内容为命令的输出
			value = value[:cmd.startPos] + strings.TrimSuffix(string(output), "\n") + value[cmd.endPos+1:]
		}
		// 2. 再执行展开
		if err := os.Setenv(key, os.Expand(value, func(v string) string {
			if val := os.Getenv(v); val != "" {
				return val
			}
			if val, ok := p.Envs.Find(v); ok {
				return val
			}
			return ""
		})); err != nil {
			logger.Printf("set env %s=%s for pipeline %s failed: %v", key, value, p.Name, err)
		}
	}

	if p.Workdir != "" {
		if err := os.Chdir(p.Workdir); err != nil {
			logger.Printf("change workdir to %s failed: %v", p.Workdir, err)
		}
	}

	stageNames := make([]string, len(p.Stages))
	for i, stage := range p.Stages {
		stageNames[i] = stage.Name
	}
	logger.Printf("Pipeline %s@%s (%s): %v", p.Name, p.Version, p.Workdir, stageNames)

	for _, stage := range p.Stages {
		if stage.Perform(ctx) == Failed {
			status = Failed
			return
		}
		p.succeedCnt++
	}
	return
}
