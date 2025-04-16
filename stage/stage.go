package stage

import "go-pipeline/job"

// Stage 定义阶段结构体
type Stage struct {
	Name string
	Jobs []job.JobInterface
}

func NewStage(name string) *Stage {
	return &Stage{
		Name: name,
		Jobs: make([]job.JobInterface, 0),
	}
}

func (s *Stage) AddJob(job job.JobInterface) *Stage {
	s.Jobs = append(s.Jobs, job)
	return s
}