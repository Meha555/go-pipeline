# Zerolog Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add zerolog-based leveled logging with console-by-default output, JSON support, and CLI flags that override environment variables.

**Architecture:** Add a focused `internal/logging` package to own zerolog configuration and parsing. Wire it into Cobra root command through persistent flags and a persistent pre-run hook before subcommand logic runs. Replace existing standard-library `log` calls in parser and pipeline code with structured zerolog events.

**Tech Stack:** Go, Cobra, zerolog, standard `testing` package.

---

## File Structure

- Create: `internal/logging/logging.go` for format/level parsing, option resolution, and zerolog configuration.
- Create: `internal/logging/logging_test.go` for unit tests around defaults, env handling, CLI precedence, invalid values, and output filtering.
- Modify: `cli/root.go` to register persistent logging flags and configure logging before each command executes.
- Create: `cli/root_test.go` for CLI-level tests proving flags override environment variables and invalid values fail early.
- Modify: `parser/include.go` to replace standard-library logging with zerolog structured logs.
- Modify: `parser/parser_test.go` to capture zerolog output instead of standard-library log output.
- Modify: `pipeline/factory.go` to replace package-local standard logger with zerolog structured logs and remove `PIPELINE_LOG_TIMESTAMP` logic.
- Modify: `README.md` to document logging flags, environment variables, defaults, and precedence.
- Modify: `go.mod` / `go.sum` with `go mod tidy` so zerolog is a direct dependency.

### Task 1: Add Internal Logging Package

**Files:**
- Create: `internal/logging/logging.go`
- Create: `internal/logging/logging_test.go`

- [ ] **Step 1: Write failing tests for logging configuration**

Create `internal/logging/logging_test.go`:

```go
package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestResolveOptionsUsesDefaults(t *testing.T) {
	t.Setenv("PIPELINE_LOG_FORMAT", "")
	t.Setenv("PIPELINE_LOG_LEVEL", "")

	opts := ResolveOptions("", false, "", false)
	if opts.Format != FormatConsole {
		t.Fatalf("Format = %q, want %q", opts.Format, FormatConsole)
	}
	if opts.Level != LevelInfo {
		t.Fatalf("Level = %q, want %q", opts.Level, LevelInfo)
	}
}

func TestResolveOptionsUsesEnvironment(t *testing.T) {
	t.Setenv("PIPELINE_LOG_FORMAT", "json")
	t.Setenv("PIPELINE_LOG_LEVEL", "debug")

	opts := ResolveOptions("", false, "", false)
	if opts.Format != FormatJSON {
		t.Fatalf("Format = %q, want %q", opts.Format, FormatJSON)
	}
	if opts.Level != LevelDebug {
		t.Fatalf("Level = %q, want %q", opts.Level, LevelDebug)
	}
}

func TestResolveOptionsCliOverridesEnvironment(t *testing.T) {
	t.Setenv("PIPELINE_LOG_FORMAT", "json")
	t.Setenv("PIPELINE_LOG_LEVEL", "error")

	opts := ResolveOptions("console", true, "debug", true)
	if opts.Format != FormatConsole {
		t.Fatalf("Format = %q, want %q", opts.Format, FormatConsole)
	}
	if opts.Level != LevelDebug {
		t.Fatalf("Level = %q, want %q", opts.Level, LevelDebug)
	}
}

func TestConfigureRejectsInvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Configure(Options{Format: "plain", Level: LevelInfo, Writer: &buf})
	if err == nil || !strings.Contains(err.Error(), "invalid log format") {
		t.Fatalf("Configure() error = %v, want invalid log format", err)
	}
}

func TestConfigureRejectsInvalidLevel(t *testing.T) {
	var buf bytes.Buffer
	err := Configure(Options{Format: FormatJSON, Level: "verbose", Writer: &buf})
	if err == nil || !strings.Contains(err.Error(), "invalid log level") {
		t.Fatalf("Configure() error = %v, want invalid log level", err)
	}
}

func TestConfigureJSONIncludesTimestampAndFiltersByLevel(t *testing.T) {
	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
	})

	var buf bytes.Buffer
	if err := Configure(Options{Format: FormatJSON, Level: LevelInfo, Writer: &buf}); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	log.Debug().Msg("hidden")
	log.Warn().Str("key", "value").Msg("visible")

	got := buf.String()
	if strings.Contains(got, "hidden") {
		t.Fatalf("logs = %q, want debug suppressed", got)
	}
	if !strings.Contains(got, `"level":"warn"`) || !strings.Contains(got, `"message":"visible"`) || !strings.Contains(got, `"time":`) {
		t.Fatalf("logs = %q, want warn JSON log with timestamp", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/logging`

