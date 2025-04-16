package job

import (
	"context"
	"log"
	"time"
)

type Status int

func (s Status) String() string {
	switch s {
	case 0:
		return "成功"
	case 1:
		return "失败"
	case 2:
		return "跳过"
	default:
		return "未知"
	}
}

type JobInterface interface {
	NameStr() string
	Execute(ctx context.Context) bool
	PostAction(ctx context.Context) Status
}

type JobFuntor func(args ...interface{}) bool

// Job 定义任务结构体
type Job struct {
	Name string
	Func JobFuntor
	Args []interface{}
}

func NewJob(name string, f func(args ...interface{}) bool, args ...interface{}) *Job {
	return &Job{
		Name: name,
		Func: JobFuntor(f),
		Args: args,
	}
}

func (j Job) NameStr() string {
	return j.Name
}

func (j Job) Execute(ctx context.Context) bool {
	log.Println("执行任务: ", j.Name)
	timeStart := time.Now()
	defer func() {
		log.Printf("任务 %s 执行耗时: %v\n", j.Name, time.Since(timeStart))
	}()
	return j.Func(j.Args...)
}

func (j Job) PostAction(ctx context.Context) Status {
	return Status(0)
}
