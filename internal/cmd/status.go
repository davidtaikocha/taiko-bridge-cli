package cmd

import (
	"context"
	"fmt"
	"time"

	bridgeops "github.com/davidcai/taiko-bridge-cli/internal/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/spf13/cobra"
)

func newStatusCmd(opts *rootOptions) *cobra.Command {
	var txHash string
	var eventIndex int
	var checkpointConfs uint64
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show message status for a source tx",
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
			evt, _, err := bridgeops.ReadMessageSentFromTx(ctx, rt.SrcClient, rt.SrcBridge, rt.Profile.Src.BridgeAddress, h, eventIndex)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("read MessageSent: %w", err))
			}

			st, err := rt.DstBridge.MessageStatus(&bind.CallOpts{Context: ctx}, evt.MsgHash)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("bridge.messageStatus: %w", err))
			}

			probe, err := buildProber(rt, *evt, checkpointConfs).Probe(ctx)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, err)
			}

			return rt.Printer.Emit(map[string]any{
				"action":              "status",
				"profile":             rt.Profile.Name,
				"source_tx_hash":      h.Hex(),
				"msg_hash":            evt.MsgHashHex(),
				"source_block":        evt.SourceBlock,
				"dest_message_status": st,
				"ready_probe": map[string]any{
					"ready":            probe.Ready,
					"checkpoint_block": probe.CheckpointBlock,
					"progress":         probe.Progress,
				},
			})
		},
	}

	cmd.Flags().StringVar(&txHash, "tx-hash", "", "Source tx hash containing MessageSent")
	cmd.Flags().IntVar(&eventIndex, "event-index", 0, "MessageSent event index in tx logs")
	cmd.Flags().Uint64Var(&checkpointConfs, "checkpoint-confs", 0, "Required checkpoint log confirmations")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Command timeout")
	_ = cmd.MarkFlagRequired("tx-hash")

	return cmd
}
