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

	if p.Workdir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			log.Printf("get current workdir failed: %v", err)
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
func (p *Pipeline) Run() Status {
	for key, value := range p.Envs {
		if err := os.Setenv(key, value); err != nil {
			log.Printf("set env %s=%s for pipeline %s failed: %v", key, value, p.Name, err)
		}
	}

	if p.Workdir != "" {
		if err := os.Chdir(p.Workdir); err != nil {
			log.Printf("change workdir to %s failed: %v", p.Workdir, err)
		}
	}

	stageNames := make([]string, len(p.Stages))
	for i, stage := range p.Stages {
		stageNames[i] = stage.Name
	}
	log.Printf("Pipeline %s@%s (%s): %v", p.Name, p.Version, p.Workdir, stageNames)

	ctx := context.Background()

	for _, stage := range p.Stages {
		if stage.Perform(ctx) == Failed {
			return Failed
		}
	}
	return Success
}