Expected: FAIL because `internal/logging` does not exist yet.

- [ ] **Step 3: Implement logging package**

Create `internal/logging/logging.go`:

```go
package logging

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	FormatConsole = "console"
	FormatJSON    = "json"

	LevelDebug    = "debug"
	LevelInfo     = "info"
	LevelWarn     = "warn"
	LevelError    = "error"
	LevelDisabled = "disabled"
)

const (
	EnvLogFormat = "PIPELINE_LOG_FORMAT"
	EnvLogLevel  = "PIPELINE_LOG_LEVEL"
)

type Options struct {
	Format string
	Level  string
	Writer io.Writer
}

func ResolveOptions(flagFormat string, hasFlagFormat bool, flagLevel string, hasFlagLevel bool) Options {
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

	return Options{Format: format, Level: level}
}

func Configure(opts Options) error {
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	levelName := strings.ToLower(strings.TrimSpace(opts.Level))
	level, err := parseLevel(levelName)
	if err != nil {
		return err
	}

	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}

	switch format {
	case FormatConsole:
		writer = zerolog.ConsoleWriter{Out: writer}
	case FormatJSON:
	default:
		return fmt.Errorf("invalid log format %q: supported values are console, json", opts.Format)
	}

	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(writer).With().Timestamp().Logger()
	return nil
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
```

- [ ] **Step 4: Run logging package tests**

Run: `go test ./internal/logging`

Expected: PASS.

### Task 2: Wire Logging Flags Into Cobra Root Command

**Files:**
- Modify: `cli/root.go`
- Create: `cli/root_test.go`

- [ ] **Step 1: Write failing CLI tests**

Create `cli/root_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestConfigureLoggingRejectsInvalidFormat(t *testing.T) {
	err := configureLogging("plain", true, "info", true)
	if err == nil || !strings.Contains(err.Error(), "invalid log format") {
		t.Fatalf("configureLogging() error = %v, want invalid log format", err)
	}
}

func TestConfigureLoggingRejectsInvalidLevel(t *testing.T) {
	err := configureLogging("json", true, "verbose", true)
	if err == nil || !strings.Contains(err.Error(), "invalid log level") {
		t.Fatalf("configureLogging() error = %v, want invalid log level", err)
	}
}

func TestConfigureLoggingCliOverridesEnvironment(t *testing.T) {
	t.Setenv("PIPELINE_LOG_FORMAT", "json")
	t.Setenv("PIPELINE_LOG_LEVEL", "error")

	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
	})

	var buf bytes.Buffer
	oldWriter := loggingWriter
	loggingWriter = &buf
	t.Cleanup(func() { loggingWriter = oldWriter })

	if err := configureLogging("json", true, "debug", true); err != nil {
		t.Fatalf("configureLogging() error = %v", err)
	}
	log.Debug().Msg("visible")

	got := buf.String()
	if !strings.Contains(got, `"level":"debug"`) || !strings.Contains(got, `"message":"visible"`) {
		t.Fatalf("logs = %q, want debug JSON log", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cli`

Expected: FAIL because `configureLogging` and `loggingWriter` do not exist.

- [ ] **Step 3: Implement root logging flags and initialization**

Modify `cli/root.go`:

