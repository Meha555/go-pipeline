package pipeline

import (
	"context"
	"errors"
	"log"
	"math"
	"time"
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

func NewJob(name string, actions []*Action, opts ...JobOptions) *Job {
	j := &Job{
		Name:         name,
		Actions:      actions,
		resCh:        make(chan Status),
		Timeout:      time.Duration(math.MaxInt64),
		AllowFailure: false,
	}

	for _, opt := range opts {
		opt(j)
	}

	return j
}

// TODO 利用ctx来终止action
func (j *Job) Do(ctx context.Context) Status {
	log.Printf("Job %s: %d actions", j.Name, len(j.Actions))

	var status = Success
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
	return status
}

func (j *Job) Result() <-chan Status {
	return j.resCh
}
