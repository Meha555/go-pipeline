package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Meha555/go-pipeline/internal"
)

// Pipeline 定义流水线结构体
type Pipeline struct {
	Name    string
	Version string
	Envs    map[string]string
	Workdir string
	Stages  []*Stage

	timer      *internal.Timer
	succeedCnt int
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
		timer:   &internal.Timer{},
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
func (p *Pipeline) Run(ctx context.Context) (status Status) {
	status = Success
	if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
		p.timer.Start()
		defer func() {
			p.timer.Elapsed()
			log.Printf("Pipeline %s@%s cost %v", p.Name, p.Version, p.timer.Elapsed())
		}()
	}

	defer func() {
		statistics := fmt.Sprintf("(%d succeed/%d total)", p.succeedCnt, len(p.Stages))
		if status == Failed {
			log.Printf("Pipeline %s@%s failed %s", p.Name, p.Version, statistics)
		} else {
			log.Printf("Pipeline %s@%s success %s", p.Name, p.Version, statistics)
		}
	}()

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

	for _, stage := range p.Stages {
		if stage.Perform(ctx) == Failed {
			status = Failed
			return
		}
		p.succeedCnt++
	}
	return
}
