package pipeline

import (
	"os"
	"os/exec"
	"testing"
)

func TestStageAddJobRejectsDuplicateJobName(t *testing.T) {
	if os.Getenv("TEST_DUPLICATE_JOB") == "1" {
		p := NewPipeline("test", "1.0.0")
		s := NewStage("build", p)
		job1 := NewJob("compile", nil, s)
		job2 := NewJob("compile", nil, s)
		s.AddJob(job1)
		s.AddJob(job2)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestStageAddJobRejectsDuplicateJobName")
	cmd.Env = append(os.Environ(), "TEST_DUPLICATE_JOB=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected process to exit with non-zero status")
	}
}
