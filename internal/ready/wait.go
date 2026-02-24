package ready

import (
	"context"
	"errors"
	"fmt"
	"time"

	signalservicebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/v4/signalservice"
	bridgetypes "github.com/davidcai/taiko-bridge-cli/internal/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var ErrTimeout = errors.New("wait-ready timeout")

type checkpointEvent struct {
	CheckpointBlock uint64
	LogBlock        uint64
	TxHash          common.Hash
}

type checkpointSource interface {
	Fetch(ctx context.Context, fromLog uint64, toLog uint64) ([]checkpointEvent, error)
}

type headReader interface {
	BlockNumber(ctx context.Context) (uint64, error)
}

type Prober struct {
	Source             checkpointSource
	DestHead           headReader
	CheckpointConfs    uint64
	BuildProof         func(ctx context.Context, checkpointBlock uint64) ([]byte, error)
	CheckMessage       func(ctx context.Context, proof []byte) (bool, error)
	SourceMessageBlock uint64

	nextLogBlock   uint64
	bestCheckpoint uint64
}

type Progress struct {
	SourceBlock              uint64 `json:"source_block"`
	BestQualifyingCheckpoint uint64 `json:"best_qualifying_checkpoint_block"`
	LastIsMessageReceived    *bool  `json:"last_is_message_received,omitempty"`
	NextRetryETAUnix         *int64 `json:"next_retry_eta_unix,omitempty"`
}

type Result struct {
	Ready           bool     `json:"ready"`
	CheckpointBlock uint64   `json:"checkpoint_block"`
	Proof           []byte   `json:"-"`
	ProofHex        string   `json:"proof_hex,omitempty"`
	Progress        Progress `json:"progress"`
}

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

type SignalServiceSource struct {
	Service *signalservicebinding.SignalService
}

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
