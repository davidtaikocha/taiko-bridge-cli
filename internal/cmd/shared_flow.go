package cmd

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"regexp"
	"time"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/claim"
	"github.com/davidcai/taiko-bridge-cli/internal/proof"
	"github.com/davidcai/taiko-bridge-cli/internal/ready"
	bridgetypes "github.com/davidcai/taiko-bridge-cli/internal/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// probeRunner abstracts readiness probing for production code and tests.
type probeRunner interface {
	// Probe performs one readiness check attempt.
	Probe(ctx context.Context) (*ready.Result, error)
}

// buildProber wires chain clients/services into a readiness prober instance.
func buildProber(rt *runtime, evt bridgetypes.MessageSent, checkpointConfs uint64) *ready.Prober {
	return ready.FromMessageEvent(
		evt,
		checkpointConfs,
		ready.SignalServiceSource{Service: rt.DstSignalService},
		rt.DstClient,
		func(ctx context.Context, checkpointBlock uint64) ([]byte, error) {
			return proof.Build(ctx, proof.BuildRequest{
				RPC:                rt.SrcClient.Client(),
				BlockReader:        rt.SrcClient,
				SignalService:      rt.SrcSignalService,
				SignalServiceAddr:  rt.SrcSignalAddress,
				HopChainID:         evt.Message.DestChainId,
				Message:            evt,
				CheckpointBlockNum: checkpointBlock,
			})
		},
		func(ctx context.Context, pf []byte) (bool, error) {
			return rt.DstBridge.IsMessageReceived(&bind.CallOpts{Context: ctx}, evt.Message, pf)
		},
	)
}

// autoClaimLoop runs the pipeline claim retry flow using the default runtime prober.
func autoClaimLoop(
	ctx context.Context,
	rt *runtime,
	privateKey *ecdsa.PrivateKey,
	evt bridgetypes.MessageSent,
	checkpointConfs uint64,
	timeout time.Duration,
	poll time.Duration,
	onProgress func(ready.Progress),
) (map[string]any, error) {
	prober := buildProber(rt, evt, checkpointConfs)
	return autoClaimWithProber(
		ctx,
		prober,
		func(ctx context.Context, proof []byte) (*claim.Result, error) {
			return claim.Process(ctx, rt.DstClient, rt.DstBridge, privateKey, claim.Request{
				Message: evt.Message,
				Proof:   proof,
			})
		},
		evt,
		timeout,
		poll,
		onProgress,
	)
}

// autoClaimWithProber retries readiness checks and claim attempts until success or timeout.
func autoClaimWithProber(
	ctx context.Context,
	prober probeRunner,
	claimFn func(ctx context.Context, proof []byte) (*claim.Result, error),
	evt bridgetypes.MessageSent,
	timeout time.Duration,
	poll time.Duration,
	onProgress func(ready.Progress),
) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	if timeout <= 0 {
		deadline = time.Now().Add(10 * time.Minute)
	}
	if poll <= 0 {
		poll = 3 * time.Second
	}

	for {
		res, err := prober.Probe(ctx)
		if err != nil {
			return nil, err
		}

		if onProgress != nil {
			next := time.Now().Add(poll).Unix()
			res.Progress.NextRetryETAUnix = &next
			onProgress(res.Progress)
		}

		if res.CheckpointBlock > 0 {
			claimRes, err := claimFn(ctx, res.Proof)
			if err == nil {
				return map[string]any{
					"send_tx_hash":     evt.SourceTxHash.Hex(),
					"msg_hash":         evt.MsgHashHex(),
					"source_block":     evt.SourceBlock,
					"checkpoint_block": res.CheckpointBlock,
					"claim_tx_hash":    claimRes.TxHash.Hex(),
					"claimed":          true,
				}, nil
			}
			if err != claim.ErrNotReady {
				return nil, err
			}
		}

		if time.Now().After(deadline) {
			return map[string]any{
				"send_tx_hash": evt.SourceTxHash.Hex(),
				"msg_hash":     evt.MsgHashHex(),
				"source_block": evt.SourceBlock,
				"claimed":      false,
			}, ready.ErrTimeout
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poll):
		}
	}
}

// parseHash validates and parses a 32-byte transaction/message hash.
func parseHash(v string, field string) (common.Hash, error) {
	matched, _ := regexp.MatchString("^0x[0-9a-fA-F]{64}$", v)
	if !matched {
		return common.Hash{}, fmt.Errorf("invalid %s", field)
	}
	return common.HexToHash(v), nil
}

// bridgeMessageToMap converts a bridge message into JSON-friendly output fields.
func bridgeMessageToMap(m bridgebinding.IBridgeMessage) map[string]any {
	return map[string]any{
		"id":            m.Id,
		"fee":           m.Fee,
		"gas_limit":     m.GasLimit,
		"from":          m.From.Hex(),
		"src_chain_id":  m.SrcChainId,
		"src_owner":     m.SrcOwner.Hex(),
		"dest_chain_id": m.DestChainId,
		"dest_owner":    m.DestOwner.Hex(),
		"to":            m.To.Hex(),
		"value":         m.Value.String(),
		"data":          "0x" + common.Bytes2Hex(m.Data),
	}
}
