# Go-Pipeline

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/Meha555/go-pipeline?tab=doc)

A lightweight, flexible, and powerful pipeline/workflow execution engine written in Go. This tool allows you to define and execute complex build, test, and deployment workflows using simple YAML configuration files.

## Features

- **Lightweight**: Lightweight design with minimal runtime dependencies and fast startup
- **YAML-based Configuration**: Define your pipelines using intuitive YAML syntax
- **Timeout Support**: Configure timeouts for individual jobs
- **Failure Handling**: Control whether job failures should fail the entire pipeline
- **Working Directory**: Set custom working directories for your pipelines
- **Stage Variable Passing**: Export dotenv files from one stage and inject variables into later stages
- **Extensible**: Easy to extend with custom actions and functionality

## Installation

```bash
go install github.com/Meha555/go-pipeline@latest
```

After installation, the `go-pipeline` binary will be available in your `$GOPATH/bin` or `$HOME/go/bin` directory (ensure this directory is in your PATH).

## Concepts

### Core Components

- **Pipeline**: A complete workflow that consists of multiple Stages.
- **Stage**: A phase within a workflow, representing a complete task that includes multiple Jobs.
- **Job**: A specific task, which is the indivisible minimum execution unit and contains multiple Actions.
- **Action**: A specific operation that constitutes a Job and exists outside the workflow concept.

### Execution Process

1. The workflow starts with a Pipeline. After one Pipeline is executed, the next one will not run automatically and must be specified manually.
2. Within a Pipeline, Stages are executed sequentially. If one Stage fails, subsequent Stages will be terminated, and the entire Pipeline will be marked as failed.
3. Within a Stage, Jobs are executed in parallel. If any Job fails, the Stage will fail—**unless the Job is marked as `allow_failure`**.

### Notes

- Only forward dependencies are allowed for all components (to avoid complex dependency relationships).
- Why is Job failure allowed while Stage failure is not? Because a Job, as the minimum execution unit, already contains many Actions and can complete a range of tasks. A Stage only serves to better isolate the relationships and order between Jobs. Therefore, if a Stage might fail, its failure handling must be properly addressed before being written into the configuration file.

You can obtain serval runtime information through builtin environment variables. Use `go-pipeline envs` to list all builtin envrionment variables.

## Notifiers

Go-Pipeline supports sending notifications when a pipeline succeeds or fails. You can configure different types of notifiers in your pipeline configuration file.

```yaml
notifiers:
  email:
    server: smtp.example.com
    port: 587
    from:
      username: your-email@example.com
      password: your-password
    to:
      - recipient@example.com
    cc:
      - cc-recipient@example.com
  # Work in Process
  # bot:
  #   server: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your-webhook-key"
  # sms:
  #   server: "https://sms.api.qq.com/send"
  #   appid: "your-app-id"
  #   appkey: "your-app-key"
```

## Quick Start

### 1. Create a Pipeline Configuration

Create a file named `pipeline.yaml`:

```yaml
name: "cmake-pipeline"
version: "1.0.0"
# Optional. Defaults to cmd on Windows and sh on Unix-like systems.
# Supported values: cmd, sh, bash, powershell.
# shell: "sh"

cron: "1 * * * *"

envs:
  CMAKE_GENERATOR: Ninja
  MOTTO: "An apple a day $(date +%Y-%m-%d), keeps the `echo 'doctor'` away"

workdir: "D:\\Codes\\C++\\myproject"

stages:
  - build
  - test
  - cleanup

skips:
  - test
  - cleanup_job

build_job:
  stage: build
  envs:
    CMAKE_BUILD_TYPE: Release
  actions:
    - echo "$STAGE_NAME - $JOB_NAME"
    - cmake -S . -B build -DCMAKE_BUILD_TYPE=$CMAKE_BUILD_TYPE
    - cmake --build build -j8
  hooks:
    before:
      - echo "Before build"
      - "echo \"MOTTO: $MOTTO\""
    after:
      - echo "After build"
      - "echo \"build dir: $(pwd)/build\""

test_job:
  stage: test
  actions:
    - echo "$STAGE_NAME - $JOB_NAME"
    - ctest --test-dir build
    - | # 多行命令，可用于让cd和设置变量之类在“多条指令（单个action）”中生效
      echo "large script"
      cwd=`pwd`
      pwd
      echo "line 1, cd to $HOME"
      cd $HOME
      pwd
      echo "line 2, cd to $cwd"
      cd -
      pwd
      echo "line 3"
  timeout: 5m
  allow_failure: yes

cleanup_job:
  stage: cleanup
  actions:
    - echo "$STAGE_NAME - $JOB_NAME"
    - rm -rf build
  allow_failure: yes
```

