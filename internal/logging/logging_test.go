package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestResolveOptionsPrecedence(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv(EnvLogFormat, "")
		t.Setenv(EnvLogLevel, "")
		t.Setenv(EnvLogColor, "")

		opts := ResolveOptions("", false, "", false, "", false)
		if opts.Format != FormatConsole || opts.Level != LevelInfo || opts.Color != ColorAuto {
			t.Fatalf("ResolveOptions() = %#v, want console/info/auto", opts)
		}
	})

	t.Run("environment", func(t *testing.T) {
		t.Setenv(EnvLogFormat, FormatJSON)
		t.Setenv(EnvLogLevel, LevelDebug)
		t.Setenv(EnvLogColor, ColorNever)

		opts := ResolveOptions("", false, "", false, "", false)
		if opts.Format != FormatJSON || opts.Level != LevelDebug || opts.Color != ColorNever {
			t.Fatalf("ResolveOptions() = %#v, want json/debug/never", opts)
		}
	})

	t.Run("cli overrides environment", func(t *testing.T) {
		t.Setenv(EnvLogFormat, FormatJSON)
		t.Setenv(EnvLogLevel, LevelError)
		t.Setenv(EnvLogColor, ColorNever)

		opts := ResolveOptions(FormatConsole, true, LevelDebug, true, ColorAuto, true)
		if opts.Format != FormatConsole || opts.Level != LevelDebug || opts.Color != ColorAuto {
			t.Fatalf("ResolveOptions() = %#v, want console/debug/auto", opts)
		}
	})
}

func TestConfigureRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{name: "format", opts: Options{Format: "plain", Level: LevelInfo}, want: "invalid log format"},
		{name: "level", opts: Options{Format: FormatJSON, Level: "verbose"}, want: "invalid log level"},
		{name: "color", opts: Options{Format: FormatConsole, Level: LevelInfo, Color: "sometimes"}, want: "invalid log color"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.opts.Writer = &buf
			err := Configure(tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Configure() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestConfigureOutputContracts(t *testing.T) {
	t.Run("json keeps structured fields and filters by level", func(t *testing.T) {
		var buf bytes.Buffer
		withTestLogger(t, Options{Format: FormatJSON, Level: LevelInfo, Writer: &buf})

		slog.Debug("hidden")
		slog.Warn("warning: key \"build_job.actions\"", "key", "build_job.actions")

		got := buf.String()
		if strings.Contains(got, "hidden") {
			t.Fatalf("logs = %q, want debug suppressed", got)
		}
		for _, want := range []string{`"level":"warn"`, `"key":"build_job.actions"`, `"time":`, `warning: key \"build_job.actions\"`} {
			if !strings.Contains(got, want) {
				t.Fatalf("logs = %q, want %q", got, want)
			}
		}
	})

	t.Run("console omits structured fields", func(t *testing.T) {
		var buf bytes.Buffer
		withTestLogger(t, Options{Format: FormatConsole, Level: LevelDebug, Color: ColorAuto, Writer: &buf})

		slog.Debug("load config /repo/config.yaml", "pipeline", "release", "version", "2.0.0", "stage", "build", "job", "compile", "path", "/repo/config.yaml")

		got := buf.String()
		for _, want := range []string{"T", "+", "DBG load config /repo/config.yaml"} {
			if !strings.Contains(got, want) {
				t.Fatalf("logs = %q, want %q", got, want)
			}
		}
		for _, unwanted := range []string{"pipeline=", "version=", "stage=", "job=", "path=", "%!s(<nil>)", "|", "\x1b["} {
			if strings.Contains(got, unwanted) {
				t.Fatalf("logs = %q, want no %q", got, unwanted)
			}
		}
	})
}

func withTestLogger(t *testing.T, opts Options) {
	t.Helper()
	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	oldDefault := slog.Default()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
		slog.SetDefault(oldDefault)
	})
	if err := Configure(opts); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
}