```go
package cli

import (
	"io"
	"os"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/Meha555/go-pipeline/internal/logging"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "go-pipeline",
	Short:   "A tool to run workflow",
	Version: internal.ResolveVersion(internal.BuildVersion()),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(logFormat, cmd.Flags().Changed("log-format"), logLevel, cmd.Flags().Changed("log-level"))
	},
}

var (
	logFormat     string
	logLevel      string
	loggingWriter io.Writer = os.Stderr
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func configureLogging(format string, hasFormat bool, level string, hasLevel bool) error {
	opts := logging.ResolveOptions(format, hasFormat, level, hasLevel)
	opts.Writer = loggingWriter
	return logging.Configure(opts)
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceUsage = true
	rootCmd.SetVersionTemplate("{{.Version}}")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", logging.FormatConsole, "log format: console or json")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", logging.LevelInfo, "log level: debug, info, warn, error, or disabled")
}
```

- [ ] **Step 4: Run CLI tests**

Run: `go test ./cli`

Expected: PASS.

### Task 3: Replace Parser Standard Logs With Zerolog

**Files:**
- Modify: `parser/include.go`
- Modify: `parser/parser_test.go`

- [ ] **Step 1: Update parser test to capture zerolog JSON output**

Modify imports in `parser/parser_test.go`: remove `log`, add `github.com/Meha555/go-pipeline/internal/logging`, `github.com/rs/zerolog`, and `github.com/rs/zerolog/log` if not already present.

Replace `TestParseConfigFileLogsLoadedFilesAndOverrideWarnings` log capture block with:

```go
	var buf bytes.Buffer
	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
	})
	if err := logging.Configure(logging.Options{Format: logging.FormatJSON, Level: logging.LevelDebug, Writer: &buf}); err != nil {
		t.Fatalf("configure logger: %v", err)
	}

	if _, err := ParseConfigFile(configPath); err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	logs := buf.String()
	if !strings.Contains(logs, `"level":"debug"`) || !strings.Contains(logs, `"message":"load config"`) {
		t.Fatalf("logs = %q, want debug load config entry", logs)
	}
	if !strings.Contains(logs, `"level":"warn"`) || !strings.Contains(logs, `"key":"build_job.actions"`) || !strings.Contains(logs, `"message":"config key overridden"`) {
		t.Fatalf("logs = %q, want override warning", logs)
	}
```

- [ ] **Step 2: Run parser tests to verify failure**

Run: `go test ./parser`

Expected: FAIL because parser still logs through standard-library `log`.

- [ ] **Step 3: Replace parser log calls**

Modify `parser/include.go` imports: remove `log`, add `github.com/rs/zerolog/log`.

Replace log calls:

```go
log.Debug().Str("path", absPath).Msg("load config")
```

```go
log.Debug().Str("pattern", includePath).Strs("matches", matches).Msg("include pattern matched")
```

```go
log.Warn().
	Str("key", pathKey).
	Str("override_file", overrideFile).
	Str("base_source", baseSource).
	Msg("config key overridden")
```

- [ ] **Step 4: Run parser tests**

Run: `go test ./parser`

Expected: PASS.

### Task 4: Replace Pipeline Factory Logger

**Files:**
- Modify: `pipeline/factory.go`
- Modify: `pipeline/factory_test.go`

- [ ] **Step 1: Write failing factory logging test**

Modify `pipeline/factory_test.go` imports:

```go
import (
	"bytes"
	"strings"
	"testing"

	"github.com/Meha555/go-pipeline/internal/logging"
	"github.com/Meha555/go-pipeline/parser"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)
```

Add test:

```go
func TestMakePipelineLogsWarningsWithZerolog(t *testing.T) {
	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
	})

	var buf bytes.Buffer
	if err := logging.Configure(logging.Options{Format: logging.FormatJSON, Level: logging.LevelInfo, Writer: &buf}); err != nil {
		t.Fatalf("configure logger: %v", err)
	}

	MakePipeline(&parser.PipelineConf{
		Name:    "test",
		Version: "1.0.0",
		Envs:    []string{"INVALID_ENV"},
		Stages:  []string{"build"},
		Jobs: map[string]parser.Job{
			"orphan_job": {
				Stage:   "missing",
				Actions: []string{"echo ok"},
			},
		},
	})

	logs := buf.String()
	if !strings.Contains(logs, `"level":"warn"`) || !strings.Contains(logs, `"env":"INVALID_ENV"`) || !strings.Contains(logs, `"message":"invalid env format"`) {
		t.Fatalf("logs = %q, want invalid env warning", logs)
	}
	if !strings.Contains(logs, `"job":"orphan_job"`) || !strings.Contains(logs, `"stage":"missing"`) || !strings.Contains(logs, `"message":"job ignored because stage is undefined"`) {
		t.Fatalf("logs = %q, want undefined stage warning", logs)
	}
}
```

