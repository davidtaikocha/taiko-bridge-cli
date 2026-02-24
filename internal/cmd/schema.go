package cmd

import (
	"os"

	"github.com/davidcai/taiko-bridge-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// newSchemaCmd builds and returns the schema cobra command.
func newSchemaCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print flag and output schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := outfmt.Printer{Format: opts.Format, Out: os.Stdout}
			return p.Emit(map[string]any{
				"required_flags": map[string]string{
					"--src-rpc":                       "string",
					"--dst-rpc":                       "string",
					"--private-key|--private-key-env": "hex private key",
				},
				"optional_overrides": map[string]string{
					"--src-bridge":         "address",
					"--dst-bridge":         "address",
					"--src-signal-service": "address",
					"--dst-signal-service": "address",
					"--src-erc20-vault":    "address",
					"--src-erc721-vault":   "address",
					"--src-erc1155-vault":  "address",
				},
				"fixed_address_resolution": "By default addresses are auto-selected from source/destination chain IDs for mainnet and hoodi.",
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
