package pipeline

import "testing"

func TestNewPipelineSetsDefaultShell(t *testing.T) {
	defaultShell, defaultFlag := GetDefaultShell()
	p := NewPipeline("test", "1.0.0")

	if p.Shell != [2]string{defaultShell, defaultFlag} {
		t.Fatalf("Shell = %#v, want %#v", p.Shell, [2]string{defaultShell, defaultFlag})
	}
}

func TestNewPipelineWithEmptyShellUsesDefaultShell(t *testing.T) {
	defaultShell, defaultFlag := GetDefaultShell()
	p := NewPipeline("test", "1.0.0", WithShell(""))

	if p.Shell != [2]string{defaultShell, defaultFlag} {
		t.Fatalf("Shell = %#v, want %#v", p.Shell, [2]string{defaultShell, defaultFlag})
	}
}

func TestNewPipelineSupportsPowerShell(t *testing.T) {
	p := NewPipeline("test", "1.0.0", WithShell("powershell"))

	if p.Shell != [2]string{"powershell", "-Command"} {
		t.Fatalf("Shell = %#v, want powershell command shell", p.Shell)
	}
}
