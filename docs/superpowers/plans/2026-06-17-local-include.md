# Local Include Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add local `include` support to the YAML pipeline parser with GitLab-like precedence, wildcard expansion, and override warnings.

**Architecture:** `parser.ParseConfigFile` will resolve the root YAML file into a merged `yaml.Node` before unmarshalling into the existing `PipelineConf`. Include resolution, wildcard expansion, cycle detection, source tracking, and YAML node merging live in a focused parser helper file.

**Tech Stack:** Go 1.22, `gopkg.in/yaml.v3`, standard library `filepath`, `os`, `log`, existing `go test` test suite.

---

## File Structure

- Modify: `parser/parser_test.go` to add behavior tests that exercise the public parser entry point.
- Create: `parser/include.go` for include resolution and YAML node merge helpers.
- Modify: `parser/parser.go` to route `ParseConfigFile` through include-aware node loading before validation.
- Modify: `parser/keywords.go` to reserve `include`.
- Modify: `README.md` to document local include syntax and merge behavior.

## Task 1: Parser Tests

**Files:**
- Modify: `parser/parser_test.go`

- [ ] **Step 1: Add failing tests for local include behavior**

Add tests that create temporary config files with `os.WriteFile`, call `ParseConfigFile`, and assert merged behavior. Cover string include, list include, nested relative paths, wildcard ordering, job field merge, sequence replacement, missing wildcard, invalid include type, and include cycle.

- [ ] **Step 2: Run tests to verify RED**

Run: `go test ./parser`

Expected: tests fail because `include` is currently treated as an inline job and include files are not loaded.

## Task 2: Include Resolver And YAML Merge

**Files:**
- Create: `parser/include.go`
- Modify: `parser/parser.go`
- Modify: `parser/keywords.go`

- [ ] **Step 1: Implement include-aware YAML loading**

Create helper functions:

```go
func loadConfigNode(configPath string) (*yaml.Node, error)
func loadConfigNodeWithStack(configPath string, stack []string) (*yaml.Node, map[string]string, error)
func resolveIncludePaths(baseFile string, includeNode *yaml.Node) ([]string, error)
func mergeMappingNodes(base, override *yaml.Node, sources map[string]string, overrideFile string, path []string) (*yaml.Node, error)
```

Use `yaml.Node` mapping content pairs. Remove `include` from the current file before merging it into included content.

- [ ] **Step 2: Implement cycle detection**

Track absolute cleaned paths in `stack`. If a path already exists in the stack, return an error containing the include chain.

- [ ] **Step 3: Implement wildcard expansion**

Use `filepath.Glob` for `*`. Implement `**` by walking the base directory and matching normalized slash paths. Sort matches by file name before returning, using the full path as a tie-breaker.

- [ ] **Step 4: Implement override warnings and source tracking**

Track source file per merged key path such as `build_job.actions`. Log `warning: key "build_job.actions" from <override> overrides value from <base>` whenever a later node replaces an existing node.

Singleton fields `name`, `version`, `shell`, `cron`, and `workdir` must not be replaced. Return an error if a later file repeats one of those fields.

- [ ] **Step 5: Wire parser entry point**

Change `ParseConfigFile` to call `loadConfigNode`, unmarshal the merged node into `PipelineConf`, then run existing validation.

- [ ] **Step 6: Reserve include keyword**

Add `include` to `parser/keywords.go` keyword constants and map.

- [ ] **Step 7: Run parser tests to verify GREEN**

Run: `go test ./parser`

Expected: parser package tests pass.

## Task 3: Documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add local include section**

Document supported syntax:

```yaml
include: base.yaml
```

```yaml
include:
  - base.yaml
  - jobs/*.yaml
  - jobs/**/*.yml
```

Document relative path resolution, wildcard sorting, GitLab-like precedence, sequence replacement, and override warnings.

## Task 4: Verification

**Files:**
- All changed Go files

- [ ] **Step 1: Format Go files**

Run: `gofmt -w parser/parser.go parser/parser_test.go parser/keywords.go parser/include.go`

Expected: no output.

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`

Expected: all packages pass.

- [ ] **Step 3: Inspect worktree**

Run: `git diff -- parser README.md docs/superpowers/specs/2026-06-17-local-include-design.md docs/superpowers/plans/2026-06-17-local-include.md`

Expected: diff contains only include feature, tests, docs, and plan/spec files.

## Self-Review

Spec coverage: the plan covers local include syntax, current-file-relative paths, wildcard support, GitLab-like merge precedence, override warnings, cycle/error handling, parser integration, keyword reservation, tests, and README docs.

Placeholder scan: no placeholders remain; each task has concrete files and commands.

Type consistency: helper signatures consistently use `*yaml.Node`, path strings, source maps, and existing `PipelineConf` validation.
