package pipeline

import (
	"os/exec"
	"strings"
	"testing"
)

func TestFindInlineCmdUsesConfiguredShell(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}

	cmds := findInlineCmd("VALUE=$(printf ok)", [2]string{"sh", "-c"})
	if len(cmds) != 1 {
		t.Fatalf("len(cmds) = %d, want 1", len(cmds))
	}

	output, err := cmds[0].cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inline command failed: %v, output: %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "ok" {
		t.Fatalf("output = %q, want ok", output)
	}
}
