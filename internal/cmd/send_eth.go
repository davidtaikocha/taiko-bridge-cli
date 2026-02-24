package cmd

import (
	"context"
	"fmt"
	"time"

	bridgeops "github.com/davidcai/taiko-bridge-cli/internal/bridge"
	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/config"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

// newSendEthCmd builds and returns the send-eth cobra command.
func newSendEthCmd(opts *rootOptions) *cobra.Command {
	var to string
	var destOwner string
	var value string
	var fee string
	var gasLimit uint32
	var dataHex string
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "send-eth",
		Short: "Send ETH bridge message on source bridge",
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

			toAddr, err := parseAddress(to, "to")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			destOwnerAddr, err := ensureDestOwner(toAddr, destOwner)
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			valueBI, err := parseBig(value, "value")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			feeBI, err := parseBig(fee, "fee")
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			dataBytes, err := parseBytesHex(dataHex)
			if err != nil {
				return clierr.Wrap(exitcodes.Validation, err)
			}
			destChainID, err := chainIDUint64(ctx, rt.DstClient)
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("dest chain id: %w", err))
			}

			from := crypto.PubkeyToAddress(pk.PublicKey)
			res, err := bridgeops.SendETH(ctx, rt.SrcClient, rt.SrcBridge, rt.SrcBridgeAddress, pk, bridgeops.SendETHRequest{
				From:        from,
				DestChainID: destChainID,
				DestOwner:   destOwnerAddr,
				To:          toAddr,
				Value:       valueBI,
				Fee:         feeBI,
				GasLimit:    gasLimit,
				Data:        dataBytes,
			})
			if err != nil {
				return clierr.Wrap(exitcodes.RPCOrProof, fmt.Errorf("send-eth failed: %w", err))
			}

			return rt.Printer.Emit(map[string]any{
				"action":           "send-eth",
				"tx_hash":          res.TxHash.Hex(),
				"message":          bridgeMessageToMap(res.Event.Message),
				"msg_hash":         res.Event.MsgHashHex(),
				"source_block":     res.Event.SourceBlock,
				"source_log_index": res.Event.SourceLogIdx,
			})
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Destination target address")
	cmd.Flags().StringVar(&destOwner, "dest-owner", "", "Destination message owner (defaults to --to)")
	cmd.Flags().StringVar(&value, "value", "", "ETH amount in wei")
	cmd.Flags().StringVar(&fee, "fee", "0", "Bridge fee in wei")
	cmd.Flags().Uint32Var(&gasLimit, "gas-limit", 140000, "Message gas limit")
	cmd.Flags().StringVar(&dataHex, "data", "", "Optional calldata as 0x-prefixed hex")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Command timeout")

	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}
