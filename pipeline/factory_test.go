package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Meha555/go-pipeline/parser"
)

func TestMakeActionsUsesSafeCmdlineForCmd(t *testing.T) {
	shell := [2]string{"cmd", "/c"}
	actions := makeActions(shell, []string{"'C:\\Program Files\\LLVM\\bin\\clang++.exe' --version"})
	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1", len(actions))
	}
	if actions[0].Shell != shell {
		t.Fatalf("Shell = %#v, want %#v", actions[0].Shell, shell)
	}
	if actions[0].Cmd != "C:\\Program Files\\LLVM\\bin\\clang++.exe" {
		t.Fatalf("Cmd = %q", actions[0].Cmd)
	}
	if len(actions[0].Args) != 1 || actions[0].Args[0] != "--version" {
		t.Fatalf("Args = %#v, want [--version]", actions[0].Args)
	}
}

func TestMakeActionsKeepsRawLineForShells(t *testing.T) {
	for _, shell := range [][2]string{{"sh", "-c"}, {"bash", "-c"}, {"powershell", "-c"}} {
		t.Run(shell[0], func(t *testing.T) {
			actions := makeActions(shell, []string{"'C:\\Program Files\\LLVM\\bin\\clang++.exe' --version"})
			if len(actions) != 1 {
				t.Fatalf("len(actions) = %d, want 1", len(actions))
			}
			if actions[0].Shell != shell {
				t.Fatalf("Shell = %#v, want %#v", actions[0].Shell, shell)
			}
			if actions[0].Cmd != "'C:\\Program Files\\LLVM\\bin\\clang++.exe' --version" {
				t.Fatalf("Cmd = %q", actions[0].Cmd)
			}
			if len(actions[0].Args) != 0 {
				t.Fatalf("Args = %#v, want empty", actions[0].Args)
			}
		})
	}
}

func TestMakePipelinePassesExportsToJob(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "pipeline.yaml")
	config := []byte(`name: test
version: 1.0.0
stages:
  - build
build_job:
  stage: build
  actions:
    - echo ok
  exports:
    - build.env
    - version.env
`)
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	conf, err := parser.ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}

	pipe := MakePipeline(conf)
	if len(pipe.Stages) != 1 || len(pipe.Stages[0].Jobs) != 1 {
		t.Fatalf("pipeline shape = %d stages, want one stage with one job", len(pipe.Stages))
	}
	got := pipe.Stages[0].Jobs[0].Exports
	want := []string{"build.env", "version.env"}
	if len(got) != len(want) {
		t.Fatalf("Exports = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Exports = %#v, want %#v", got, want)
		}
	}
}

func TestMakePipelineRejectsDuplicateStageName(t *testing.T) {
	if os.Getenv("TEST_DUPLICATE_STAGE") == "1" {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "pipeline.yaml")
		config := []byte(`name: test
version: 1.0.0
stages:
  - build
  - build
`)
		os.WriteFile(configPath, config, 0o600)
		conf, _ := parser.ParseConfigFile(configPath)
		MakePipeline(conf)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMakePipelineRejectsDuplicateStageName")
	cmd.Env = append(os.Environ(), "TEST_DUPLICATE_STAGE=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected process to exit with non-zero status")
	}
}

func TestMakePipelinePassesEnvsToJob(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "pipeline.yaml")
	config := []byte(`name: test
version: 1.0.0
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
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	conf, err := parser.ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}

	pipe := MakePipeline(conf)
	if len(pipe.Stages) != 1 || len(pipe.Stages[0].Jobs) != 1 {
		t.Fatalf("pipeline shape = %d stages, want one stage with one job", len(pipe.Stages))
	}
	got := pipe.Stages[0].Jobs[0].Envs
	if len(got) != 2 || got[0].Key != "JOB_VAR" || got[0].Value != "build-only" || got[1].Key != "SHARED" || got[1].Value != "from-job" {
		t.Fatalf("Envs = %#v, want job envs", got)
	}
}
