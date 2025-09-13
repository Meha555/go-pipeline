package pipeline

import (
	"context"
	"log"
	"os"
)

// Pipeline 定义流水线结构体
type Pipeline struct {
	Name    string
	Version string
	Envs    map[string]string
	Workdir string
	Stages  []*Stage
}

type PipelineOptions func(*Pipeline)

func WithEnvs(envs map[string]string) PipelineOptions {
	return func(p *Pipeline) {
		p.Envs = envs
	}
}

func WithWorkdir(workdir string) PipelineOptions {
	return func(p *Pipeline) {
		p.Workdir = workdir
	}
}

// NewPipeline 创建新的流水线实例
func NewPipeline(name, version string, opts ...PipelineOptions) *Pipeline {
	p := &Pipeline{
		Name:    name,
		Version: version,
		Envs:    map[string]string{},
		Stages:  []*Stage{},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// AddStage 向流水线添加阶段
func (p *Pipeline) AddStage(stage *Stage) *Pipeline {
	p.Stages = append(p.Stages, stage)
	return p
}

// Run 执行流水线
func (p *Pipeline) Run() Status {
	shell := os.Getenv("SHELL")

	if shell == "" {
		// Windows系统通常没有SHELL环境变量，尝试其他方式
		log.Println("can't get current shell, might by in windows")
	} else {
		log.Printf("current shell: %s", shell)
	}

	for key, value := range p.Envs {
		if err := os.Setenv(key, value); err != nil {
			log.Printf("set env %s=%s for pipeline %s failed: %v", key, value, p.Name, err)
		}
	}

	if err := os.Chdir(p.Workdir); err != nil {
		log.Printf("change workdir to %s failed: %v", p.Workdir, err)
	}

	log.Printf("Pipeline %s @ %s:\n", p.Name, p.Version)
	stageNames := make([]string, len(p.Stages))
	for i, stage := range p.Stages {
		stageNames[i] = stage.Name
	}
	log.Println(stageNames)

	ctx := context.Background()

	for _, stage := range p.Stages {
		if stage.Perform(ctx) == Failed {
			return Failed
		}
	}
	return Success
}