- [ ] **Step 2: Run pipeline tests to verify failure**

Run: `go test ./pipeline`

Expected: FAIL because `pipeline/factory.go` still uses its package-local standard logger.

- [ ] **Step 3: Replace factory logger with zerolog**

Modify `pipeline/factory.go` imports: remove `log`, keep `os` for existing `os.Exit`, add `github.com/rs/zerolog/log`.

Replace:

```go
logger.Printf("invalid env format: %s (expected key=value)", envLine)
```

with:

```go
log.Warn().Str("env", envLine).Msg("invalid env format")
```

Replace:

```go
logger.Printf("job %s belong to undefined stage %s, ignored it", jobName, jobDef.Stage)
```

with:

```go
log.Warn().Str("job", jobName).Str("stage", jobDef.Stage).Msg("job ignored because stage is undefined")
```

Replace:

```go
logger.Printf("invalid action format: %s (error: %v)", actionLine, err)
```

with:

```go
log.Error().Err(err).Str("action", actionLine).Msg("invalid action format")
```

Delete the package-level `logger *log.Logger` variable and its `init()` function. This removes `PIPELINE_LOG_TIMESTAMP` support.

- [ ] **Step 4: Run pipeline tests**

Run: `go test ./pipeline`

Expected: PASS.

### Task 5: Update README And Dependency Metadata

**Files:**
- Modify: `README.md`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Update README logging documentation**

Modify the feature bullet:

```markdown
- **Lightweight**: Lightweight design with minimal runtime dependencies and fast startup
```

Add a section before `### 2. Run Your Pipeline`:

```markdown
### Logging

Go-Pipeline writes logs to stderr so stdout remains available for command output.

By default, logs use human-readable console format at `info` level:

```bash
./go-pipeline run -f pipeline.yaml
```

You can switch format and level with CLI flags:

```bash
./go-pipeline --log-format json --log-level debug run -f pipeline.yaml
```

Supported formats:

- `console`
- `json`

Supported levels:

- `debug`
- `info`
- `warn`
- `error`
- `disabled`

Environment variables can also set defaults:

```bash
PIPELINE_LOG_FORMAT=json PIPELINE_LOG_LEVEL=warn ./go-pipeline run -f pipeline.yaml
```

CLI flags take precedence over environment variables. Every log event includes a timestamp; `PIPELINE_LOG_TIMESTAMP` is not supported.
```

- [ ] **Step 2: Tidy module metadata**

Run: `go mod tidy`

Expected: `github.com/rs/zerolog` is a direct dependency in `go.mod`.

### Task 6: Full Verification

**Files:**
- All changed files

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 2: Inspect working tree diff**

Run: `git diff -- docs/superpowers/specs/2026-06-17-zerolog-logging-design.md docs/superpowers/plans/2026-06-17-zerolog-logging.md internal/logging/logging.go internal/logging/logging_test.go cli/root.go cli/root_test.go parser/include.go parser/parser_test.go pipeline/factory.go pipeline/factory_test.go README.md go.mod go.sum`

Expected: Diff contains only zerolog logging implementation, documentation, and tests.

## Self-Review

- Spec coverage: tasks cover CLI/env format control, CLI/env level control, CLI precedence, console default, JSON support, timestamp on every event, removal of `PIPELINE_LOG_TIMESTAMP`, stderr output, existing log replacement, tests, and README updates.
- Placeholder scan: no placeholder tasks remain; each code task includes exact file paths, code, commands, and expected results.
- Type consistency: `logging.Options`, `ResolveOptions`, `Configure`, `FormatConsole`, `FormatJSON`, level constants, and environment constants are consistently named across tasks.
