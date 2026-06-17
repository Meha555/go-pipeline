package pipeline

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/Meha555/go-pipeline/internal/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestNewPipelineSetsDefaultShell(t *testing.T) {
	defaultShell, defaultFlag := GetDefaultShell()
	p := NewPipeline("test", "1.0.0")

	if p.Shell != [2]string{defaultShell, defaultFlag} {
		t.Fatalf("Shell = %#v, want %#v", p.Shell, [2]string{defaultShell, defaultFlag})
	}
}

func TestNewPipelineWithEmptyShellUsesDefaultShell(t *testing.T) {
	defaultShell, defaultFlag := GetDefaultShell()
	p := NewPipeline("test", "1.0.0", WithShell(""))

	if p.Shell != [2]string{defaultShell, defaultFlag} {
		t.Fatalf("Shell = %#v, want %#v", p.Shell, [2]string{defaultShell, defaultFlag})
	}
}

func TestNewPipelineSupportsPowerShell(t *testing.T) {
	p := NewPipeline("test", "1.0.0", WithShell("powershell"))

	if p.Shell != [2]string{"powershell", "-Command"} {
		t.Fatalf("Shell = %#v, want powershell command shell", p.Shell)
	}
}

func TestPipelineLoggersCarryInheritedContext(t *testing.T) {
	var buf bytes.Buffer
	withPipelineTestLogger(t, &buf)

	p := NewPipeline("release", "2.0.0")
	s := NewStage("build", p)
	j := NewJob("compile", nil, s)

	p.logger.Info("pipeline log")
	s.logger.Info("stage log")
	j.logger.Info("job log")

	logs := buf.String()
	for _, want := range []string{
		`"pipeline":"release"`,
		`"version":"2.0.0"`,
		`"stage":"build"`,
		`"job":"compile"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs = %q, want %q", logs, want)
		}
	}
}

func withPipelineTestLogger(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	oldDefault := slog.Default()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
		slog.SetDefault(oldDefault)
	})
	if err := logging.Configure(logging.Options{Format: logging.FormatJSON, Level: logging.LevelInfo, Writer: buf}); err != nil {
		t.Fatalf("configure logger: %v", err)
	}
}
