package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPipelineRemovesExportFilesAfterRun(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	exportPath := filepath.Join(tmpDir, "build.env")

	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir))
	build := NewStage("build", p)
	build.AddJob(NewJob("build_job", []*Action{
		NewAction(p.Shell, "echo BUILD_DIR=build/release > build.env"),
	}, build, WithExports([]string{"build.env"})))
	testStage := NewStage("test", p)
	testStage.AddJob(NewJob("test_job", []*Action{
		NewAction(p.Shell, "test \"$BUILD_DIR\" = \"build/release\""),
	}, testStage))
	p.AddStage(build).AddStage(testStage)

	if status := p.Run(context.Background()); status != Success {
		t.Fatalf("Pipeline status = %s, want Success", status)
	}
	if _, err := os.Stat(exportPath); !os.IsNotExist(err) {
		t.Fatalf("export file still exists after pipeline run, stat error = %v", err)
	}
}

func TestJobDoesNotImportExportsBeforeStageCompletes(t *testing.T) {
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "build.env")
	content := []byte("BUILD_DIR=build/release\n")
	if err := os.WriteFile(exportPath, content, 0o600); err != nil {
		t.Fatalf("write export file: %v", err)
	}
	t.Setenv("BUILD_DIR", "")

	p := NewPipeline("test", "1.0.0", WithWorkdir(tmpDir))
	s := NewStage("build", p)
	j := NewJob("build_job", nil, s, WithExports([]string{"build.env"}))

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
	exportPath := filepath.Join(tmpDir, "build.env")
	content := []byte("BUILD_DIR=build/release\n# ignored\nBUILD_VERSION=1.2.3\nEMPTY=\n非法=skip\n")
	if err := os.WriteFile(exportPath, content, 0o600); err != nil {
		t.Fatalf("write export file: %v", err)
	}
	for _, key := range []string{"BUILD_DIR", "BUILD_VERSION", "EMPTY", "非法"} {
		t.Setenv(key, "")
	}

	p := NewPipeline("test", "1.0.0", WithWorkdir(tmpDir))
	s := NewStage("build", p)
	s.AddJob(NewJob("build_job", nil, s, WithExports([]string{"build.env"})))

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
	if got := os.Getenv("非法"); got != "" {
		t.Fatalf("非法 = %q, want empty because non-ASCII keys are invalid", got)
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

func TestJobDoesNotImportExportsAfterFailure(t *testing.T) {
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "build.env")
	if err := os.WriteFile(exportPath, []byte("SHOULD_NOT_IMPORT=yes\n"), 0o600); err != nil {
		t.Fatalf("write export file: %v", err)
	}
	t.Setenv("SHOULD_NOT_IMPORT", "")

	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir))
	s := NewStage("build", p)
	j := NewJob("build_job", []*Action{NewAction(p.Shell, "exit 1")}, s, WithExports([]string{"build.env"}))

	s.wg.Add(1)
	go j.Do(context.Background())
	if status := <-j.Result(); status != Failed {
		t.Fatalf("Job status = %s, want Failed", status)
	}
	s.wg.Wait()

	if got := os.Getenv("SHOULD_NOT_IMPORT"); got != "" {
		t.Fatalf("SHOULD_NOT_IMPORT = %q, want empty", got)
	}
}
