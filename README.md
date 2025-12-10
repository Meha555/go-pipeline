# Go-Pipeline

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/Meha555/go-pipeline?tab=doc)

A lightweight, flexible, and powerful pipeline/workflow execution engine written in Go. This tool allows you to define and execute complex build, test, and deployment workflows using simple YAML configuration files.

## Features

- **Lightweight**: Lightweight design, no external dependencies, quickly startup
- **YAML-based Configuration**: Define your pipelines using intuitive YAML syntax
- **Timeout Support**: Configure timeouts for individual jobs
- **Failure Handling**: Control whether job failures should fail the entire pipeline
- **Working Directory**: Set custom working directories for your pipelines
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
shell: sh

cron: "1 * * * *"

envs:
  - CMAKE_GENERATOR=Ninja
  - MOTTO="An apple a day $(date +%Y-%m-%d), keeps the `echo 'doctor'` away"

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
  actions:
    - echo "$STAGE_NAME - $JOB_NAME"
    - cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
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

### 2. Run Your Pipeline

```bash
./go-pipeline run -f pipeline.yaml
```
