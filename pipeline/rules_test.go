package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJobRulesSkipWhenVariableIsFalse(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir), WithEnvs(EnvList{{Key: "RUN_JOB", Value: "false"}}))
	stage := NewStage("build", p)
	stage.AddJob(NewJob("build_job", []*Action{
		NewAction(p.Shell, "touch should-not-exist"),
	}, stage, WithRules([]Rule{{On: RuleOn{Value: "$RUN_JOB"}}})))
	p.AddStage(stage)

	if status := p.Run(context.Background()); status != Success {
		t.Fatalf("Pipeline status = %s, want Success", status)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "should-not-exist")); !os.IsNotExist(err) {
		t.Fatalf("skipped job created output, stat error = %v", err)
	}
}

func TestJobRulesRunWhenShellCommandReturnsZero(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir))
	stage := NewStage("build", p)
	stage.AddJob(NewJob("build_job", []*Action{
		NewAction(p.Shell, "touch should-exist"),
	}, stage, WithRules([]Rule{{On: RuleOn{Value: "test \"$JOB_NAME\" = \"build_job\""}}})))
	p.AddStage(stage)

	if status := p.Run(context.Background()); status != Success {
		t.Fatalf("Pipeline status = %s, want Success", status)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "should-exist")); err != nil {
		t.Fatalf("expected job output: %v", err)
	}
}

func TestJobRulesSkipWhenShellCommandReturnsNonZero(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWdAfterTest(t)
	p := NewPipeline("test", "1.0.0", WithShell("sh"), WithWorkdir(tmpDir))
	stage := NewStage("build", p)
	stage.AddJob(NewJob("build_job", []*Action{
		NewAction(p.Shell, "touch should-not-exist"),
	}, stage, WithRules([]Rule{{On: RuleOn{Value: "test \"$JOB_NAME\" = \"other_job\""}}})))
	p.AddStage(stage)

	if status := p.Run(context.Background()); status != Success {
		t.Fatalf("Pipeline status = %s, want Success", status)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "should-not-exist")); !os.IsNotExist(err) {
		t.Fatalf("skipped job created output, stat error = %v", err)
	}
}
