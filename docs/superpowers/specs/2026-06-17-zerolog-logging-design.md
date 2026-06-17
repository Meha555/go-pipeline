# Zerolog Logging Design

## Goal

Add leveled logging based on `github.com/rs/zerolog` while keeping the CLI pleasant for humans by default. The default output is console format. JSON logging remains available for automation and log collectors.

## Requirements

- Use zerolog for logs that currently use the standard library `log` package.
- Support log format control from both CLI flags and environment variables.
- Support log level control from both CLI flags and environment variables.
- CLI flags take precedence over environment variables.
- Logs continue to write to stderr so command stdout stays available for user-facing output.
- Default behavior is `console` format and `info` level.
- Every log event includes a timestamp.
- `PIPELINE_LOG_TIMESTAMP` is removed and no longer supported.

## User Interface

Add persistent root flags so every subcommand can share the same logging behavior:

```text
--log-format console|json
--log-level debug|info|warn|error|disabled
```

Add environment variables used as defaults when the matching CLI flag is not explicitly provided:

```text
PIPELINE_LOG_FORMAT=console|json
PIPELINE_LOG_LEVEL=debug|info|warn|error|disabled
```

Precedence is:

```text
explicit CLI flag > environment variable > built-in default
```

Invalid values should fail command execution with a clear error before running pipeline logic.

## Architecture

Introduce a small internal logging package, for example `internal/logging`, that owns zerolog configuration.

Responsibilities:

- Parse and validate log format values.
- Parse and validate log level values.
- Build the zerolog output writer.
- Configure `github.com/rs/zerolog/log.Logger` as the process-global logger.
- Set zerolog's global level.

The CLI package remains responsible for reading flags and environment variables. It passes resolved values into `internal/logging` during command initialization, before command-specific logic runs.

## CLI Integration

Register the two logging flags on `rootCmd` with `PersistentFlags()`.

During command execution:

- Resolve `log-format` from explicit flag value, then `PIPELINE_LOG_FORMAT`, then `console`.
- Resolve `log-level` from explicit flag value, then `PIPELINE_LOG_LEVEL`, then `info`.
- Call `logging.Configure(...)` before subcommand `Run` or `RunE` logic.

The implementation should use Cobra flag metadata such as `cmd.Flags().Changed(...)` or `cmd.InheritedFlags().Changed(...)` to distinguish an explicit CLI flag from a default flag value.

## Business Log Mapping

Replace existing standard-library logging with zerolog calls.

Current log points should map as follows:

- `parser/include.go`: loading a config file -> `Debug`.
- `parser/include.go`: include pattern matched files -> `Debug`.
- `parser/include.go`: key override warning -> `Warn`.
- `pipeline/factory.go`: invalid environment variable line -> `Warn`.
- `pipeline/factory.go`: job ignored because its stage is undefined -> `Warn`.
- `pipeline/factory.go`: invalid action format -> `Error`, preserving existing exit behavior.

Use structured fields for dynamic values such as path, include pattern, job name, stage name, config key, and error.

## Output Behavior

Console format uses `zerolog.ConsoleWriter` writing to `os.Stderr`. It should produce readable CLI logs without requiring users to parse JSON.

JSON format writes zerolog JSON events directly to `os.Stderr`.

Configure zerolog with timestamps for both formats so console and JSON logs always carry the same event metadata. Do not keep `PIPELINE_LOG_TIMESTAMP`; timestamp output is no longer configurable. Format, level, and console color are controlled by `PIPELINE_LOG_FORMAT`, `PIPELINE_LOG_LEVEL`, `PIPELINE_LOG_COLOR`, `--log-format`, `--log-level`, and `--log-color`.

Console output keeps zerolog's default short time format. Color mode supports `auto`, `always`, and `never`; `always` requests color but still does not emit color when the output stream is not a terminal that supports color.

## Testing

Update tests that currently capture standard-library `log` output to capture zerolog output instead.

Add or update tests for:

- Default configuration resolves to `console` and `info`.
- Environment variables set format and level.
- CLI flags override environment variables.
- Invalid format fails with a clear error.
- Invalid level fails with a clear error.
- Warning logs remain visible at the default `info` level.
- Debug logs are suppressed at `info` and visible at `debug`.

## Documentation

Update README usage notes to describe:

- `--log-format`.
- `--log-level`.
- `PIPELINE_LOG_FORMAT`.
- `PIPELINE_LOG_LEVEL`.
- CLI flag precedence over environment variables.

Because zerolog is now an intentional runtime dependency, adjust the feature description that currently says the project has no external dependencies.

## Non-Goals

- Add log file output.
- Add per-package log levels.
- Add config-file logging settings.
- Change pipeline execution behavior beyond logging.
- Replace normal command output such as `envs` stdout with logs.
