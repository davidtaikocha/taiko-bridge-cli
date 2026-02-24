package ready

import (
	"context"
	"testing"
	"time"
)

// fakeSource is a deterministic checkpoint source for readiness tests.
type fakeSource struct {
	// calls tracks fetch invocations.
	calls int
	// eventsByCall stores events returned for each call count.
	eventsByCall map[int][]checkpointEvent
}

// Fetch returns events mapped to current call count.
func (f *fakeSource) Fetch(ctx context.Context, fromLog uint64, toLog uint64) ([]checkpointEvent, error) {
	f.calls++
	return f.eventsByCall[f.calls], nil
}

// fakeHead is a deterministic head reader for tests.
type fakeHead struct {
	// values contains block number sequence.
	values []uint64
	// idx tracks next value index.
	idx int
}

// BlockNumber returns head sequence values and repeats last value when exhausted.
func (f *fakeHead) BlockNumber(ctx context.Context) (uint64, error) {
	if len(f.values) == 0 {
		return 0, nil
	}
	if f.idx >= len(f.values) {
		return f.values[len(f.values)-1], nil
	}
	v := f.values[f.idx]
	f.idx++
	return v, nil
}

// TestProberImmediateReady verifies immediate readiness when qualifying checkpoint exists.
func TestProberImmediateReady(t *testing.T) {
	src := &fakeSource{eventsByCall: map[int][]checkpointEvent{
		1: {{CheckpointBlock: 120, LogBlock: 10}},
	}}
	head := &fakeHead{values: []uint64{20}}
	p := NewProber(
		src,
		head,
		100,
		0,
		func(ctx context.Context, checkpointBlock uint64) ([]byte, error) { return []byte{0x01}, nil },
		func(ctx context.Context, proof []byte) (bool, error) { return true, nil },
	)

	res, err := p.Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe error: %v", err)
	}
	if !res.Ready || res.CheckpointBlock != 120 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

// TestProberEventualReady verifies polling path that becomes ready in later probe.
func TestProberEventualReady(t *testing.T) {
	src := &fakeSource{eventsByCall: map[int][]checkpointEvent{
		1: {},
		2: {{CheckpointBlock: 150, LogBlock: 21}},
	}}
	head := &fakeHead{values: []uint64{20, 30, 30}}
	call := 0
	p := NewProber(
		src,
		head,
		100,
		0,
		func(ctx context.Context, checkpointBlock uint64) ([]byte, error) { return []byte{0x01}, nil },
		func(ctx context.Context, proof []byte) (bool, error) {
			call++
			return call >= 1, nil
		},
	)

	res, err := p.Wait(context.Background(), 200*time.Millisecond, 5*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Wait error: %v", err)
	}
	if !res.Ready || res.CheckpointBlock != 150 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

// TestProberTimeout verifies timeout error when readiness never becomes true.
func TestProberTimeout(t *testing.T) {
	src := &fakeSource{eventsByCall: map[int][]checkpointEvent{}}
	head := &fakeHead{values: []uint64{5, 6, 7, 8}}
	p := NewProber(
		src,
		head,
		100,
		0,
		func(ctx context.Context, checkpointBlock uint64) ([]byte, error) { return []byte{0x01}, nil },
		func(ctx context.Context, proof []byte) (bool, error) { return false, nil },
	)

	_, err := p.Wait(context.Background(), 20*time.Millisecond, 5*time.Millisecond, nil)
	if err != ErrTimeout {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}
