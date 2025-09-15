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

	"github.com/Meha555/go-pipeline/internal"
)

// Action 一次性动作
type Action struct {
	Cmd    string
	Args   []string
	valid  bool
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
	// a.stdout, _ = cmd.StdoutPipe()
	// a.stderr, _ = cmd.StderrPipe()
	return cmd
}

// Exec 阻塞地执行动作
func (a *Action) Exec(ctx context.Context) (err error) {
	cmd := a.prepare(ctx)
	if cmd == nil {
		err = ErrActionInvalid
		return
	}

	if dryRun, ok := ctx.Value(internal.DryRunKey).(bool); ok && dryRun {
		log.Println(a.String())
		return
	}
	if verbose, ok := ctx.Value(internal.VerboseKey).(bool); ok && verbose {
		a.stdout, _ = cmd.StdoutPipe()
		a.stderr, _ = cmd.StderrPipe()
		go readOutput(a.stdout, os.Stdout)
		go readOutput(a.stderr, os.Stderr)
		// } else {
		// 	// 即使不显示输出，也要读取并丢弃输出，防止管道写端阻塞而导致当前goroutine卡死
		// 	go readOutput(a.stdout, io.Discard)
		// 	go readOutput(a.stderr, io.Discard)
	}

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
