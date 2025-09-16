package pipeline

import (
	"fmt"
	"time"
)

// Env 表示Pipeline内置的环境变量（不是系统环境变量，类似Jenkins的内置环境变量：${YOUR_JENKINS_HOST}/env-vars.html）
// 主要用途是获取当前Pipeline运行时的一些信息（只是获取，不能修改）
type Env struct {
	Name        string
	Value       string
	Description string
}

func (e *Env) String() string {
	return fmt.Sprintf("%s=%s", e.Name, e.Value)
}

var Builtins = []*Env{
	{
		Name:        "PIPELINE_NAME",
		Description: "Current Pipeline name",
	},
	{
		Name:        "PIPELINE_VERSION",
		Description: "Current Pipeline version",
	},
	{
		Name:        "PIPELINE_TIMESTAMP",
		Description: "Current Pipeline starting timestamp in format 'YYYYMMDDHHMMSS'",
	},
	{
		Name:        "PIPELINE_WORKDIR",
		Description: "Current Pipeline working directory",
	},
	{
		Name:        "STAGE_NAME",
		Description: "Current Stage name",
	},
	{
		Name:        "JOB_NAME",
		Description: "Current Job name",
	},
}

func setupBuiltins(p *Pipeline) {
	Builtins = []*Env{
		{
			Name:        "PIPELINE_NAME",
			Value:       p.Name,
			Description: "Current Pipeline name",
		},
		{
			Name:        "PIPELINE_VERSION",
			Value:       p.Version,
			Description: "Current Pipeline version",
		},
		{
			Name:        "PIPELINE_TIMESTAMP",
			Value:       time.Now().Format("20060102150405"),
			Description: "Current Pipeline starting timestamp in format 'YYYYMMDDHHMMSS'",
		},
		{
			Name:        "PIPELINE_WORKDIR",
			Value:       p.Workdir,
			Description: "Current Pipeline working directory",
		},
		{
			Name:        "STAGE_NAME",
			Value:       "",
			Description: "Current Stage name",
		},
		{
			Name:        "JOB_NAME",
			Value:       "",
			Description: "Current Job name",
		},
	}
	for _, env := range Builtins {
		p.Envs[env.Name] = env.Value
	}
}
