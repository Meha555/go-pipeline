package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Meha555/go-pipeline/parser"
)

type Rule = parser.RuleConf
type RuleOn = parser.RuleOn

func (j *Job) matchRules(ctx context.Context, envs []string) bool {
	for _, rule := range j.Rules {
		if j.matchRule(ctx, rule, envs) {
			return true
		}
	}
	return false
}

func (j *Job) matchRule(ctx context.Context, rule Rule, envs []string) bool {
	if rule.On.Default {
		return true
	}
	if rule.On.Bool != nil {
		return *rule.On.Bool
	}
	condition := strings.TrimSpace(rule.On.Value)
	if condition == "" {
		return false
	}
	// 如果是条件是变量，则直接验证变量值
	if name, ok := variableReferenceName(condition); ok {
		return truthy(lookupEnv(name, envs))
	}
	// 否则认为条件是shell命令，以shell命令执行结果作为条件值
	return j.runRuleCommand(ctx, condition, envs)
}

func (j *Job) runRuleCommand(ctx context.Context, command string, envs []string) bool {
	cmd := ShellCommandContext(ctx, j.s.p.Shell[0], j.s.p.Shell[1], command)
	cmd.Env = append(os.Environ(), envs...)
	cmd.Dir = j.s.p.Workdir
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			j.logger.Warn(fmt.Sprintf("rule command %q failed to start: %v", command, err), "command", command, "error", err)
		}
		return false
	}
	return true
}

/*
检查条件是否是这样的形式：
rules:
  - on: $RUN_JOB
  - on: ${RUN_JOB}

一旦条件是这样的形式，则不应当作shell命令解释，
否则其值会被认为是shell命令，出现找不到的命令的错误。
*/
func variableReferenceName(cond string) (string, bool) {
	if strings.HasPrefix(cond, "${") && strings.HasSuffix(cond, "}") {
		name := strings.TrimSuffix(strings.TrimPrefix(cond, "${"), "}")
		return name, isValidEnvKey(name)
	}
	if !strings.HasPrefix(cond, "$") {
		return "", false
	}
	name := strings.TrimPrefix(cond, "$")
	return name, isValidEnvKey(name)
}

func lookupEnv(name string, overrides []string) string {
	for i := len(overrides) - 1; i >= 0; i-- {
		key, value, ok := strings.Cut(overrides[i], "=")
		if ok && key == name {
			return value
		}
	}
	return os.Getenv(name)
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
