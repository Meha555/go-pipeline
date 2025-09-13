package pipeline

import (
	"context"
	"fmt"
	"log"
)

// Stage 定义阶段结构体
type Stage struct {
	Name string
	Jobs []*Job

	failedCnt int
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

func (s *Stage) Perform(ctx context.Context) (status Status) {
	status = Success
	log.Printf("Stage %s: %d jobs", s.Name, len(s.Jobs))
	defer func() {
		statistics := fmt.Sprintf("(%d failed/%d total)", s.failedCnt, len(s.Jobs))
		if status == Failed {
			log.Printf("Stage %s failed %s", s.Name, statistics)
		} else {
			log.Printf("Stage %s success %s", s.Name, statistics)
		}
	}()

	for _, job := range s.Jobs {
		go job.Do(ctx)
	}
	// 等待所有任务完成
	for _, job := range s.Jobs {
		status = <-job.Result()
		if status == Failed {
			s.failedCnt++
		}
	}
	return
}