### Environment Variables

Use `envs` to define environment variables in the same shape as GitLab CI `variables`, with this project's keyword name. Top-level `envs` are available to every job. Job-level `envs` are available only to that job and override top-level variables with the same name.

```yaml
envs:
  MODE: global
  OUT_DIR: build/out

build_job:
  stage: build
  envs:
    MODE: build
    PACKAGE_DIR: "$OUT_DIR/$JOB_NAME"
  actions:
    - echo "$MODE"
    - echo "$PACKAGE_DIR"
```

`JOB_NAME` is a job-level builtin variable injected into each job's actions and hooks. It is not written to the parent process environment, so jobs running in parallel do not overwrite each other's `JOB_NAME`.

### Job Rules

Use job-level `rules` to decide whether a job should run. `rules` must be a non-empty list. Each rule can define `on`; if `on` is omitted, that rule defaults to true. A job runs when any rule matches. If no rules match, the job is skipped successfully.

```yaml
envs:
  RUN_BUILD: "true"

build_job:
  stage: build
  rules:
    - on: $RUN_BUILD
    - on: python scripts/should_build.py
  actions:
    - echo build
```

`on` supports variable references and shell commands:

- `$VAR` or `${VAR}` reads the variable and applies truthy matching. Empty, `0`, `false`, `no`, and `off` are false; other values are true.
- Any other string is executed with the pipeline shell in the pipeline `workdir`. Exit code `0` is true; non-zero is false.
- `true` and `false` YAML booleans are supported directly.

Rules are supported on jobs only. Stages do not have rules; if every job in a stage is skipped, the stage completes successfully.

### Passing Variables Between Stages

Jobs can export local dotenv files with the `exports` keyword. After all jobs in a stage finish successfully, Go-Pipeline reads each exported file and injects its `KEY=VALUE` entries into the pipeline environment. Later stages can use these variables in actions and hooks.

```yaml
stages:
  - build
  - test

build_job:
  stage: build
  actions:
    - echo "BUILD_DIR=build/release" >> build.env
    - echo "BUILD_VERSION=1.2.3" >> build.env
  exports:
    - build.env

test_job:
  stage: test
  actions:
    - echo "Build dir: $BUILD_DIR"
    - echo "Build version: $BUILD_VERSION"
```

`exports` is a list of local temporary file paths. Relative paths are resolved from the pipeline `workdir`; absolute paths are used as-is. Export files are not uploaded, archived, or downloaded. Go-Pipeline removes the configured export files when the pipeline run finishes.

Export files use a minimal dotenv format:

```text
KEY=VALUE
# comments and empty lines are ignored
```

Variables are only guaranteed to be available to later stages. Jobs in the same stage run in parallel, so they should not depend on each other's exports.

### Local Includes

Pipeline files can include other local YAML files before validation and execution. This is useful for sharing common stages, jobs, environment variables, and notifier configuration.

```yaml
includes: base.yaml
```

```yaml
includes:
  - base.yaml
  - jobs/*.yaml
  - jobs/**/*.yml
```

You can declare `includes` more than once in the same YAML file. Blocks are processed in the order they appear:

```yaml
includes:
  - base.yaml
includes:
  - jobs/*.yaml
```

Include paths are resolved relative to the YAML file that declares `includes`. For example, if `configs/main.yaml` includes `base.yaml`, Go-Pipeline loads `configs/base.yaml`. Nested includes are resolved relative to the nested file.

Wildcard includes support `*` and `**`. Matched files are loaded in file-name order for stable merge behavior. If two matches have the same file name, the full path is used as a tie-breaker. A wildcard that matches no files is treated as an error.

Included files are merged first, then the current file is merged on top. This matches GitLab-style precedence: local values override included values. Top-level jobs with the same name are merged by field, so a local job can override `actions` while keeping an included `stage` or `timeout`. Sequence fields such as `stages`, `skips`, `actions`, `hooks.before`, and `hooks.after` are replaced as a whole, not appended. Top-level `envs` are replaced as a whole. Job-level `envs` are merged by key because they are part of the job mapping.

