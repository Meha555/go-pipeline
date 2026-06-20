package pipeline

import (
	"context"
	"fmt"
	"log/slog"
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
	Shell   [2]string
	Cron    string

	Envs    EnvList // 为了确保环境变量初始化时按照conf.Envs中切片中的顺序，这里不能采用map
	Workdir string
	Stages  []*Stage

	timer      *internal.Timer
	succeedCnt int

	logger *slog.Logger
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

func WithShell(shell string) PipelineOptions {
	return func(p *Pipeline) {
		if shell == "" {
			return
		}
		p.Shell[0], p.Shell[1] = getShell(shell)
	}
}

func NewPipeline(name, version string, opts ...PipelineOptions) *Pipeline {
	p := &Pipeline{
		Name:    name,
		Version: version,
		Envs:    EnvList{},
		Stages:  []*Stage{},
		timer:   &internal.Timer{},
		logger:  slog.Default().With("pipeline", name, "version", version),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.Shell[0] == "" {
		p.Shell[0], p.Shell[1] = GetDefaultShell()
	}

	if p.Workdir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			p.logger.Error(fmt.Sprintf("get current workdir failed: %v", err), "error", err)
			os.Exit(1)
		}
		p.Workdir = pwd
	}

	return p
}

func getShell(shell string) (cmd, flag string) {
	switch shell {
	case "bash":
		cmd, flag = "bash", "-c"
	case "sh":
		cmd, flag = "sh", "-c"
	case "cmd":
		cmd, flag = "cmd", "/c"
	case "powershell":
		cmd, flag = "powershell", "-Command"
	default:
		panic("Unknown shell: " + shell)
	}
	return
}

func (p *Pipeline) AddStage(stage *Stage) *Pipeline {
	p.Stages = append(p.Stages, stage)
	return p
}

// // 会恢复日志前缀、但不会恢复环境变量。不需要恢复工作目录，因为运行时改动工作目录是脚本中启动的子进程做的，和父进程无关
// var stackRestore = internal.NewStack()

func (p *Pipeline) preRun(context.Context) Status {
	// stackRestore.Push(logger.Prefix())
	// 处理环境变量
	{
		// 初始化内置环境变量
		setupBuiltins(p)
		// 初始化定制环境变量
		p.Envs = resolveEnvList(p.Shell, p.Envs)
		for _, env := range p.Envs {
			if err := os.Setenv(env.Key, env.Value); err != nil {
				p.logger.Error(fmt.Sprintf("set env %s=%s for pipeline %s failed: %v", env.Key, env.Value, p.Name, err), "error", err, "key", env.Key, "value", env.Value)
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
				p.logger.Error(fmt.Sprintf("failed to expr %s: %v", cmd.cmd.String(), err), "error", err, "expr", cmd.cmd.String())
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
				p.logger.Error(fmt.Sprintf("change workdir to %s failed: %v", p.Workdir, err), "error", err, "workdir", p.Workdir)
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
	p.logger.Info(fmt.Sprintf("%s@%s %s (%s): %v", p.Name, p.Version, cronStr, p.Workdir, stageNames), "cron", cronStr, "workdir", p.Workdir, "stages", stageNames)

	work := func() {
		status = Success
		defer func() {
			statistics := fmt.Sprintf("(%d succeed/%d total)", p.succeedCnt, len(p.Stages))
			p.logger.Info(fmt.Sprintf("%s %s", status, statistics), "status", status.String(), "succeed", p.succeedCnt, "total", len(p.Stages))
		}()
		if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
			p.timer.Start()
			defer func() {
				p.logger.Info(fmt.Sprintf("Cost %v", p.timer.Elapsed()), "cost", p.timer.Elapsed())
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
			p.logger.Warn("wait some job to quit for too long, force quit!")
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
