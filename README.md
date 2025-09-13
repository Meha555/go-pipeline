# Go-Pipeline

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

## Quick Start

### 1. Create a Pipeline Configuration

Create a file named `pipeline.yaml`:

```yaml
name: "cmake-pipeline"
version: "1.0.0"

envs:
  - CMAKE_GENERATOR=Ninja

workdir: "D:\\Codes\\C++\\myproject"

stages:
  - build
  - test
  - cleanup

build_job:
  stage: build
  actions:
    - cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
    - cmake --build build -j8

test_job:
  stage: test
  actions:
    - ctest --test-dir build
  timeout: 5m
  allow_failure: yes

cleanup_job:
  stage: cleanup
  actions:
    - rm -rf build
  allow_failure: yes
```

### 2. Run Your Pipeline

```bash
./go-pipeline run -f pipeline.yaml
```
