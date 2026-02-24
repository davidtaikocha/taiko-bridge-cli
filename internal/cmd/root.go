package cmd

import (
	"fmt"
	"os"

	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	ConfigPath    string
	Profile       string
	PrivateKey    string
	PrivateKeyEnv string
	Format        string
}

func NewRootCmd() *cobra.Command {
	opts := &rootOptions{}

	rootCmd := &cobra.Command{
		Use:           "tbc",
		Short:         "Taiko Bridge CLI (Shasta)",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&opts.ConfigPath, "config", "./tbc.yaml", "Path to YAML config")
	rootCmd.PersistentFlags().StringVar(&opts.Profile, "profile", "l1_to_l2", "Profile name: l1_to_l2 or l2_to_l1")
	rootCmd.PersistentFlags().StringVar(&opts.PrivateKey, "private-key", "", "Private key hex")
	rootCmd.PersistentFlags().StringVar(&opts.PrivateKeyEnv, "private-key-env", "PRIVATE_KEY", "Env var containing private key")
	rootCmd.PersistentFlags().StringVar(&opts.Format, "format", "json", "Output format: json|text")

	rootCmd.AddCommand(
		newClaimEthCmd(opts),
		newClaimERC20Cmd(opts),
		newClaimERC721Cmd(opts),
		newClaimERC1155Cmd(opts),
		newSendEthCmd(opts),
		newSendERC20Cmd(opts),
		newSendERC721Cmd(opts),
		newSendERC1155Cmd(opts),
		newWaitReadyCmd(opts),
		newClaimCmd(opts),
		newStatusCmd(opts),
		newSchemaCmd(opts),
		newAgentCmd(opts),
	)

	return rootCmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		code := clierr.Code(err, exitcodes.Unexpected)
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
