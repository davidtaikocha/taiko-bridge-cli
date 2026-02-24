package ready

import (
	"context"
	"errors"
	"fmt"
	"time"

	signalservicebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/signalservice"
	bridgetypes "github.com/davidcai/taiko-bridge-cli/internal/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// ErrTimeout indicates wait-ready exhausted its timeout before message became ready.
var ErrTimeout = errors.New("wait-ready timeout")

// checkpointEvent is a normalized checkpoint log entry.
type checkpointEvent struct {
	// CheckpointBlock is the synced source block number in checkpoint payload.
	CheckpointBlock uint64
	// LogBlock is the destination block number where CheckpointSaved was emitted.
	LogBlock uint64
	// TxHash is the checkpoint transaction hash.
	TxHash common.Hash
}

// checkpointSource lists checkpoint events from destination signal service.
type checkpointSource interface {
	// Fetch returns checkpoint events in [fromLog, toLog] range.
	Fetch(ctx context.Context, fromLog uint64, toLog uint64) ([]checkpointEvent, error)
}

// headReader provides destination chain head block number.
type headReader interface {
	// BlockNumber returns latest known chain head.
	BlockNumber(ctx context.Context) (uint64, error)
}

// Prober implements the readiness algorithm for a single message.
type Prober struct {
	// Source provides checkpoint log data.
	Source checkpointSource
	// DestHead provides destination head block updates.
	DestHead headReader
	// CheckpointConfs is required confirmations for checkpoint log block.
	CheckpointConfs uint64
	// BuildProof builds claim proof for selected checkpoint block.
	BuildProof func(ctx context.Context, checkpointBlock uint64) ([]byte, error)
	// CheckMessage checks destination bridge IsMessageReceived.
	CheckMessage func(ctx context.Context, proof []byte) (bool, error)
	// SourceMessageBlock is source MessageSent block used for checkpoint filtering.
	SourceMessageBlock uint64

	// nextLogBlock is the next destination log block to scan from.
	nextLogBlock uint64
	// bestCheckpoint tracks highest qualifying checkpoint block seen so far.
	bestCheckpoint uint64
}

// Progress is an agent-friendly status snapshot for polling loops.
type Progress struct {
	// SourceBlock is the source MessageSent block.
	SourceBlock uint64 `json:"source_block"`
	// BestQualifyingCheckpoint is the highest checkpoint >= source block.
	BestQualifyingCheckpoint uint64 `json:"best_qualifying_checkpoint_block"`
	// LastIsMessageReceived is the last IsMessageReceived result, when evaluated.
	LastIsMessageReceived *bool `json:"last_is_message_received,omitempty"`
	// NextRetryETAUnix is the next planned retry timestamp.
	NextRetryETAUnix *int64 `json:"next_retry_eta_unix,omitempty"`
}

// Result is the output of one probe or wait cycle.
type Result struct {
	// Ready indicates whether destination currently accepts the message proof.
	Ready bool `json:"ready"`
	// CheckpointBlock is the checkpoint block used to build proof.
	CheckpointBlock uint64 `json:"checkpoint_block"`
	// Proof is raw encoded proof bytes.
	Proof []byte `json:"-"`
	// ProofHex is hex-encoded proof bytes without 0x prefix.
	ProofHex string `json:"proof_hex,omitempty"`
	// Progress contains latest polling status values.
	Progress Progress `json:"progress"`
}

// NewProber constructs a readiness prober from dependencies and message metadata.
func NewProber(
	source checkpointSource,
	destHead headReader,
	sourceMessageBlock uint64,
	checkpointConfs uint64,
	buildProof func(ctx context.Context, checkpointBlock uint64) ([]byte, error),
	checkMessage func(ctx context.Context, proof []byte) (bool, error),
) *Prober {
	return &Prober{
		Source:             source,
		DestHead:           destHead,
		SourceMessageBlock: sourceMessageBlock,
		CheckpointConfs:    checkpointConfs,
		BuildProof:         buildProof,
		CheckMessage:       checkMessage,
		nextLogBlock:       0,
	}
}

// Probe performs one readiness pass: scan checkpoints, build proof, and check message.
func (p *Prober) Probe(ctx context.Context) (*Result, error) {
	if p.Source == nil || p.DestHead == nil || p.BuildProof == nil || p.CheckMessage == nil {
		return nil, fmt.Errorf("incomplete readiness prober")
	}

	head, err := p.DestHead.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("dest head block: %w", err)
	}

	events, err := p.Source.Fetch(ctx, p.nextLogBlock, head)
	if err != nil {
		return nil, fmt.Errorf("fetch checkpoints: %w", err)
	}

	for _, e := range events {
		if e.LogBlock > head {
			continue
		}
		if p.CheckpointConfs > 0 && e.LogBlock+p.CheckpointConfs > head {
			continue
		}
		if e.CheckpointBlock < p.SourceMessageBlock {
			continue
		}
		if e.CheckpointBlock > p.bestCheckpoint {
			p.bestCheckpoint = e.CheckpointBlock
		}
	}

	p.nextLogBlock = head + 1

	progress := Progress{
		SourceBlock:              p.SourceMessageBlock,
		BestQualifyingCheckpoint: p.bestCheckpoint,
	}

	if p.bestCheckpoint == 0 {
		return &Result{Ready: false, CheckpointBlock: 0, Progress: progress}, nil
	}

	proof, err := p.BuildProof(ctx, p.bestCheckpoint)
	if err != nil {
		return nil, fmt.Errorf("build proof: %w", err)
	}

	received, err := p.CheckMessage(ctx, proof)
	if err != nil {
		return nil, fmt.Errorf("isMessageReceived: %w", err)
	}
	progress.LastIsMessageReceived = &received

	return &Result{
		Ready:           received,
		CheckpointBlock: p.bestCheckpoint,
		Proof:           proof,
		ProofHex:        common.Bytes2Hex(proof),
		Progress:        progress,
	}, nil
}

