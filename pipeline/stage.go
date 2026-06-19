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
	for _, existing := range s.Jobs {
		if existing.Name == job.Name {
			s.logger.Error(fmt.Sprintf("duplicate job %q in stage %s", job.Name, s.Name), "job", job.Name, "stage", s.Name)
			os.Exit(1)
		}
	}
	s.Jobs = append(s.Jobs, job)
	return s
}

func (s *Stage) Perform(ctx context.Context) (status Status) {
	// Stage是串行执行的，所以这里不会对当前进程的环境变量表产生并发写入
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
		jobStatus := <-job.Result()
		if jobStatus == Failed {
			s.failedCnt++
			status = Failed
		}
	}
	// 等待所有任务完成
	s.wg.Wait()
	if status == Success {
		for _, job := range s.Jobs {
			if len(job.Exports) == 0 {
				continue
			}
			// 注入环境变量到当前进程
			if err := job.importExports(); err != nil {
				job.logger.Error(fmt.Sprintf("import exports failed: %v", err), "error", err)
				return Failed
			}
		}
	}
	return
}
