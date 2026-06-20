package pipeline

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestJobDoesNotImportExportsBeforeStageCompletes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("BUILD_DIR", "")

	p := NewPipeline("test", "1.0.0", WithWorkdir(tmpDir))
	s := NewStage("build", p)
	j := NewJob("build_job", nil, s, WithExports(EnvList{{Key: "BUILD_DIR", Value: "build/release"}}))

	s.wg.Add(1)
	go j.Do(context.Background())
	if status := <-j.Result(); status != Success {
		t.Fatalf("Job status = %s, want Success", status)
	}
	s.wg.Wait()

	if got := os.Getenv("BUILD_DIR"); got != "" {
		t.Fatalf("BUILD_DIR = %q, want empty before stage import", got)
	}
}

func TestStageImportsExportsAfterSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	for _, key := range []string{"BUILD_DIR", "BUILD_VERSION", "EMPTY"} {
		t.Setenv(key, "")
	}

	p := NewPipeline("test", "1.0.0", WithWorkdir(tmpDir))
	s := NewStage("build", p)
	s.AddJob(NewJob("build_job", nil, s, WithExports(EnvList{
		{Key: "BUILD_DIR", Value: "build/release"},
		{Key: "BUILD_VERSION", Value: "1.2.3"},
		{Key: "EMPTY", Value: ""},
	})))

	if status := s.Perform(context.Background()); status != Success {
		t.Fatalf("Stage status = %s, want Success", status)
	}

	if got := os.Getenv("BUILD_DIR"); got != "build/release" {
		t.Fatalf("BUILD_DIR = %q, want build/release", got)
	}
	if got := os.Getenv("BUILD_VERSION"); got != "1.2.3" {
		t.Fatalf("BUILD_VERSION = %q, want 1.2.3", got)
	}
	if got := os.Getenv("EMPTY"); got != "" {
		t.Fatalf("EMPTY = %q, want empty", got)
	}
}

func restoreWdAfterTest(t *testing.T) {
	t.Helper()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func TestStageDoesNotImportExportsAfterFailure(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}

	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	t.Setenv("SHOULD_NOT_IMPORT", "")

	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir))
	s := NewStage("build", p)
	s.AddJob(NewJob("build_job", []*Action{NewAction(p.Shell, "exit 1")}, s, WithExports(EnvList{{Key: "SHOULD_NOT_IMPORT", Value: "yes"}})))

	if status := s.Perform(context.Background()); status != Failed {
		t.Fatalf("Stage status = %s, want Failed", status)
	}

	if got := os.Getenv("SHOULD_NOT_IMPORT"); got != "" {
		t.Fatalf("SHOULD_NOT_IMPORT = %q, want empty", got)
	}
}

func TestStageResolvesExportValuesAfterSuccess(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}

	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	t.Setenv("BASE_VERSION", "1.2.3")
	t.Setenv("RELEASE", "")
	t.Setenv("COMMAND_VALUE", "")

	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir))
	s := NewStage("build", p)
	s.AddJob(NewJob("build_job", nil, s, WithExports(EnvList{
		{Key: "RELEASE", Value: "$BASE_VERSION/release"},
		{Key: "COMMAND_VALUE", Value: "$(printf export-ok)"},
	})))

	if status := s.Perform(context.Background()); status != Success {
		t.Fatalf("Stage status = %s, want Success", status)
	}

	if got := os.Getenv("RELEASE"); got != "1.2.3/release" {
		t.Fatalf("RELEASE = %q, want 1.2.3/release", got)
	}
	if got := os.Getenv("COMMAND_VALUE"); got != "export-ok" {
		t.Fatalf("COMMAND_VALUE = %q, want export-ok", got)
	}
}
