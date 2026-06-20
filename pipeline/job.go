package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	Envs         EnvList
	Rules        []Rule
	Exports      EnvList
	Hooks        *Hooks
	Timeout      time.Duration
	AllowFailure bool
	resCh        chan Status
	timer        *internal.Timer
	logger       *slog.Logger

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

func WithExports(exports EnvList) JobOptions {
	return func(j *Job) {
		j.Exports = exports
	}
}

func WithJobEnvs(envs EnvList) JobOptions {
	return func(j *Job) {
		j.Envs = envs
	}
}

func WithRules(rules []Rule) JobOptions {
	return func(j *Job) {
		j.Rules = rules
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
		logger:       s.logger.With("job", name),
		s:            s,
	}

	for _, opt := range opts {
		opt(j)
	}

	return j
}

func (j *Job) Do(ctx context.Context) (status Status) {
	status = Success
	// 如果不同步一下，单纯的 <- j.resCh 不能代表Job.Do的执行逻辑走完了，特别是还存在defer的情况下
	defer j.s.wg.Done()

	if trace, ok := ctx.Value(internal.TraceKey).(bool); ok && trace {
		j.timer.Start()
		defer func() {
			j.logger.Info(fmt.Sprintf("Job@%s cost %v", j.Name, j.timer.Elapsed()), "cost", j.timer.Elapsed())
		}()
	}

	j.logger.Info(fmt.Sprintf("Job@%s: %d actions", j.Name, len(j.Actions)), "actions", len(j.Actions))
	defer func() {
		switch status {
		case Failed:
			j.logger.Error(fmt.Sprintf("Job@%s failed", j.Name))
		case Skiped:
			j.logger.Info(fmt.Sprintf("Job@%s skipped by rules", j.Name))
		case Success:
			j.logger.Info(fmt.Sprintf("Job@%s success", j.Name))
		default:
			j.logger.Error(fmt.Sprintf("Job@%s finished with status: %s", j.Name, status))
		}
	}()

	if j.Timeout != time.Duration(math.MaxInt64) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, j.Timeout)
		defer cancel()
	}

	// 向Job中的Actions/Hooks注入环境变量。不能直接给当前进程注入，因为Job是并发执行的，在Job.Do中修改。
	jobEnv := j.buildEnv()
	applyActionEnvs(j.Hooks.Before, jobEnv)
	applyActionEnvs(j.Actions, jobEnv)
	applyActionEnvs(j.Hooks.After, jobEnv)

	// 检查Job的rules
	if len(j.Rules) > 0 && !j.matchRules(ctx, jobEnv) {
		status = Skiped
		j.resCh <- status
		return
	}

	if len(j.Hooks.Before) > 0 {
		if err := j.Hooks.DoBefore(ctx); err != nil {
			j.logger.Error(fmt.Sprintf("hooks before failed: %v", err), "error", err)
		}
	}
	for _, action := range j.Actions {
		// 要求Exec是阻塞的
		if err := action.Exec(ctx); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				j.logger.Error(fmt.Sprintf("action (%s) timeout %v exceeded", action, j.Timeout), "action", action.String(), "timeout", j.Timeout)
			} else {
				j.logger.Error(fmt.Sprintf("action (%s) failed: %v", action, err), "error", err, "action", action.String())
			}
			if !j.AllowFailure {
				status = Failed
				break
			}
		}
	}
	if len(j.Hooks.After) > 0 {
		if err := j.Hooks.DoAfter(ctx); err != nil {
			j.logger.Error(fmt.Sprintf("hooks after failed: %v", err), "error", err)
		}
	}
	j.resCh <- status
	return
}

func (j *Job) buildEnv() []string {
	// 初始化job的环境变量（往pipeline的环境变量列表中覆盖）
	builtin := EnvList{{Key: "JOB_NAME", Value: j.Name}}
	resolved := resolveEnvList(j.s.p.Shell, j.Envs, builtin, j.s.p.Envs)
	result := make([]string, 0, len(builtin)+len(resolved))
	for _, env := range builtin {
		result = append(result, envLine(env.Key, env.Value))
	}
	for _, env := range resolved {
		result = append(result, envLine(env.Key, env.Value))
	}
	return result
}

func envLine(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func applyActionEnvs(actions []*Action, envs []string) {
	for _, action := range actions {
		action.SetEnvs(envs)
	}
}

func (j *Job) Result() <-chan Status {
	return j.resCh
}

func (j *Job) importExports() error {
	resolved := resolveEnvList(j.s.p.Shell, j.Exports, j.s.p.Envs)
	seen := make(map[string]struct{})
	for _, env := range resolved {
		if _, exists := seen[env.Key]; exists {
			slog.Warn(fmt.Sprintf("export variable %s is overwritten", env.Key), "key", env.Key)
		}
		seen[env.Key] = struct{}{}
		if err := os.Setenv(env.Key, env.Value); err != nil {
			return fmt.Errorf("set export %s failed: %w", env.Key, err)
		}
	}
	return nil
}