// Wait repeatedly probes until ready, timeout, or context cancellation.
func (p *Prober) Wait(
	ctx context.Context,
	timeout time.Duration,
	pollInterval time.Duration,
	onProgress func(Progress),
) (*Result, error) {
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)
	if timeout <= 0 {
		deadline = time.Now().Add(10 * time.Minute)
	}

	for {
		res, err := p.Probe(ctx)
		if err != nil {
			return nil, err
		}
		if res.Ready {
			return res, nil
		}
		if time.Now().After(deadline) {
			return res, ErrTimeout
		}

		next := time.Now().Add(pollInterval).Unix()
		res.Progress.NextRetryETAUnix = &next
		if onProgress != nil {
			onProgress(res.Progress)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// SignalServiceSource reads CheckpointSaved events from a v4 signal service.
type SignalServiceSource struct {
	// Service is the destination signal service binding.
	Service *signalservicebinding.SignalService
}

// Fetch retrieves CheckpointSaved events between fromLog and toLog inclusive.
func (s SignalServiceSource) Fetch(ctx context.Context, fromLog uint64, toLog uint64) ([]checkpointEvent, error) {
	if s.Service == nil {
		return nil, fmt.Errorf("nil signal service")
	}
	if toLog < fromLog {
		return nil, nil
	}
	iter, err := s.Service.FilterCheckpointSaved(&bind.FilterOpts{
		Start:   fromLog,
		End:     &toLog,
		Context: ctx,
	}, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	out := make([]checkpointEvent, 0)
	for iter.Next() {
		e := iter.Event
		if e == nil || e.BlockNumber == nil {
			continue
		}
		out = append(out, checkpointEvent{
			CheckpointBlock: e.BlockNumber.Uint64(),
			LogBlock:        e.Raw.BlockNumber,
			TxHash:          e.Raw.TxHash,
		})
	}
	if iter.Error() != nil {
		return nil, iter.Error()
	}
	return out, nil
}

// FromMessageEvent creates a prober from source message event data and dependencies.
func FromMessageEvent(
	e bridgetypes.MessageSent,
	checkpointConfs uint64,
	source checkpointSource,
	destHead headReader,
	buildProof func(ctx context.Context, checkpointBlock uint64) ([]byte, error),
	checkMessage func(ctx context.Context, proof []byte) (bool, error),
) *Prober {
	return NewProber(source, destHead, e.SourceBlock, checkpointConfs, buildProof, checkMessage)
}
