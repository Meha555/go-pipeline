package parser

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Meha555/go-pipeline/internal/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestParseConfigFileAllowsOptionalShell(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "pipeline.yaml")
	config := []byte(`name: test
version: 1.0.0
stages:
  - build
job:
  stage: build
  actions:
    - echo ok
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if conf.Shell != "" {
		t.Fatalf("Shell = %q, want empty default", conf.Shell)
	}
}

func TestParseConfigFileReadsShell(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "pipeline.yaml")
	config := []byte(`name: test
version: 1.0.0
shell: sh
stages:
  - build
job:
  stage: build
  actions:
    - echo ok
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if conf.Shell != "sh" {
		t.Fatalf("Shell = %q, want sh", conf.Shell)
	}
}

func TestParseConfigFileIncludesLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "base.yaml", `name: test
version: 1.0.0
stages:
  - build
build_job:
  stage: build
  actions:
    - echo from base
  timeout: 1m
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes: base.yaml
build_job:
  actions:
    - echo from main
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if conf.Name != "test" || conf.Version != "1.0.0" {
		t.Fatalf("Name/Version = %q/%q, want test/1.0.0", conf.Name, conf.Version)
	}
	job := conf.Jobs["build_job"]
	if job.Stage != "build" {
		t.Fatalf("build_job.Stage = %q, want build", job.Stage)
	}
	if len(job.Actions) != 1 || job.Actions[0] != "echo from main" {
		t.Fatalf("build_job.Actions = %#v, want main override", job.Actions)
	}
	if job.Timeout != "1m" {
		t.Fatalf("build_job.Timeout = %q, want 1m", job.Timeout)
	}
}

func TestParseConfigFileIncludesListAndReplacesSequences(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "base.yaml", `name: test
version: 1.0.0
stages:
  - build
  - test
envs:
  A: "1"
base_job:
  stage: build
  actions:
    - echo base
`)
	writeTestFile(t, tmpDir, "extra.yaml", `extra_job:
  stage: test
  actions:
    - echo extra
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes:
  - base.yaml
  - extra.yaml
envs:
  B: "2"
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if len(conf.Envs) != 2 {
		t.Fatalf("Envs = %#v, want merged envs with 2 entries", conf.Envs)
	}
	envMap := conf.Envs.ToMap()
	if envMap["A"] != "1" || envMap["B"] != "2" {
		t.Fatalf("Envs = %#v, want A=1 and B=2", conf.Envs)
	}
	if _, ok := conf.Jobs["base_job"]; !ok {
		t.Fatalf("base_job missing after include")
	}
	if _, ok := conf.Jobs["extra_job"]; !ok {
		t.Fatalf("extra_job missing after include")
	}
}

func TestParseConfigFileSupportsMultipleIncludesBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "base.yaml", `name: test
version: 1.0.0
stages:
  - build
  - test
build_job:
  stage: build
  actions:
    - echo build
`)
	writeTestFile(t, tmpDir, "jobs/test.yaml", `test_job:
  stage: test
  actions:
    - echo test
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes:
  - base.yaml
includes:
  - jobs/*.yaml
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if _, ok := conf.Jobs["build_job"]; !ok {
		t.Fatalf("build_job missing after first includes block")
	}
	if _, ok := conf.Jobs["test_job"]; !ok {
		t.Fatalf("test_job missing after second includes block")
	}
}

func TestParseConfigFileReadsMapEnvsAtPipelineAndJob(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestFile(t, tmpDir, "pipeline.yaml", `name: test
version: 1.0.0
envs:
  GLOBAL_VAR: global
  SHARED: from-global
stages:
  - build
build_job:
  stage: build
  envs:
    JOB_VAR: build-only
    SHARED: from-job
  actions:
    - echo ok
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if len(conf.Envs) != 2 || conf.Envs[0].Key != "GLOBAL_VAR" || conf.Envs[0].Value != "global" || conf.Envs[1].Key != "SHARED" || conf.Envs[1].Value != "from-global" {
		t.Fatalf("Pipeline Envs = %#v, want map entries in YAML order", conf.Envs)
	}
	job := conf.Jobs["build_job"]
	if len(job.Envs) != 2 || job.Envs[0].Key != "JOB_VAR" || job.Envs[0].Value != "build-only" || job.Envs[1].Key != "SHARED" || job.Envs[1].Value != "from-job" {
		t.Fatalf("Job Envs = %#v, want map entries in YAML order", job.Envs)
	}
}

