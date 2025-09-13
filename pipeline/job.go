package pipeline

import (
	"context"
	"log"
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
	Name    string
	Actions []*Action
	resCh   chan Status
}

func NewJob(name string, actions ...*Action) *Job {
	return &Job{
		Name:    name,
		Actions: actions,
		resCh:   make(chan Status),
	}
}

// TODO 利用ctx来终止action
func (j *Job) Do(ctx context.Context) Status {
	var status = Success
	for _, action := range j.Actions {
		// 要求Exec是阻塞的
		if err := action.Exec(ctx); err != nil {
			log.Printf("action %s exec failed with %s", action, err)
			status = Failed
			break
		}
	}
	j.resCh <- status
	return status
}

func (j *Job) Result() <-chan Status {
	return j.resCh
}
