package cmd

import (
	"context"
	"fmt"
	"time"

	bridgeops "github.com/davidcai/taiko-bridge-cli/internal/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/config"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/davidcai/taiko-bridge-cli/internal/ready"
	"github.com/spf13/cobra"
)

// newClaimERC1155Cmd builds and returns the claim-erc1155 cobra command.
func newClaimERC1155Cmd(opts *rootOptions) *cobra.Command {
	var token string
	var to string
	var destOwner string
	var tokenIDs string
	var amounts string
	var fee string
	var gasLimit uint32
	var timeout time.Duration
	var poll time.Duration
	var checkpointConfs uint64
	var progress bool

	cmd := &cobra.Command{
		Use:     "claim-erc1155",
		Aliases: []string{"bridge-erc1155"},
		Short:   "Send ERC1155 then auto-claim on destination",
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

			tokenAddr, err := parseAddress(token, "token")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			toAddr, err := parseAddress(to, "to")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			destOwnerAddr, err := ensureDestOwner(toAddr, destOwner)
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			ids, err := parseCSVBigInts(tokenIDs, "token-ids")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			amts, err := parseCSVBigInts(amounts, "amounts")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			feeBI, err := parseBig(fee, "fee")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			destChainID, err := chainIDUint64(ctx, rt.DstClient)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("dest chain id: %w", err))
			}

			sendRes, err := bridgeops.SendERC1155(ctx, rt.SrcClient, rt.SrcERC1155Vault, rt.SrcBridge, rt.SrcBridgeAddress, pk, bridgeops.SendNFTRequest{
				DestChainID: destChainID,
				DestOwner:   destOwnerAddr,
				To:          toAddr,
				Fee:         feeBI,
				Token:       tokenAddr,
				GasLimit:    gasLimit,
				TokenIDs:    ids,
				Amounts:     amts,
			})
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("send-erc1155 failed: %w", err))
			}

			out, err := autoClaimLoop(ctx, rt, pk, sendRes.Event, checkpointConfs, timeout, poll, func(p ready.Progress) {
				if !progress {
					return
				}
				_ = rt.Printer.Emit(map[string]any{"type": "progress", "progress": p})
			})
			if err != nil {
				if err == ready.ErrTimeout {
					_ = rt.Printer.Emit(out)
					return clierr.Wrap(exitcodes.Timeout, err)
				}
				return clierr.Wrap(exitcodes.RPCOrProof, err)
			}

			out["action"] = "claim-erc1155"
			return rt.Printer.Emit(out)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Canonical token address")
	cmd.Flags().StringVar(&to, "to", "", "Destination target address")
	cmd.Flags().StringVar(&destOwner, "dest-owner", "", "Destination message owner (defaults to --to)")
	cmd.Flags().StringVar(&tokenIDs, "token-ids", "", "Comma-separated token ids")
	cmd.Flags().StringVar(&amounts, "amounts", "", "Comma-separated amounts")
	cmd.Flags().StringVar(&fee, "fee", "0", "Bridge fee in wei")
	cmd.Flags().Uint32Var(&gasLimit, "gas-limit", 450000, "Message gas limit")
	cmd.Flags().DurationVar(&timeout, "timeout", 15*time.Minute, "Pipeline timeout")
	cmd.Flags().DurationVar(&poll, "poll-interval", 3*time.Second, "Polling interval")
	cmd.Flags().Uint64Var(&checkpointConfs, "checkpoint-confs", 0, "Required checkpoint log confirmations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Emit progress JSON lines")
	_ = cmd.MarkFlagRequired("token")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("token-ids")
	_ = cmd.MarkFlagRequired("amounts")

	return cmd
}