func TestParseConfigFileReadsJobExports(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestFile(t, tmpDir, "pipeline.yaml", `name: test
version: 1.0.0
stages:
  - build
build_job:
  stage: build
  actions:
    - echo ok
  exports:
    BUILD_DIR: build/release
    BUILD_VERSION: 1.2.3
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	got := conf.Jobs["build_job"].Exports
	if len(got) != 2 || got[0].Key != "BUILD_DIR" || got[0].Value != "build/release" || got[1].Key != "BUILD_VERSION" || got[1].Value != "1.2.3" {
		t.Fatalf("Exports = %#v, want map entries in YAML order", got)
	}
}

func TestParseConfigFileReadsJobRules(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestFile(t, tmpDir, "pipeline.yaml", `name: test
version: 1.0.0
stages:
  - build
build_job:
  stage: build
  rules:
    - on: $RUN_BUILD
    - on: python scripts/should_build.py
    - {}
  actions:
    - echo ok
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	rules := conf.Jobs["build_job"].Rules
	if len(rules) != 3 {
		t.Fatalf("len(Rules) = %d, want 3", len(rules))
	}
	if rules[0].On.Value != "$RUN_BUILD" || rules[1].On.Value != "python scripts/should_build.py" {
		t.Fatalf("Rules = %#v, want on values", rules)
	}
	if !rules[2].On.Default {
		t.Fatalf("third rule should default to on: true, got %#v", rules[2].On)
	}
}

func TestParseConfigFileRejectsEmptyRules(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestFile(t, tmpDir, "pipeline.yaml", `name: test
version: 1.0.0
stages:
  - build
build_job:
  stage: build
  rules: []
  actions:
    - echo ok
`)

	_, err := ParseConfigFile(configPath)
	if err == nil || !strings.Contains(err.Error(), "Rules") || !strings.Contains(err.Error(), "min") {
		t.Fatalf("ParseConfigFile() error = %v, want empty rules validation error", err)
	}
}

func TestParseConfigFileReadsExampleTestConfig(t *testing.T) {
	if _, err := ParseConfigFile(filepath.Join("..", "configs", "test.yaml")); err != nil {
		t.Fatalf("ParseConfigFile(configs/test.yaml) error = %v", err)
	}
}

func TestParseConfigFileReplacesNonJobTopLevelMappings(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "base.yaml", `name: test
version: 1.0.0
stages:
  - build
notifiers:
  email:
    server: smtp.example.com
    port: 587
    from:
      username: sender@example.com
      password: secret
    to:
      - receiver@example.com
job:
  stage: build
  actions:
    - echo ok
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes: base.yaml
notifiers:
  bot:
    server: https://example.com/hook
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if conf.Notifiers.Email == nil {
		t.Fatalf("Email notifier missing after merge")
	}
	if conf.Notifiers.Bot == nil {
		t.Fatalf("Bot notifier missing after merge")
	}
}

func TestParseConfigFileResolvesNestedIncludesRelativeToDeclaringFile(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "shared/base.yaml", `name: nested
version: 1.0.0
stages:
  - build
build_job:
  stage: build
  actions:
    - echo nested
`)
	writeTestFile(t, tmpDir, "configs/middle.yaml", `includes: ../shared/base.yaml
`)
	configPath := writeTestFile(t, tmpDir, "configs/main.yaml", `includes: middle.yaml
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if conf.Name != "nested" {
		t.Fatalf("Name = %q, want nested", conf.Name)
	}
}

func TestParseConfigFileExpandsWildcardIncludesInSortedOrder(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "jobs/b.yaml", `ordered_job:
  stage: build
  actions:
    - echo b
`)
	writeTestFile(t, tmpDir, "jobs/a.yaml", `ordered_job:
  stage: build
  actions:
    - echo a
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes: jobs/*.yaml
name: wildcard
version: 1.0.0
stages:
  - build
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	got := conf.Jobs["ordered_job"].Actions
	if len(got) != 1 || got[0] != "echo b" {
		t.Fatalf("ordered_job.Actions = %#v, want last sorted include to win", got)
	}
}

func TestParseConfigFileExpandsDoubleStarWildcardIncludes(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "jobs/nested/job.yml", `nested_job:
  stage: build
  actions:
    - echo nested
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes: jobs/**/*.yml
name: doublestar
version: 1.0.0
stages:
  - build
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	if _, ok := conf.Jobs["nested_job"]; !ok {
		t.Fatalf("nested_job missing after ** include")
	}
}

func TestParseConfigFileOrdersWildcardIncludesByFileName(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "jobs/a/z.yml", `ordered_job:
  stage: build
  actions:
    - echo z
`)
	writeTestFile(t, tmpDir, "jobs/z/a.yml", `ordered_job:
  stage: build
  actions:
    - echo a
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes: jobs/**/*.yml
name: filename-order
version: 1.0.0
stages:
  - build
`)

	conf, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	got := conf.Jobs["ordered_job"].Actions
	if len(got) != 1 || got[0] != "echo z" {
		t.Fatalf("ordered_job.Actions = %#v, want filename order with z.yml loaded last", got)
	}
}

