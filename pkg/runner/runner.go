package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	cliroot "github.com/davidcai/taiko-bridge-cli/internal/cmd"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/spf13/cobra"
)

// rootCmdConfig carries command runtime wiring used by the runner.
type rootCmdConfig struct {
	// Stdout receives command stdout.
	Stdout io.Writer
	// Stderr receives command stderr.
	Stderr io.Writer
	// LookupEnv resolves environment variables.
	LookupEnv func(string) string
}

// Runner executes bridge-cli command tree in-process.
type Runner struct {
	newRootCmd func(cfg rootCmdConfig) *cobra.Command
}

// NewRunner creates a library-backed in-process runner.
func NewRunner() *Runner {
	return &Runner{
		newRootCmd: func(cfg rootCmdConfig) *cobra.Command {
			return cliroot.NewRootCmdWithConfig(cliroot.RootCommandConfig{
				Stdout:    cfg.Stdout,
				Stderr:    cfg.Stderr,
				LookupEnv: cfg.LookupEnv,
			})
		},
	}
}

// Run executes one command invocation and parses line-based output events.
func (r *Runner) Run(ctx context.Context, req RunRequest) (*RunResult, error) {
	if len(req.Subcommand) == 0 {
		return nil, fmt.Errorf("subcommand is required")
	}

	args := make([]string, 0, len(req.GlobalFlags)+len(req.Subcommand)+len(req.CommandFlags)+4)
	args = append(args, req.GlobalFlags...)
	args = append(args, "--format", "json")
	if strings.TrimSpace(req.PrivateKey) != "" {
		args = append(args, "--private-key", req.PrivateKey)
	}
	args = append(args, req.Subcommand...)
	args = append(args, req.CommandFlags...)

	var (
		mu           sync.Mutex
		finalPayload map[string]any
		progressLast map[string]any
		stderrLines  []string
	)

	stdoutWriter := newLineCaptureWriter(func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		obj, err := parseJSONLine(line)
		if err != nil {
			if req.OnEvent != nil {
				_ = req.OnEvent(ctx, Event{Type: EventTypeStdoutRaw, Raw: line, Timestamp: time.Now().UTC()})
			}
			return
		}

		evtType := EventTypeStdoutJSON
		if progress, ok := extractProgress(obj); ok {
			evtType = EventTypeProgress
			mu.Lock()
			progressLast = progress
			mu.Unlock()
			if req.OnProgress != nil {
				_ = req.OnProgress(ctx, progress)
			}
		} else {
			mu.Lock()
			finalPayload = obj
			mu.Unlock()
		}

		if req.OnEvent != nil {
			_ = req.OnEvent(ctx, Event{Type: evtType, Payload: obj, Raw: line, Timestamp: time.Now().UTC()})
		}
	})
	stderrWriter := newLineCaptureWriter(func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		line = redact(line, req.PrivateKey)
		mu.Lock()
		stderrLines = append(stderrLines, line)
		mu.Unlock()
		if req.OnEvent != nil {
			_ = req.OnEvent(ctx, Event{Type: EventTypeStderr, Raw: line, Timestamp: time.Now().UTC()})
		}
	})

	newRoot := r.newRootCmd
	if newRoot == nil {
		newRoot = NewRunner().newRootCmd
	}
	root := newRoot(rootCmdConfig{Stdout: stdoutWriter, Stderr: stderrWriter})
	root.SetArgs(args)

	startedAt := time.Now().UTC()
	execErr := root.ExecuteContext(ctx)
	_ = stdoutWriter.Flush()
	_ = stderrWriter.Flush()
	completedAt := time.Now().UTC()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	exitCode := 0
	if execErr != nil {
		exitCode = clierr.Code(execErr, exitcodes.Unexpected)
		errText := strings.TrimSpace(redact(execErr.Error(), req.PrivateKey))
		if errText != "" {
			mu.Lock()
			if len(stderrLines) == 0 || stderrLines[len(stderrLines)-1] != errText {
				stderrLines = append(stderrLines, errText)
			}
			mu.Unlock()
			if req.OnEvent != nil {
				_ = req.OnEvent(ctx, Event{Type: EventTypeStderr, Raw: errText, Timestamp: time.Now().UTC()})
			}
		}
	}

	mu.Lock()
	defer mu.Unlock()
	return &RunResult{
		ExitCode:     exitCode,
		FinalPayload: finalPayload,
		ProgressLast: progressLast,
		Stderr:       strings.Join(stderrLines, "\n"),
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
		Duration:     completedAt.Sub(startedAt),
	}, nil
}

// parseJSONLine parses one JSON line object.
func parseJSONLine(line string) (map[string]any, error) {
	obj := make(map[string]any)
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// extractProgress returns the nested progress object for progress payloads.
func extractProgress(payload map[string]any) (map[string]any, bool) {
	if payload == nil {
		return nil, false
	}
	typ, _ := payload["type"].(string)
	if typ != "progress" {
		return nil, false
	}
	progress, ok := payload["progress"].(map[string]any)
	if !ok {
		return nil, false
	}
	return progress, true
}

// redact replaces secret values from log output.
func redact(line string, secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return line
	}
	return strings.ReplaceAll(line, secret, "[REDACTED]")
}

// lineCaptureWriter captures line-delimited output and emits callbacks per line.
type lineCaptureWriter struct {
	mu      sync.Mutex
	partial string
	handle  func(string)
}

// newLineCaptureWriter creates a line capture writer.
func newLineCaptureWriter(handle func(string)) *lineCaptureWriter {
	return &lineCaptureWriter{handle: handle}
}

// Write appends output and emits completed lines.
func (w *lineCaptureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	data := w.partial + string(p)
	parts := strings.Split(data, "\n")
	w.partial = parts[len(parts)-1]
	lines := append([]string{}, parts[:len(parts)-1]...)
	w.mu.Unlock()

	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if w.handle != nil {
			w.handle(line)
		}
	}
	return len(p), nil
}

// Flush emits any trailing partial line.
func (w *lineCaptureWriter) Flush() error {
	w.mu.Lock()
	line := strings.TrimSuffix(w.partial, "\r")
	w.partial = ""
	w.mu.Unlock()
	if strings.TrimSpace(line) == "" {
		return nil
	}
	if w.handle != nil {
		w.handle(line)
	}
	return nil
}
