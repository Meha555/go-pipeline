package pipeline

import (
	"context"
	"errors"
	"log"
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

func (s Status) Error() string {
	switch s {
	case 0:
		return "Success"
	case 1:
		return "Failed"
	case 2:
		return "Skiped"
	default:
		return "Unknown"
	}
}

type Job struct {
	Name         string
	Actions      []*Action
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

func NewJob(name string, actions []*Action, s *Stage, opts ...JobOptions) *Job {
	j := &Job{
		Name:         name,
		Actions:      actions,
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
	os.Setenv("JOB_NAME", j.Name)
	status = Success
	// 如果不同步一下，单纯的 <- j.resCh 不能代表Job.Do的执行逻辑走完了，特别是还存在defer的情况下
	defer j.s.wg.Done()

	if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
		j.timer.Start()
		defer func() {
			j.timer.Elapsed()
			log.Printf("Job %s cost %v", j.Name, j.timer.Elapsed())
		}()
	}

	log.Printf("Job %s: %d actions", j.Name, len(j.Actions))
	defer func() {
		if status == Failed {
			log.Printf("Job %s failed", j.Name)
		} else {
			log.Printf("Job %s success", j.Name)
		}
	}()

	if j.Timeout != time.Duration(math.MaxInt64) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, j.Timeout)
		defer cancel()
	}
	for _, action := range j.Actions {
		// 要求Exec是阻塞的
		if err := action.Exec(ctx); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log.Printf("action (%s) timeout %v exceeded", action, j.Timeout)
			} else {
				log.Printf("action (%s) failed: %v", action, err)
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