The singleton fields `name`, `version`, `shell`, `cron`, and `workdir` can appear only once across the full include chain. If any included or current file defines one of these fields more than once, parsing fails instead of overriding it.

When a later file overrides an existing key, Go-Pipeline prints a warning to stderr, for example:

```text
warning: key "build_job.actions" from configs/main.yaml overrides value from configs/base.yaml
```

### Shell Selection And Paths With Spaces

Each pipeline runs actions through a shell. If `shell` is omitted, Go-Pipeline selects the platform default shell:

- Windows: `cmd /c`
- Linux/macOS and other non-Windows platforms: `sh -c`

You can override it in the YAML configuration:

```yaml
name: "example"
version: "1.0.0"
shell: "sh" # cmd, sh, bash, or powershell
```

The selected shell is also exposed as the builtin `PIPELINE_SHELL` environment variable.

Action command handling depends on the selected shell:

- `cmd`: Go-Pipeline applies a small safe command-line splitter before invoking `cmd /c`. This supports single-quoted raw strings for paths that contain spaces, because `cmd` does not treat single quotes as quotes.
- `sh` and `bash`: actions are passed as raw shell lines. Use normal POSIX shell syntax.
- `powershell`: actions are passed as raw PowerShell commands. Use PowerShell's call operator (`&`) when executing a quoted path.

Examples for an executable path with spaces:

```yaml
# Windows default cmd: Go-Pipeline strips the single quotes and preserves the path as one argument.
actions:
  - "'C:\\Program Files\\LLVM\\bin\\clang++.exe' --version"

# sh/bash: the shell understands single-quoted paths directly.
shell: "sh"
actions:
  - "'C:\\Program Files\\LLVM\\bin\\clang++.exe' --version"

# powershell: use the call operator for quoted command paths.
shell: "powershell"
actions:
  - "& 'C:\\Program Files\\LLVM\\bin\\clang++.exe' --version"
```

For `cmd`, the safe splitter is intentionally narrow: it is meant to preserve single-quoted command/path segments with spaces. For complex shell syntax, choose a shell that natively supports the syntax you need, such as `sh`, `bash`, or `powershell`.

### Logging

Go-Pipeline writes logs to stderr so stdout remains available for command output.

By default, logs use human-readable console format at `info` level. Console logs are intentionally compact: they show only timestamp, level, and message. Structured fields are hidden in console mode so command output stays easy to read:

```bash
./go-pipeline run -f pipeline.yaml
```

Example console output:

```text
2026-06-17T17:51:02+08:00 INF Stage@build: 1 jobs
2026-06-17T17:51:02+08:00 INF Job@build_job success
2026-06-17T17:51:02+08:00 INF Success (3 succeed/3 total)
```

Use JSON format when you need structured fields for log processing. JSON logs keep all fields, including pipeline context inherited through the logger hierarchy:

```bash
./go-pipeline --log-format json run -f pipeline.yaml
```

Example JSON output:

```json
{"level":"info","pipeline":"include-demo","version":"1.0.0","stage":"build","job":"build_job","actions":2,"time":"2026-06-17T17:51:02+08:00","message":"Job@build_job: 2 actions"}
```

The logger hierarchy is:

- Pipeline logs carry `pipeline` and `version`.
- Stage logs inherit Pipeline fields and add `stage`.
- Job logs inherit Stage fields and add `job`.
- Action logs use the global logger and do not inherit pipeline, stage, or job context.

Internally, Go-Pipeline uses the standard library `log/slog` API and routes logs through zerolog for encoding and output.

Console color defaults to `auto`, which enables color only when stderr is a terminal that supports it.

You can switch format, level, and color behavior with CLI flags:

```bash
./go-pipeline --log-format json --log-level debug --log-color never run -f pipeline.yaml
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

Supported color modes:

- `auto`
- `never`

`auto` enables color only when stderr is a terminal that supports it. `never` disables color.

Environment variables can also set defaults:

```bash
PIPELINE_LOG_FORMAT=json PIPELINE_LOG_LEVEL=warn PIPELINE_LOG_COLOR=never ./go-pipeline run -f pipeline.yaml
```

CLI flags take precedence over environment variables. Every log event includes an RFC3339 timestamp; `PIPELINE_LOG_TIMESTAMP` is not supported.

### 2. Run Your Pipeline

```bash
./go-pipeline run -f pipeline.yaml
```
