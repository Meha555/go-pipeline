//go:build windows

package pipeline

import (
	"context"
	"os/exec"
)

func GetDefaultShell() (cmd, flag string) {
	cmd = "cmd"
	flag = "/c"
	return
}

func ShellCommand(shellCmd, shellFlag, name string, args ...string) *exec.Cmd {
	shellArgs := append([]string{shellFlag, name}, args...)
	return exec.Command(shellCmd, shellArgs...)
}

func ShellCommandContext(ctx context.Context, shellCmd, shellFlag, name string, args ...string) *exec.Cmd {
	shellArgs := append([]string{shellFlag, name}, args...)
	return exec.CommandContext(ctx, shellCmd, shellArgs...)
}
