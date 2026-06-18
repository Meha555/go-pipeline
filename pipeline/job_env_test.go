package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJobEnvsOverridePipelineEnvsOnlyForThatJob(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	t.Setenv("JOB_VAR", "")
	t.Setenv("JOB_NAME", "")
	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir), WithEnvs(EnvList{
		{Key: "GLOBAL_VAR", Value: "global"},
		{Key: "SHARED", Value: "from-global"},
	}))
	build := NewStage("build", p)
	build.AddJob(NewJob("build_job", []*Action{
		NewAction(p.Shell, "printf '%s,%s,%s,%s' \"$GLOBAL_VAR\" \"$JOB_VAR\" \"$SHARED\" \"$JOB_NAME\" > job.env.out"),
	}, build, WithJobEnvs(EnvList{
		{Key: "JOB_VAR", Value: "build-only"},
		{Key: "SHARED", Value: "from-job"},
	})))
	p.AddStage(build)

	if status := p.Run(context.Background()); status != Success {
		t.Fatalf("Pipeline status = %s, want Success", status)
	}
	got, err := os.ReadFile(filepath.Join(tmpDir, "job.env.out"))
	if err != nil {
		t.Fatalf("read job env output: %v", err)
	}
	if string(got) != "global,build-only,from-job,build_job" {
		t.Fatalf("job env output = %q, want global,build-only,from-job,build_job", got)
	}
	if got := os.Getenv("JOB_VAR"); got != "" {
		t.Fatalf("JOB_VAR leaked to process env: %q", got)
	}
	if got := os.Getenv("JOB_NAME"); got != "" {
		t.Fatalf("JOB_NAME leaked to process env: %q", got)
	}
}

func TestJobEnvsCanReferenceJobNameAndPreviousJobEnv(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir))
	build := NewStage("build", p)
	build.AddJob(NewJob("build_job", []*Action{
		NewAction(p.Shell, "printf '%s' \"$PACKAGE\" > package.out"),
	}, build, WithJobEnvs(EnvList{
		{Key: "BASE", Value: "$JOB_NAME"},
		{Key: "PACKAGE", Value: "$BASE/pkg"},
	})))
	p.AddStage(build)

	if status := p.Run(context.Background()); status != Success {
		t.Fatalf("Pipeline status = %s, want Success", status)
	}
	got, err := os.ReadFile(filepath.Join(tmpDir, "package.out"))
	if err != nil {
		t.Fatalf("read package output: %v", err)
	}
	if string(got) != "build_job/pkg" {
		t.Fatalf("package output = %q, want build_job/pkg", got)
	}
}
