package pipeline

import (
	"context"
	"go-pipeline/stage"
	"log"
	"time"
)

// Pipeline 定义流水线结构体
type Pipeline struct {
	Name   string
	Stages []stage.Stage
}

// NewPipeline 创建新的流水线实例
func NewPipeline(name string) *Pipeline {
	return &Pipeline{
		Name:   name,
		Stages: []stage.Stage{},
	}
}

// AddStage 向流水线添加阶段
func (p *Pipeline) AddStage(stage stage.Stage) *Pipeline {
	p.Stages = append(p.Stages, stage)
	return p
}

// Run 执行流水线
func (p *Pipeline) Run() bool {
	log.Printf("流水线 %s:\n", p.Name)
	stageNames := make([]string, len(p.Stages))
	for i, stage := range p.Stages {
		stageNames[i] = stage.Name
	}
	log.Println(stageNames)

	start := time.Now()
	defer func() {
		log.Printf("%s 耗时: %v\n", p.Name, time.Since(start))
	}()

	ctx := context.Background()

	for idx, stage := range p.Stages {
		log.Printf("阶段 %d/%d: %s\n", idx+1, len(p.Stages), stage.Name)
		for jobIdx, job := range stage.Jobs {
			log.Printf("  任务 %d/%d: %s\n", jobIdx+1, len(stage.Jobs), job.NameStr())
			jobStart := time.Now()
			if !job.Execute(ctx) {
				log.Printf("任务 %s 失败\n", job.NameStr())
				return false
			}
			log.Printf("  任务 %s 耗时: %v\n", job.NameStr(), time.Since(jobStart))
		}
	}
	return true
}
