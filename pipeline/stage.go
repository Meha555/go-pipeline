package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/Meha555/go-pipeline/internal"
)

// Stage 定义阶段结构体
type Stage struct {
	Name string
	Jobs []*Job

	failedCnt int

	timer *internal.Timer
	wg    *sync.WaitGroup
	p     *Pipeline
}

func NewStage(name string, p *Pipeline) *Stage {
	return &Stage{
		Name:  name,
		Jobs:  make([]*Job, 0),
		p:     p,
		timer: &internal.Timer{},
		wg:    &sync.WaitGroup{},
	}
}

func (s *Stage) AddJob(job *Job) *Stage {
	s.Jobs = append(s.Jobs, job)
	return s
}

func (s *Stage) Perform(ctx context.Context) (status Status) {
	status = Success
	if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
		s.timer.Start()
		defer func() {
			s.timer.Elapsed()
			log.Printf("Stage %s cost %v", s.Name, s.timer.Elapsed())
		}()
	}

	log.Printf("Stage %s: %d jobs", s.Name, len(s.Jobs))

	if err := os.Chdir(s.p.Workdir); err != nil {
		log.Printf("change workdir to %s failed: %v", s.p.Workdir, err)
		return Failed
	}

	defer func() {
		statistics := fmt.Sprintf("(%d failed/%d total)", s.failedCnt, len(s.Jobs))
		if status == Failed {
			log.Printf("Stage %s failed %s", s.Name, statistics)
		} else {
			log.Printf("Stage %s success %s", s.Name, statistics)
		}
	}()

	for _, job := range s.Jobs {
		s.wg.Add(1)
		go job.Do(ctx)
	}
	// 收集结果
	for _, job := range s.Jobs {
		status = <-job.Result()
		if status == Failed {
			s.failedCnt++
		}
	}
	// 等待所有任务完成
	s.wg.Wait()
	return
}
