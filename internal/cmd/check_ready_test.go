package cmd

import "testing"

// TestCheckReadyCommandRegistered verifies the one-shot readiness command is available.
func TestCheckReadyCommandRegistered(t *testing.T) {
	root := NewRootCmd()

	cmd, _, err := root.Find([]string{"check-ready"})
	if err != nil {
		t.Fatalf("Find(check-ready) error: %v", err)
	}
	if cmd.Name() != "check-ready" {
		t.Fatalf("resolved command = %s, want check-ready", cmd.Name())
	}
}
