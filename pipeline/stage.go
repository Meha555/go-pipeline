package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/Meha555/go-pipeline/internal"
)

// Stage 定义阶段结构体
type Stage struct {
	Name string
	Jobs []*Job

	timer     *internal.Timer
	wg        *sync.WaitGroup
	p         *Pipeline
	failedCnt int
	logger    *slog.Logger
}

func NewStage(name string, p *Pipeline) *Stage {
	return &Stage{
		Name:   name,
		Jobs:   make([]*Job, 0),
		p:      p,
		timer:  &internal.Timer{},
		wg:     &sync.WaitGroup{},
		logger: p.logger.With("stage", name),
	}
}

func (s *Stage) AddJob(job *Job) *Stage {
	s.Jobs = append(s.Jobs, job)
	return s
}

func (s *Stage) Perform(ctx context.Context) (status Status) {
	os.Setenv("STAGE_NAME", s.Name)
	status = Success
	if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
		s.timer.Start()
		defer func() {
			s.logger.Info(fmt.Sprintf("Stage@%s cost %v", s.Name, s.timer.Elapsed()), "cost", s.timer.Elapsed())
		}()
	}

	s.logger.Info(fmt.Sprintf("Stage@%s: %d jobs", s.Name, len(s.Jobs)), "jobs", len(s.Jobs))

	if err := os.Chdir(s.p.Workdir); err != nil {
		s.logger.Error(fmt.Sprintf("change workdir to %s failed: %v", s.p.Workdir, err), "error", err, "workdir", s.p.Workdir)
		return Failed
	}

	defer func() {
		statistics := fmt.Sprintf("(%d failed/%d total)", s.failedCnt, len(s.Jobs))
		if status == Failed {
			s.logger.Error(fmt.Sprintf("Stage@%s failed %s", s.Name, statistics), "failed", s.failedCnt, "total", len(s.Jobs))
		} else {
			s.logger.Info(fmt.Sprintf("Stage@%s success %s", s.Name, statistics), "failed", s.failedCnt, "total", len(s.Jobs))
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
