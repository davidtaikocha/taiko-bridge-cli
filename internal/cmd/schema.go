package cmd

import (
	"os"

	"github.com/davidcai/taiko-bridge-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newSchemaCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print config and output schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := outfmt.Printer{Format: opts.Format, Out: os.Stdout}
			return p.Emit(map[string]any{
				"config_yaml": map[string]any{
					"profiles.<name>.src": map[string]string{
						"rpc_url":        "string",
						"bridge":         "address",
						"signal_service": "address",
						"erc20_vault":    "address",
						"erc721_vault":   "address",
						"erc1155_vault":  "address",
					},
					"profiles.<name>.dest": map[string]string{
						"rpc_url":        "string",
						"bridge":         "address",
						"signal_service": "address",
						"erc20_vault":    "address",
						"erc721_vault":   "address",
						"erc1155_vault":  "address",
					},
				},
				"command_outputs": []string{
					"send-* => tx_hash,msg_hash,source_block",
					"wait-ready => ready,checkpoint_block,proof,progress",
					"claim => claim_tx_hash,claimed",
					"claim-* => send+claim pipeline output",
					"status => message status + readiness probe",
				},
			})
		},
	}
}
