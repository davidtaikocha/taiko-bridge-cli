package runner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/spf13/cobra"
)

func TestRunner_Run_SchemaSuccess(t *testing.T) {
	r := NewRunner()
	res, err := r.Run(context.Background(), RunRequest{Subcommand: []string{"schema"}})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", res.ExitCode, res.Stderr)
	}
	if res.FinalPayload == nil {
		t.Fatal("expected final payload")
	}
	if _, ok := res.FinalPayload["required_flags"]; !ok {
		t.Fatalf("expected required_flags in payload: %#v", res.FinalPayload)
	}
}

func TestRunner_Run_ProgressAndFinalEvents(t *testing.T) {
	r := &Runner{
		newRootCmd: func(cfg rootCmdConfig) *cobra.Command {
			cmd := &cobra.Command{Use: "bridge-cli", SilenceUsage: true, SilenceErrors: true}
			cmd.PersistentFlags().String("format", "json", "")
			cmd.PersistentFlags().String("private-key", "", "")
			cmd.AddCommand(&cobra.Command{
				Use: "fake",
				RunE: func(cmd *cobra.Command, args []string) error {
					_, _ = cfg.Stdout.Write([]byte("{\"type\":\"progress\",\"progress\":{\"attempt\":1}}\n"))
					_, _ = cfg.Stdout.Write([]byte("{\"action\":\"claim\",\"claimed\":true}\n"))
					return nil
				},
			})
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				_, _ = cfg.Stdout.Write([]byte("{\"type\":\"progress\",\"progress\":{\"attempt\":1}}\n"))
				_, _ = cfg.Stdout.Write([]byte("{\"action\":\"claim\",\"claimed\":true}\n"))
				return nil
			}
			return cmd
		},
	}

	progressCalls := 0
	events := 0
	res, err := r.Run(context.Background(), RunRequest{
		Subcommand: []string{"fake"},
		OnProgress: func(ctx context.Context, progress map[string]any) error {
			progressCalls++
			if progress["attempt"] != float64(1) {
				t.Fatalf("unexpected progress: %#v", progress)
			}
			return nil
		},
		OnEvent: func(ctx context.Context, event Event) error {
			events++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if progressCalls != 1 {
		t.Fatalf("expected 1 progress callback, got %d", progressCalls)
	}
	if events < 2 {
		t.Fatalf("expected at least 2 events, got %d", events)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", res.ExitCode)
	}
	if res.FinalPayload["action"] != "claim" {
		t.Fatalf("unexpected final payload: %#v", res.FinalPayload)
	}
	if res.ProgressLast["attempt"] != float64(1) {
		t.Fatalf("unexpected progress_last: %#v", res.ProgressLast)
	}
}

func TestRunner_Run_NonZeroExitCodeFromCLIError(t *testing.T) {
	r := &Runner{
		newRootCmd: func(cfg rootCmdConfig) *cobra.Command {
			cmd := &cobra.Command{Use: "bridge-cli", SilenceUsage: true, SilenceErrors: true}
			cmd.PersistentFlags().String("format", "json", "")
			cmd.PersistentFlags().String("private-key", "", "")
			cmd.AddCommand(&cobra.Command{
				Use: "fake",
				RunE: func(cmd *cobra.Command, args []string) error {
					return clierr.Wrap(exitcodes.Validation, errors.New("invalid input"))
				},
			})
			return cmd
		},
	}

	res, err := r.Run(context.Background(), RunRequest{Subcommand: []string{"fake"}})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if res.ExitCode != exitcodes.Validation {
		t.Fatalf("expected exit code %d, got %d", exitcodes.Validation, res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "invalid input") {
		t.Fatalf("expected stderr to include error text, got %q", res.Stderr)
	}
}

func TestRunner_Run_StderrRedactsPrivateKey(t *testing.T) {
	secret := "0xsupersecret"
	r := &Runner{
		newRootCmd: func(cfg rootCmdConfig) *cobra.Command {
			cmd := &cobra.Command{Use: "bridge-cli", SilenceUsage: true, SilenceErrors: true}
			cmd.PersistentFlags().String("format", "json", "")
			cmd.PersistentFlags().String("private-key", "", "")
			cmd.AddCommand(&cobra.Command{
				Use: "fake",
				RunE: func(cmd *cobra.Command, args []string) error {
					_, _ = cfg.Stderr.Write([]byte("boom " + secret + "\n"))
					return clierr.Wrap(exitcodes.Unexpected, errors.New("err "+secret))
				},
			})
			return cmd
		},
	}

	res, err := r.Run(context.Background(), RunRequest{Subcommand: []string{"fake"}, PrivateKey: secret})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if strings.Contains(res.Stderr, secret) {
		t.Fatalf("stderr leaked secret: %q", res.Stderr)
	}
}
