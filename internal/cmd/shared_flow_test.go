package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davidcai/taiko-bridge-cli/internal/claim"
	"github.com/davidcai/taiko-bridge-cli/internal/ready"
	bridgetypes "github.com/davidcai/taiko-bridge-cli/internal/types"
	"github.com/ethereum/go-ethereum/common"
)

type fakeProber struct {
	results []*ready.Result
	err     error
	idx     int
}

func (f *fakeProber) Probe(ctx context.Context) (*ready.Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.idx >= len(f.results) {
		return f.results[len(f.results)-1], nil
	}
	r := f.results[f.idx]
	f.idx++
	return r, nil
}

func TestAutoClaimWithProber_RetrysUntilSuccess(t *testing.T) {
	p := &fakeProber{results: []*ready.Result{
		{Ready: false, CheckpointBlock: 101, Proof: []byte{0x01}, Progress: ready.Progress{SourceBlock: 100}},
		{Ready: true, CheckpointBlock: 101, Proof: []byte{0x01}, Progress: ready.Progress{SourceBlock: 100}},
	}}
	calls := 0
	claimFn := func(ctx context.Context, proof []byte) (*claim.Result, error) {
		calls++
		if calls == 1 {
			return nil, claim.ErrNotReady
		}
		return &claim.Result{Claimed: true, TxHash: common.HexToHash("0x1234")}, nil
	}

	evt := bridgetypes.MessageSent{
		SourceTxHash: common.HexToHash("0x4567"),
		SourceBlock:  100,
	}

	out, err := autoClaimWithProber(context.Background(), p, claimFn, evt, 200*time.Millisecond, 5*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("autoClaimWithProber error: %v", err)
	}
	if out["claimed"] != true {
		t.Fatalf("expected claimed=true, got %+v", out)
	}
	if calls != 2 {
		t.Fatalf("expected 2 claim attempts, got %d", calls)
	}
}

func TestAutoClaimWithProber_Timeout(t *testing.T) {
	p := &fakeProber{results: []*ready.Result{{Ready: false, CheckpointBlock: 0, Progress: ready.Progress{SourceBlock: 100}}}}
	claimFn := func(ctx context.Context, proof []byte) (*claim.Result, error) {
		return nil, errors.New("should not be called")
	}

	evt := bridgetypes.MessageSent{SourceTxHash: common.HexToHash("0x4567"), SourceBlock: 100}
	_, err := autoClaimWithProber(context.Background(), p, claimFn, evt, 15*time.Millisecond, 5*time.Millisecond, nil)
	if err != ready.ErrTimeout {
		t.Fatalf("expected timeout, got %v", err)
	}
}
