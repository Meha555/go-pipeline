package pipeline

import "testing"

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
