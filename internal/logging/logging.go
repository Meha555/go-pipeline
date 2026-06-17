package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

const (
	FormatConsole = "console"
	FormatJSON    = "json"

	LevelDebug    = "debug"
	LevelInfo     = "info"
	LevelWarn     = "warn"
	LevelError    = "error"
	LevelDisabled = "disabled"

	ColorAuto  = "auto"
	ColorNever = "never"
)

const (
	EnvLogFormat = "PIPELINE_LOG_FORMAT"
	EnvLogLevel  = "PIPELINE_LOG_LEVEL"
	EnvLogColor  = "PIPELINE_LOG_COLOR"
)

type Options struct {
	Format string
	Level  string
	Color  string
	Writer io.Writer
}

func ResolveOptions(flagFormat string, hasFlagFormat bool, flagLevel string, hasFlagLevel bool, flagColor string, hasFlagColor bool) Options {
	format := FormatConsole
	if envFormat := strings.TrimSpace(os.Getenv(EnvLogFormat)); envFormat != "" {
		format = envFormat
	}
	if hasFlagFormat {
		format = flagFormat
	}

	level := LevelInfo
	if envLevel := strings.TrimSpace(os.Getenv(EnvLogLevel)); envLevel != "" {
		level = envLevel
	}
	if hasFlagLevel {
		level = flagLevel
	}

	color := ColorAuto
	if envColor := strings.TrimSpace(os.Getenv(EnvLogColor)); envColor != "" {
		color = envColor
	}
	if hasFlagColor {
		color = flagColor
	}

	return Options{Format: format, Level: level, Color: color}
}

func Configure(opts Options) error {
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	levelName := strings.ToLower(strings.TrimSpace(opts.Level))
	colorName := strings.ToLower(strings.TrimSpace(opts.Color))
	if colorName == "" {
		colorName = ColorAuto
	}
	level, err := parseLevel(levelName)
	if err != nil {
		return err
	}
	if err := validateColor(colorName); err != nil {
		return err
	}

	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}

	switch format {
	case FormatConsole:
		writer = zerolog.ConsoleWriter{
			Out:           writer,
			NoColor:       !shouldUseColor(colorName, writer),
			TimeFormat:    time.RFC3339,
			PartsOrder:    []string{zerolog.TimestampFieldName, zerolog.LevelFieldName, zerolog.MessageFieldName},
			FormatPrepare: stripConsoleFields,
		}
	case FormatJSON:
	default:
		return fmt.Errorf("invalid log format %q: supported values are console, json", opts.Format)
	}

	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(writer).With().Timestamp().Logger()
	slog.SetDefault(slog.New(zerolog.NewSlogHandler(log.Logger)))
	return nil
}

func stripConsoleFields(evt map[string]interface{}) error {
	for field := range evt {
		switch field {
		case zerolog.TimestampFieldName, zerolog.LevelFieldName, zerolog.MessageFieldName:
			continue
		default:
			delete(evt, field)
		}
	}
	return nil
}

func validateColor(color string) error {
	switch color {
	case ColorAuto, ColorNever:
		return nil
	default:
		return fmt.Errorf("invalid log color %q: supported values are auto, never", color)
	}
}

func shouldUseColor(color string, writer io.Writer) bool {
	if color == ColorNever {
		return false
	}
	return writerSupportsColor(writer)
}

func writerSupportsColor(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func parseLevel(level string) (zerolog.Level, error) {
	switch level {
	case LevelDebug:
		return zerolog.DebugLevel, nil
	case LevelInfo:
		return zerolog.InfoLevel, nil
	case LevelWarn:
		return zerolog.WarnLevel, nil
	case LevelError:
		return zerolog.ErrorLevel, nil
	case LevelDisabled:
		return zerolog.Disabled, nil
	default:
		return zerolog.NoLevel, fmt.Errorf("invalid log level %q: supported values are debug, info, warn, error, disabled", level)
	}
}
