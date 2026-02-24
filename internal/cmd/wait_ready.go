package cmd

import (
	"context"
	"fmt"
	"time"

	bridgeops "github.com/davidcai/taiko-bridge-cli/internal/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/davidcai/taiko-bridge-cli/internal/ready"
	"github.com/spf13/cobra"
)

// newWaitReadyCmd builds and returns the wait-ready cobra command.
func newWaitReadyCmd(opts *rootOptions) *cobra.Command {
	var txHash string
	var eventIndex int
	var timeout time.Duration
	var poll time.Duration
	var checkpointConfs uint64
	var progress bool

	cmd := &cobra.Command{
		Use:   "wait-ready",
		Short: "Wait for destination readiness and return proof",
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

			prober := buildProber(rt, *evt, checkpointConfs)
			res, err := prober.Wait(ctx, timeout, poll, func(p ready.Progress) {
				if !progress {
					return
				}
				_ = rt.Printer.Emit(map[string]any{
					"type":     "progress",
					"progress": p,
				})
			})
			if err != nil {
				if err == ready.ErrTimeout {
					_ = rt.Printer.Emit(map[string]any{
						"action":   "wait-ready",
						"ready":    false,
						"tx_hash":  h.Hex(),
						"msg_hash": evt.MsgHashHex(),
						"progress": res.Progress,
					})
					return clierr.Wrap(exitcodes.Timeout, err)
				}
				return clierr.Wrap(exitcodes.RPCOrProof, err)
			}

			return rt.Printer.Emit(map[string]any{
				"action":           "wait-ready",
				"ready":            true,
				"tx_hash":          h.Hex(),
				"msg_hash":         evt.MsgHashHex(),
				"checkpoint_block": res.CheckpointBlock,
				"proof":            "0x" + res.ProofHex,
				"progress":         res.Progress,
			})
		},
	}

	cmd.Flags().StringVar(&txHash, "tx-hash", "", "Source tx hash containing MessageSent")
	cmd.Flags().IntVar(&eventIndex, "event-index", 0, "MessageSent event index inside tx logs")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Wait timeout")
	cmd.Flags().DurationVar(&poll, "poll-interval", 3*time.Second, "Polling interval")
	cmd.Flags().Uint64Var(&checkpointConfs, "checkpoint-confs", 0, "Required checkpoint log confirmations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Emit progress JSON lines")
	_ = cmd.MarkFlagRequired("tx-hash")

	return cmd
}
