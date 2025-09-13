package pipeline

import (
	"context"
	"log"
)

// Stage 定义阶段结构体
type Stage struct {
	Name string
	Jobs []*Job
}

func NewStage(name string) *Stage {
	return &Stage{
		Name: name,
		Jobs: make([]*Job, 0),
	}
}

func (s *Stage) AddJob(job *Job) *Stage {
	s.Jobs = append(s.Jobs, job)
	return s
}

func (s *Stage) Perform(ctx context.Context) Status {
	log.Printf("Stage %s: %d jobs", s.Name, len(s.Jobs))

	for _, job := range s.Jobs {
		go job.Do(ctx)
	}
	var status Status = Success
	// 等待所有任务完成
	for _, job := range s.Jobs {
		status = <-job.Result()
	}
	return status
}
