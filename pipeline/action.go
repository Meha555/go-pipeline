package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Action 一次性动作
type Action struct {
	Cmd   string
	Args  []string
	valid bool
	stdout io.ReadCloser
	stderr io.ReadCloser
}

var (
	ErrActionInvalid = fmt.Errorf("action is invalid because is has been executed")
)

func NewAction(cmd string, args ...string) *Action {
	return &Action{
		Cmd:   cmd,
		Args:  args,
		valid: true,
	}
}

func (a *Action) prepare(ctx context.Context) *exec.Cmd {
	defer func() {
		a.valid = false
	}()
	if !a.valid {
		return nil
	}
	cmd := exec.CommandContext(ctx, a.Cmd, a.Args...)
	a.stdout, _ = cmd.StdoutPipe()
	a.stderr, _ = cmd.StderrPipe()
	return cmd
}

// Exec 阻塞地执行动作
func (a *Action) Exec(ctx context.Context) (err error) {
	cmd := a.prepare(ctx)
	if cmd == nil {
		err = ErrActionInvalid
		return
	}
	go readOutput(a.stdout, os.Stdout)
	go readOutput(a.stderr, os.Stderr)
	err = cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}
	return
}

func (a *Action) String() string {
	return fmt.Sprintf("%s %s", a.Cmd, strings.Join(a.Args, " "))
}

func readOutput(reader io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		io.WriteString(out, scanner.Text())
		io.WriteString(out, "\n")
	}
	if err := scanner.Err(); err != nil {
		log.Printf("read output error: %v", err)
	}
}
