//go:build windows

package pipeline

import (
	"context"
	"os/exec"
)

func GetShell() (cmd, flag string) {
	cmd = "cmd"
	flag = "/c"
	return
}

func ShellCommand(name string, args ...string) *exec.Cmd {
	shellCmd, shellFlag := GetShell()
	shellArgs := append([]string{shellFlag, name}, args...)
	return exec.Command(shellCmd, shellArgs...)
}

func ShellCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	shellCmd, shellFlag := GetShell()
	shellArgs := append([]string{shellFlag, name}, args...)
	return exec.CommandContext(ctx, shellCmd, shellArgs...)
}
