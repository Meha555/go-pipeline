package pipeline

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/Meha555/go-pipeline/internal"
)

type Status int

const (
	Unknown Status = iota - 1
	Success
	Failed
	Skiped
)

func (s Status) String() string {
	switch s {
	case Success:
		return "Success"
	case Failed:
		return "Failed"
	case Skiped:
		return "Skiped"
	default:
		return "Unknown"
	}
}

// Job 组织一个可以并发执行的任务
// 因此Job的执行可以认为是没有顺序的概念的，如果需要顺序执行两个Job，则应该让这两个Job分别位于两个Stage中
type Job struct {
	Name         string
	Actions      []*Action
	Hooks        *Hooks
	Timeout      time.Duration
	AllowFailure bool
	resCh        chan Status
	timer        *internal.Timer

	s *Stage
}

type JobOptions func(*Job)

func WithTimeout(timeout time.Duration) JobOptions {
	return func(j *Job) {
		j.Timeout = timeout
	}
}

func WithAllowFailure(allowFailure bool) JobOptions {
	return func(j *Job) {
		j.AllowFailure = allowFailure
	}
}

func WithHooks(hooks *Hooks) JobOptions {
	return func(j *Job) {
		j.Hooks = hooks
	}
}

func NewJob(name string, actions []*Action, s *Stage, opts ...JobOptions) *Job {
	j := &Job{
		Name:         name,
		Actions:      actions,
		Hooks:        &Hooks{},
		resCh:        make(chan Status),
		Timeout:      time.Duration(math.MaxInt64),
		AllowFailure: false,
		timer:        &internal.Timer{},
		s:            s,
	}

	for _, opt := range opts {
		opt(j)
	}

	return j
}

func (j *Job) Do(ctx context.Context) (status Status) {
	defer logger.SetPrefix(logger.Prefix())
	logger.SetPrefix(fmt.Sprintf("job[%s] ", j.Name))
	os.Setenv("JOB_NAME", j.Name)
	status = Success
	// 如果不同步一下，单纯的 <- j.resCh 不能代表Job.Do的执行逻辑走完了，特别是还存在defer的情况下
	defer j.s.wg.Done()

	if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
		j.timer.Start()
		defer func() {
			j.timer.Elapsed()
			logger.Printf("Job %s cost %v", j.Name, j.timer.Elapsed())
		}()
	}

	logger.Printf("Job %s: %d actions", j.Name, len(j.Actions))
	defer func() {
		if status == Failed {
			logger.Printf("Job %s failed", j.Name)
		} else {
			logger.Printf("Job %s success", j.Name)
		}
	}()

	if j.Timeout != time.Duration(math.MaxInt64) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, j.Timeout)
		defer cancel()
	}

	if len(j.Hooks.Before) > 0 {
		if err := j.Hooks.DoBefore(ctx); err != nil {
			logger.Printf("hooks before failed: %v", err)
		}
	}
	defer func() {
		if len(j.Hooks.After) > 0 {
			if err := j.Hooks.DoAfter(ctx); err != nil {
				logger.Printf("hooks after failed: %v", err)
			}
		}
	}()

	for _, action := range j.Actions {
		// 要求Exec是阻塞的
		if err := action.Exec(ctx); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				logger.Printf("action (%s) timeout %v exceeded", action, j.Timeout)
			} else {
				logger.Printf("action (%s) failed: %v", action, err)
			}
			if !j.AllowFailure {
				status = Failed
				break
			}
		}
	}
	j.resCh <- status
	return
}

func (j *Job) Result() <-chan Status {
	return j.resCh
}
