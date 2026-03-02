package cmd

import (
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/spf13/cobra"
)

// newAgentCmd creates helper commands intended for automation integrations.
func newAgentCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "agent", Short: "Agent helper commands"}
	cmd.AddCommand(&cobra.Command{
		Use:   "exit-codes",
		Short: "Print CLI exit code schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printJSON(opts.Format, opts.stdoutWriter(), map[string]any{
				"ok":                 exitcodes.OK,
				"config_error":       exitcodes.ConfigError,
				"validation_error":   exitcodes.Validation,
				"timeout":            exitcodes.Timeout,
				"tx_reverted":        exitcodes.TxReverted,
				"not_ready":          exitcodes.NotReady,
				"rpc_or_proof_error": exitcodes.RPCOrProof,
				"unexpected":         exitcodes.Unexpected,
			})
		},
	})
	return cmd
}
