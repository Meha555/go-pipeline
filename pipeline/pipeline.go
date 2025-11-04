package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/Meha555/go-pipeline/parser"
	"github.com/robfig/cron/v3"
)

type EnvList = parser.DictList[string, string]

// Pipeline 定义流水线结构体
type Pipeline struct {
	Name    string
	Version string
	Shell   string
	Cron    string

	Envs    EnvList // 为了确保环境变量初始化时按照conf.Envs中切片中的顺序，这里不能采用map
	Workdir string
	Stages  []*Stage

	timer      *internal.Timer
	succeedCnt int
}

type PipelineOptions func(*Pipeline)

func WithCron(cron string) PipelineOptions {
	return func(p *Pipeline) {
		p.Cron = cron
	}
}

func WithEnvs(envs EnvList) PipelineOptions {
	return func(p *Pipeline) {
		p.Envs = envs
	}
}

func WithWorkdir(workdir string) PipelineOptions {
	return func(p *Pipeline) {
		p.Workdir = workdir
	}
}

func NewPipeline(name, version, shell string, opts ...PipelineOptions) *Pipeline {
	p := &Pipeline{
		Name:    name,
		Version: version,
		Shell:   shell,
		Envs:    EnvList{},
		Stages:  []*Stage{},
		timer:   &internal.Timer{},
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.Workdir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			logger.Printf("get current workdir failed: %v", err)
		}
		p.Workdir = pwd
	}

	return p
}

func (p *Pipeline) AddStage(stage *Stage) *Pipeline {
	p.Stages = append(p.Stages, stage)
	return p
}

// // 会恢复日志前缀、但不会恢复环境变量。不需要恢复工作目录，因为运行时改动工作目录是脚本中启动的子进程做的，和父进程无关
// var stackRestore = internal.NewStack()

func (p *Pipeline) preRun(context.Context) Status {
	// stackRestore.Push(logger.Prefix())
	logger.SetPrefix(fmt.Sprintf("pipeline[%s@%s] ", p.Name, p.Version))
	// 处理环境变量
	{
		// 初始化内置环境变量
		setupBuiltins(p)
		// 初始化定制环境变量
		for i := range p.Envs {
			key := p.Envs[i].Key
			value := p.Envs[i].Value
			// 继续处理value中可能存在的'$'进行变量展开，以及命令的执行
			// 1. 先执行命令
			cmds := findInlineCmd(value, p.Shell)
			for _, cmd := range cmds {
				output, err := cmd.cmd.CombinedOutput()
				if err != nil {
					logger.Printf("failed to expr %s: %v", cmd.cmd.String(), err)
					continue
				}
				// 替换value中cmd.startPos到cmd.endPos的内容为命令的输出
				value = value[:cmd.startPos] + strings.TrimSuffix(string(output), "\n") + value[cmd.endPos+1:]
			}
			// 2. 再执行展开
			if err := os.Setenv(key, os.Expand(value, func(v string) string {
				if val := os.Getenv(v); val != "" {
					return val
				}
				if val, ok := p.Envs.Find(v); ok {
					return val
				}
				return ""
			})); err != nil {
				logger.Printf("set env %s=%s for pipeline %s failed: %v", key, value, p.Name, err)
				return Failed
			}
		}
	}

	// 处理workdir
	{
		cmds := findInlineCmd(p.Workdir, p.Shell)
		for _, cmd := range cmds {
			output, err := cmd.cmd.CombinedOutput()
			if err != nil {
				logger.Printf("failed to expr %s: %v", cmd.cmd.String(), err)
				continue
			}
			// 替换Workdir中cmd.startPos到cmd.endPos的内容为命令的输出
			p.Workdir = p.Workdir[:cmd.startPos] + strings.TrimSuffix(string(output), "\n") + p.Workdir[cmd.endPos+1:]
		}
		// 处理workdir中的环境变量展开
		p.Workdir = os.Expand(p.Workdir, func(v string) string {
			if val := os.Getenv(v); val != "" {
				return val
			}
			if val, ok := p.Envs.Find(v); ok {
				return val
			}
			return ""
		})

		if p.Workdir != "" {
			if err := os.Chdir(p.Workdir); err != nil {
				logger.Printf("change workdir to %s failed: %v", p.Workdir, err)
				return Failed
			}
		}
	}
	return Success
}

func (p *Pipeline) postRun(context.Context) Status {
	// if prefix, err := stackRestore.Pop(); err == nil {
	// 	logger.SetPrefix(prefix.(string))
	// }
	return Success
}

func (p *Pipeline) run(ctx context.Context) (status Status) {
	stageNames := make([]string, len(p.Stages))
	for i, stage := range p.Stages {
		stageNames[i] = stage.Name
	}

	var cronStr string
	var cronDaemon *cron.Cron
	if p.Cron != "" {
		cronStr = fmt.Sprintf("{%s}", p.Cron)
		cronDaemon = cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	}
	logger.Printf("%s (%s): %v", cronStr, p.Workdir, stageNames)

	work := func() {
		status = Success
		defer func() {
			statistics := fmt.Sprintf("(%d succeed/%d total)", p.succeedCnt, len(p.Stages))
			logger.Printf("%s %s", status, statistics)
		}()
		if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
			p.timer.Start()
			defer func() {
				p.timer.Elapsed()
				logger.Printf("Cost %v", p.timer.Elapsed())
			}()
		}
		for _, stage := range p.Stages {
			if stage.Perform(ctx) == Failed {
				status = Failed
				return
			}
			p.succeedCnt++
		}
	}

	if cronDaemon != nil {
		cronDaemon.AddFunc(p.Cron, work) // 失败的任务仍然会继续执行
		cronDaemon.Start()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-sigChan
		c := cronDaemon.Stop()
		select {
		case <-c.Done():
		case <-time.After(5 * time.Second):
			logger.Println("wait some job to quit for too long, force quit!")
		}
	} else {
		work()
	}
	return
}

func (p *Pipeline) Run(ctx context.Context) (status Status) {
	defer p.postRun(ctx)
	if status = p.preRun(ctx); status != Success {
		return
	}
	status = p.run(ctx)
	return
}

func getShell(shell string) (cmd, flag string) {
	switch shell {
	case "bash":
		cmd, flag = "bash", "-c"
	case "sh":
		cmd, flag = "sh", "-c"
	case "cmd":
		cmd, flag = "cmd", "/c"
	}
	return
}
