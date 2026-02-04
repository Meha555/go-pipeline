//go:build linux

package pipeline

import (
	"context"
	"os/exec"
	"strings"
)

func GetShell() (cmd, flag string) {
	cmd = "sh"
	flag = "-c"
	return
}

func ShellCommand(name string, args ...string) *exec.Cmd {
	shellCmd, shellFlag := GetShell()
	shellArgs := append([]string{name}, args...)
	return exec.Command(shellCmd, shellFlag, strings.Join(shellArgs, " "))
}

func ShellCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	shellCmd, shellFlag := GetShell()
	shellArgs := append([]string{name}, args...)
	return exec.CommandContext(ctx, shellCmd, shellFlag, strings.Join(shellArgs, " "))
}
