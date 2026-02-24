package cmd

import (
	"context"
	"fmt"
	"time"

	bridgeops "github.com/davidcai/taiko-bridge-cli/internal/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/claim"
	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/config"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/davidcai/taiko-bridge-cli/internal/ready"
	"github.com/spf13/cobra"
)

// newClaimCmd builds and returns the claim cobra command.
func newClaimCmd(opts *rootOptions) *cobra.Command {
	var txHash string
	var eventIndex int
	var timeout time.Duration
	var poll time.Duration
	var checkpointConfs uint64

	cmd := &cobra.Command{
		Use:   "claim",
		Short: "Claim a MessageSent tx on destination bridge",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			rt, err := loadRuntime(ctx, opts)
			if err != nil {
				return clierr.Wrap(exitcodes.ConfigError, err)
			}
			defer rt.close()
			pk, err := config.LoadPrivateKey(opts.PrivateKey, opts.PrivateKeyEnv)
			if err != nil {
				return clierr.Wrap(exitcodes.ConfigError, err)
			}

			h, err := parseHash(txHash, "tx-hash")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			evt, _, err := bridgeops.ReadMessageSentFromTx(ctx, rt.SrcClient, rt.SrcBridge, rt.SrcBridgeAddress, h, eventIndex)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("read MessageSent: %w", err))
			}

			prober := buildProber(rt, *evt, checkpointConfs)
			waitRes, err := prober.Wait(ctx, timeout, poll, nil)
			if err != nil {
				if err == ready.ErrTimeout {
					return clierr.Wrap(exitcodes.Timeout, err)
				}
				return clierr.Wrap(exitcodes.RPCOrProof, err)
			}

			claimRes, err := claim.Process(ctx, rt.DstClient, rt.DstBridge, pk, claim.Request{
				Message: evt.Message,
				Proof:   waitRes.Proof,
			})
			if err != nil {
				if err == claim.ErrNotReady {
					return clierr.Wrap(exitcodes.NotReady, err)
				}
				if err == claim.ErrReverted {
					return clierr.Wrap(exitcodes.TxReverted, err)
				}
				return clierr.Wrap(exitcodes.RPCOrProof, err)
			}

			return rt.Printer.Emit(map[string]any{
				"action":           "claim",
				"source_tx_hash":   h.Hex(),
				"msg_hash":         evt.MsgHashHex(),
				"checkpoint_block": waitRes.CheckpointBlock,
				"claim_tx_hash":    claimRes.TxHash.Hex(),
				"claimed":          claimRes.Claimed,
			})
		},
	}

	cmd.Flags().StringVar(&txHash, "tx-hash", "", "Source tx hash containing MessageSent")
	cmd.Flags().IntVar(&eventIndex, "event-index", 0, "MessageSent event index in tx logs")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Claim timeout")
	cmd.Flags().DurationVar(&poll, "poll-interval", 3*time.Second, "Polling interval")
	cmd.Flags().Uint64Var(&checkpointConfs, "checkpoint-confs", 0, "Required checkpoint log confirmations")
	_ = cmd.MarkFlagRequired("tx-hash")

	return cmd
}
