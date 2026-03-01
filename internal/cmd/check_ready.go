package cmd

import (
	"context"
	"fmt"
	"time"

	bridgeops "github.com/davidcai/taiko-bridge-cli/internal/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/spf13/cobra"
)

// newCheckReadyCmd builds and returns the check-ready cobra command.
func newCheckReadyCmd(opts *rootOptions) *cobra.Command {
	var txHash string
	var eventIndex int
	var checkpointConfs uint64
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "check-ready",
		Short: "Run one readiness probe and return status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			rt, err := loadRuntime(ctx, opts)
			if err != nil {
				return clierr.Wrap(exitcodes.ConfigError, err)
			}
			defer rt.close()

			h, err := parseHash(txHash, "tx-hash")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			evt, _, err := bridgeops.ReadMessageSentFromTx(ctx, rt.SrcClient, rt.SrcBridge, rt.SrcBridgeAddress, h, eventIndex)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("read MessageSent: %w", err))
			}

			res, err := buildProber(rt, *evt, checkpointConfs).Probe(ctx)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, err)
			}

			out := map[string]any{
				"action":           "check-ready",
				"ready":            res.Ready,
				"tx_hash":          h.Hex(),
				"msg_hash":         evt.MsgHashHex(),
				"source_block":     evt.SourceBlock,
				"checkpoint_block": res.CheckpointBlock,
				"progress":         res.Progress,
			}
			if res.Ready {
				out["proof"] = "0x" + res.ProofHex
			}

			return rt.Printer.Emit(out)
		},
	}

	cmd.Flags().StringVar(&txHash, "tx-hash", "", "Source tx hash containing MessageSent")
	cmd.Flags().IntVar(&eventIndex, "event-index", 0, "MessageSent event index in tx logs")
	cmd.Flags().Uint64Var(&checkpointConfs, "checkpoint-confs", 0, "Required checkpoint log confirmations")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Command timeout")
	_ = cmd.MarkFlagRequired("tx-hash")

	return cmd
}
