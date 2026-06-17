package cli

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestConfigureLoggingRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name   string
		format string
		level  string
		color  string
		want   string
	}{
		{name: "format", format: "plain", level: "info", color: "auto", want: "invalid log format"},
		{name: "level", format: "json", level: "verbose", color: "auto", want: "invalid log level"},
		{name: "color", format: "console", level: "info", color: "sometimes", want: "invalid log color"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := configureLogging(tt.format, true, tt.level, true, tt.color, true)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("configureLogging() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestConfigureLoggingCliOverridesEnvironment(t *testing.T) {
	t.Setenv("PIPELINE_LOG_FORMAT", "json")
	t.Setenv("PIPELINE_LOG_LEVEL", "error")
	t.Setenv("PIPELINE_LOG_COLOR", "never")

	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	oldDefault := slog.Default()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
		slog.SetDefault(oldDefault)
	})

	var buf bytes.Buffer
	oldWriter := loggingWriter
	loggingWriter = &buf
	t.Cleanup(func() { loggingWriter = oldWriter })

	if err := configureLogging("json", true, "debug", true, "auto", true); err != nil {
		t.Fatalf("configureLogging() error = %v", err)
	}
	slog.Debug("visible")

	got := buf.String()
	if !strings.Contains(got, `"level":"debug"`) || !strings.Contains(got, `"message":"visible"`) {
		t.Fatalf("logs = %q, want debug JSON log", got)
	}
}
