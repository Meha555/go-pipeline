package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/Meha555/go-pipeline/internal"
)

// Action 一次性动作
type Action struct {
	Cmd    string
	Args   []string
	busy   bool
	stdout io.ReadCloser
	stderr io.ReadCloser
}

var ErrActionBusy = fmt.Errorf("action is busy because is has not finished")

func NewAction(cmd string, args ...string) *Action {
	return &Action{
		Cmd:  cmd,
		Args: args,
		busy: false,
	}
}

func (a *Action) prepare(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, a.Cmd, a.Args...)
	// a.stdout, _ = cmd.StdoutPipe()
	// a.stderr, _ = cmd.StderrPipe()
	return cmd
}

// Exec 阻塞地执行动作
func (a *Action) Exec(ctx context.Context) (err error) {
	cmd := a.prepare(ctx)
	if a.busy {
		return ErrActionBusy
	}
	defer func() {
		a.busy = false
	}()
	a.busy = true
	if noSilence, ok := ctx.Value(internal.NoSilenceKey).(bool); ok && noSilence {
		logger.Printf("exec action: %s", a.String())
	}
	if dryRun, ok := ctx.Value(internal.DryRunKey).(bool); ok && dryRun {
		logger.Println(a.String())
		return
	}
	// 由于scanner.Scan()可能在cmd.Wait()关闭管道写端后继续读取而导致报错"file already closed"。这点在cmd.StdoutPipe()的文档中有说明。这里显式等待输出完成后再等待命令执行完成。
	wg := &sync.WaitGroup{}
	if verbose, ok := ctx.Value(internal.VerboseKey).(bool); ok && verbose {
		a.stdout, _ = cmd.StdoutPipe()
		a.stderr, _ = cmd.StderrPipe()
		wg.Add(2)
		go readOutput(wg, a.stdout, os.Stdout)
		go readOutput(wg, a.stderr, os.Stderr)
		// } else {
		// 	// 即使不显示输出，也要读取并丢弃输出，防止管道写端阻塞而导致当前goroutine卡死
		// 	go readOutput(a.stdout, io.Discard)
		// 	go readOutput(a.stderr, io.Discard)
	}

	err = cmd.Start()
	if err == nil {
		wg.Wait()
		err = cmd.Wait()
	}
	return
}

func (a *Action) String() string {
	return fmt.Sprintf("%s %s", a.Cmd, strings.Join(a.Args, " "))
}

func readOutput(wg *sync.WaitGroup, reader io.Reader, out io.Writer) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		io.WriteString(out, scanner.Text())
		io.WriteString(out, "\n")
	}
	if err := scanner.Err(); err != nil { // 说明不是io.EOF
		logger.Printf("read output error: %v", err)
	}
}