func TestParseConfigFileLogsLoadedFilesAndOverrideWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "base.yaml", `name: test
version: 1.0.0
stages:
  - build
build_job:
  stage: build
  actions:
    - echo base
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes: base.yaml
build_job:
  actions:
    - echo main
`)

	var buf bytes.Buffer
	oldLogger := log.Logger
	oldLevel := zerolog.GlobalLevel()
	oldDefault := slog.Default()
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.SetGlobalLevel(oldLevel)
		slog.SetDefault(oldDefault)
	})
	if err := logging.Configure(logging.Options{Format: logging.FormatJSON, Level: logging.LevelDebug, Writer: &buf}); err != nil {
		t.Fatalf("configure logger: %v", err)
	}

	if _, err := ParseConfigFile(configPath); err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}
	logs := buf.String()
	if !strings.Contains(logs, `"level":"debug"`) || !strings.Contains(logs, `"message":"loading config`) {
		t.Fatalf("logs = %q, want debug load config entry", logs)
	}
	if !strings.Contains(logs, `"level":"warn"`) || !strings.Contains(logs, `"key":"build_job.actions"`) || !strings.Contains(logs, `warning: key \"build_job.actions\"`) {
		t.Fatalf("logs = %q, want override warning", logs)
	}
}

func TestParseConfigFileReturnsErrorForEmptyWildcard(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes: missing/*.yaml
name: test
version: 1.0.0
stages:
  - build
job:
  stage: build
  actions:
    - echo ok
`)

	_, err := ParseConfigFile(configPath)
	if err == nil || !strings.Contains(err.Error(), "no files matched") {
		t.Fatalf("ParseConfigFile() error = %v, want empty wildcard error", err)
	}
}

func TestParseConfigFileReturnsErrorForInvalidIncludeItemType(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestFile(t, tmpDir, "main.yaml", `includes:
  - local: base.yaml
name: test
version: 1.0.0
stages:
  - build
job:
  stage: build
  actions:
    - echo ok
`)

	_, err := ParseConfigFile(configPath)
	if err == nil || !strings.Contains(err.Error(), "includes entries must be strings") {
		t.Fatalf("ParseConfigFile() error = %v, want invalid include item error", err)
	}
}

func TestParseConfigFileDoesNotSupportLegacyIncludeKey(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "base.yaml", `name: test
version: 1.0.0
stages:
  - build
job:
  stage: build
  actions:
    - echo ok
`)
	configPath := writeTestFile(t, tmpDir, "main.yaml", `include: base.yaml
`)

	_, err := ParseConfigFile(configPath)
	if err == nil {
		t.Fatalf("ParseConfigFile() error = %v, want error for unsupported legacy include key", err)
	}
}

func TestParseConfigFileReturnsErrorForDuplicateSingletonFields(t *testing.T) {
	tests := []struct {
		name  string
		field string
		base  string
		main  string
	}{
		{name: "name", field: "name", base: "base", main: "main"},
		{name: "version", field: "version", base: "1.0.0", main: "2.0.0"},
		{name: "shell", field: "shell", base: "sh", main: "bash"},
		{name: "cron", field: "cron", base: "@every 1m", main: "@every 2m"},
		{name: "workdir", field: "workdir", base: "/tmp/base", main: "/tmp/main"},
		{name: "stages", field: "stages", base: "[build]", main: "[test]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			baseContent := `name: test
version: 1.0.0
stages:
  - build
job:
  stage: build
  actions:
    - echo ok
`
			if tt.field == "name" {
				baseContent = strings.Replace(baseContent, "name: test\n", "", 1)
			}
			if tt.field == "version" {
				baseContent = strings.Replace(baseContent, "version: 1.0.0\n", "", 1)
			}
			if tt.field == "stages" {
				baseContent = strings.Replace(baseContent, "stages:\n  - build\n", "", 1)
				baseContent += tt.field + `: ` + tt.base + `
`
			} else {
				baseContent += tt.field + `: "` + tt.base + `"
`
			}
			writeTestFile(t, tmpDir, "base.yaml", baseContent)
			mainContent := `includes: base.yaml
`
			if tt.field == "stages" {
				mainContent += tt.field + `: ` + tt.main + `
`
			} else {
				mainContent += tt.field + `: "` + tt.main + `"
`
			}
			configPath := writeTestFile(t, tmpDir, "main.yaml", mainContent)

			_, err := ParseConfigFile(configPath)
			if err == nil || !strings.Contains(err.Error(), "duplicate singleton field \""+tt.field+"\"") {
				t.Fatalf("ParseConfigFile() error = %v, want duplicate singleton field error", err)
			}
		})
	}
}

func TestParseConfigFileReturnsErrorForIncludeCycle(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "a.yaml", `includes: b.yaml
name: test
version: 1.0.0
stages:
  - build
job:
  stage: build
  actions:
    - echo ok
`)
	configPath := writeTestFile(t, tmpDir, "b.yaml", `includes: a.yaml
`)

	_, err := ParseConfigFile(configPath)
	if err == nil || !strings.Contains(err.Error(), "includes cycle") {
		t.Fatalf("ParseConfigFile() error = %v, want includes cycle error", err)
	}
}

func writeTestFile(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create parent dir for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}
