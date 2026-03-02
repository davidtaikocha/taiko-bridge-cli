package runner

import (
	"context"
	"time"
)

// EventType classifies runtime stream events.
type EventType string

const (
	// EventTypeStdoutJSON marks a parsed JSON line from stdout.
	EventTypeStdoutJSON EventType = "stdout_json"
	// EventTypeStdoutRaw marks an unparseable/non-JSON stdout line.
	EventTypeStdoutRaw EventType = "stdout_raw"
	// EventTypeStderr marks a stderr line.
	EventTypeStderr EventType = "stderr"
	// EventTypeProgress marks a parsed progress JSON event.
	EventTypeProgress EventType = "progress"
)

// Event is emitted while a command is running.
type Event struct {
	// Type indicates which stream/event category was observed.
	Type EventType
	// Payload contains parsed JSON for JSON-bearing event types.
	Payload map[string]any
	// Raw contains the original output line for stdout/stderr events.
	Raw string
	// Timestamp records when the event was observed.
	Timestamp time.Time
}

// RunRequest configures one command invocation.
type RunRequest struct {
	// Subcommand is the command path, for example []string{"claim-eth"}.
	Subcommand []string
	// GlobalFlags are root-level flags (rpc endpoints, overrides).
	GlobalFlags []string
	// CommandFlags are subcommand-specific flags.
	CommandFlags []string
	// PrivateKey is a per-request secret passed as --private-key.
	PrivateKey string
	// OnProgress receives parsed progress payloads while the command runs.
	OnProgress func(ctx context.Context, progress map[string]any) error
	// OnEvent receives stream events (stdout/stderr/json/progress).
	OnEvent func(ctx context.Context, event Event) error
}

// RunResult captures a completed invocation.
type RunResult struct {
	// ExitCode is the command exit status (0 on success).
	ExitCode int
	// FinalPayload is the last non-progress JSON object seen on stdout.
	FinalPayload map[string]any
	// ProgressLast is the latest parsed progress payload.
	ProgressLast map[string]any
	// Stderr is newline-joined, redacted stderr output.
	Stderr string
	// StartedAt is when execution began.
	StartedAt time.Time
	// CompletedAt is when execution completed.
	CompletedAt time.Time
	// Duration is the wall-clock execution duration.
	Duration time.Duration
}
