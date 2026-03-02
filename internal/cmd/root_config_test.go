package cmd

import (
	"bytes"
	"context"
	"testing"
)

func TestNewRootCmdWithConfig_UsesConfiguredWriters(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	root := NewRootCmdWithConfig(RootCommandConfig{Stdout: &out, Stderr: &errOut})
	root.SetArgs([]string{"schema"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected stdout output")
	}
}
