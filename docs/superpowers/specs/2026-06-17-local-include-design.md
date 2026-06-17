# Local Include Design

## Goal

Add GitLab-style local `include` support to the YAML pipeline parser. The feature lets a pipeline configuration reuse jobs, stages, environment variables, and other supported top-level configuration from other local YAML files before validation and execution.

## Scope

The first version supports only local file includes. It does not support GitLab `local`, `remote`, `project`, or `template` object syntax.

Supported syntax:

```yaml
include: base.yaml
```

```yaml
include:
  - base.yaml
  - jobs/*.yaml
  - jobs/**/*.yml
```

Unsupported syntax returns a parser error:

```yaml
include:
  - local: base.yaml
```

## Path Resolution

Include paths are resolved relative to the configuration file that declares them.

Example:

```text
configs/main.yaml
configs/base.yaml
configs/jobs/build.yaml
```

In `configs/main.yaml`:

```yaml
include:
  - base.yaml
  - jobs/*.yaml
```

The parser resolves those paths as `configs/base.yaml` and `configs/jobs/*.yaml`.

Nested includes use the nested file's directory as their base directory.

## Wildcards

The parser supports `*` and `**` wildcard patterns in include entries.

Wildcard matches are sorted by file name before loading so merge order is stable across platforms. If two matches have the same file name, the full path is used as a tie-breaker. A wildcard include with no matches is an error, because silently ignoring a missing include can hide pipeline configuration mistakes.

## Merge Semantics

The parser follows GitLab's include precedence model: included files load first, then the current file is merged on top. Later configuration overrides earlier configuration.

Merge order:

1. Includes are processed in declaration order.
2. Wildcard matches are expanded in file-name order at the position where the wildcard appears.
3. Nested includes are fully resolved before their including file is merged.
4. The current file is merged last and therefore has final precedence.

The merge is performed on YAML nodes before unmarshalling into `PipelineConf`. This preserves information about whether fields are present and avoids confusing omitted fields with Go zero values.

Rules:

- Top-level scalar, sequence, and mapping fields are overridden by the later file.
- Singleton fields `name`, `version`, `shell`, `cron`, and `workdir` are exceptions: they can appear only once across the full include chain. Repeating any of them is an error.
- Top-level job mappings with the same name are merged recursively.
- Fields inside the same job are overridden by the later file when present.
- Sequence fields such as `stages`, `envs`, `skips`, `actions`, `hooks.before`, and `hooks.after` are replaced as a whole, not appended.
- The `include` key is used only for loading and is removed before final unmarshalling and validation.

Example:

```yaml
# base.yaml
stages:
  - build
  - test

envs:
  - A=1

build_job:
  stage: build
  actions:
    - echo from base
  timeout: 1m
```

```yaml
# main.yaml
include: base.yaml

envs:
  - B=2

build_job:
  actions:
    - echo from main
```

Effective result:

```yaml
stages:
  - build
  - test

envs:
  - B=2

build_job:
  stage: build
  actions:
    - echo from main
  timeout: 1m
```

## Logging

The parser logs include activity to stderr with the standard library logger, matching the existing lightweight logging style.

Logs include:

- Each configuration file loaded.
- Each wildcard pattern and the files it expands to.
- A warning whenever a later file overrides an existing top-level key or job field.

Example warning:

```text
warning: key "build_job.actions" from configs/main.yaml overrides value from configs/base.yaml
```

## Error Handling

The parser reports errors for:

- Missing explicitly included files.
- Wildcard include patterns with no matches.
- Include entries that are not strings.
- YAML parse failures in included files.
- Include cycles, such as `a.yaml -> b.yaml -> a.yaml`.
- Final configuration validation failures from the existing validator.

Cycle errors include the include chain so the user can find the loop quickly.

## Integration Points

`parser.ParseConfigFile` remains the public entry point. Internally it changes from reading and unmarshalling one file to resolving includes into one merged YAML node before unmarshalling into `PipelineConf`.

`parser.IsKeyword` should treat `include` as a reserved keyword so it is not considered a job name if any intermediate representation still contains it.

The pipeline execution layer does not need to change because it already consumes the final `PipelineConf`.

## Tests

Parser tests should cover:

- Single string include.
- Include list.
- Nested includes with paths relative to each declaring file.
- Wildcard include expansion and stable ordering.
- Current file overriding included top-level fields.
- Current job fields overriding included job fields while preserving unspecified fields.
- Sequence replacement instead of append.
- Override warning logs.
- Missing file error.
- Empty wildcard error.
- Invalid include item type error.
- Include cycle error.

## Non-Goals

This feature does not implement network includes, GitLab project includes, built-in GitLab templates, authentication, repository-root path resolution, or GitLab's full CI/CD schema.
