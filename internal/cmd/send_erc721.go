package cmd

import (
	"context"
	"fmt"
	"time"

	bridgeops "github.com/davidcai/taiko-bridge-cli/internal/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/config"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/spf13/cobra"
)

func newSendERC721Cmd(opts *rootOptions) *cobra.Command {
	var token string
	var to string
	var destOwner string
	var tokenIDs string
	var amounts string
	var fee string
	var gasLimit uint32
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "send-erc721",
		Short: "Send ERC721 via source ERC721Vault.sendToken",
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
			amts, err := parseCSVBigIntsOptional(amounts)
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

			res, err := bridgeops.SendERC721(ctx, rt.SrcClient, rt.SrcERC721Vault, rt.SrcBridge, rt.Profile.Src.BridgeAddress, pk, bridgeops.SendNFTRequest{
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
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("send-erc721 failed: %w", err))
			}

			return rt.Printer.Emit(map[string]any{
				"action":           "send-erc721",
				"profile":          rt.Profile.Name,
				"tx_hash":          res.TxHash.Hex(),
				"message":          bridgeMessageToMap(res.Event.Message),
				"msg_hash":         res.Event.MsgHashHex(),
				"source_block":     res.Event.SourceBlock,
				"source_log_index": res.Event.SourceLogIdx,
			})
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Canonical token address")
	cmd.Flags().StringVar(&to, "to", "", "Destination target address")
	cmd.Flags().StringVar(&destOwner, "dest-owner", "", "Destination message owner (defaults to --to)")
	cmd.Flags().StringVar(&tokenIDs, "token-ids", "", "Comma-separated token ids")
	cmd.Flags().StringVar(&amounts, "amounts", "", "Comma-separated amounts (optional; default all 1)")
	cmd.Flags().StringVar(&fee, "fee", "0", "Bridge fee in wei")
	cmd.Flags().Uint32Var(&gasLimit, "gas-limit", 350000, "Message gas limit")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Command timeout")
	_ = cmd.MarkFlagRequired("token")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("token-ids")
	return cmd
}
