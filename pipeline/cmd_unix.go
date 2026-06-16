//go:build !windows

package pipeline

import (
	"context"
	"os/exec"
	"strings"
)

func GetDefaultShell() (cmd, flag string) {
	cmd = "sh"
	flag = "-c"
	return
}

func ShellCommand(shellCmd, shellFlag, name string, args ...string) *exec.Cmd {
	shellArgs := append([]string{name}, args...)
	return exec.Command(shellCmd, shellFlag, strings.Join(shellArgs, " "))
}

func ShellCommandContext(ctx context.Context, shellCmd, shellFlag, name string, args ...string) *exec.Cmd {
	shellArgs := append([]string{name}, args...)
	return exec.CommandContext(ctx, shellCmd, shellFlag, strings.Join(shellArgs, " "))
}
