package parser

import (
	"os"
	"path/filepath"
	"testing"
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
