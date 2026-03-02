package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/davidcai/taiko-bridge-cli/internal/clierr"
	"github.com/davidcai/taiko-bridge-cli/internal/exitcodes"
	"github.com/spf13/cobra"
)

// RootCommandConfig configures command tree I/O and env lookups.
type RootCommandConfig struct {
	// Stdout receives command output lines.
	Stdout io.Writer
	// Stderr receives command error output.
	Stderr io.Writer
	// LookupEnv resolves environment variables used by command runtime.
	LookupEnv func(string) string
}

// rootOptions stores global CLI flags shared by all subcommands.
type rootOptions struct {
	// SrcRPCURL is the source chain RPC endpoint.
	SrcRPCURL string
	// DstRPCURL is the destination chain RPC endpoint.
	DstRPCURL string
	// SrcBridge is the source Bridge contract address.
	SrcBridge string
	// DstBridge is the destination Bridge contract address.
	DstBridge string
	// SrcSignal is the source SignalService contract address.
	SrcSignal string
	// DstSignal is the destination SignalService contract address.
	DstSignal string
	// SrcERC20Vault is the source ERC20Vault contract address.
	SrcERC20Vault string
	// SrcERC721Vault is the source ERC721Vault contract address.
	SrcERC721Vault string
	// SrcERC1155Vault is the source ERC1155Vault contract address.
	SrcERC1155Vault string
	// PrivateKey is an optional inline private key value.
	PrivateKey string
	// PrivateKeyEnv is the env var name used to read private key value.
	PrivateKeyEnv string
	// Format controls output formatting.
	Format string
	// Stdout receives command output lines.
	Stdout io.Writer
	// Stderr receives command error output.
	Stderr io.Writer
	// LookupEnv resolves environment variables used by command runtime.
	LookupEnv func(string) string
}

// stdoutWriter returns configured stdout fallbacking to os.Stdout.
func (o *rootOptions) stdoutWriter() io.Writer {
	if o != nil && o.Stdout != nil {
		return o.Stdout
	}
	return os.Stdout
}

// stderrWriter returns configured stderr fallbacking to os.Stderr.
func (o *rootOptions) stderrWriter() io.Writer {
	if o != nil && o.Stderr != nil {
		return o.Stderr
	}
	return os.Stderr
}

// getEnv resolves environment values using configured lookup fallbacking to os.Getenv.
func (o *rootOptions) getEnv(key string) string {
	if o != nil && o.LookupEnv != nil {
		return o.LookupEnv(key)
	}
	return os.Getenv(key)
}

// NewRootCmd builds the top-level bridge-cli cobra command.
func NewRootCmd() *cobra.Command {
	return NewRootCmdWithConfig(RootCommandConfig{})
}

// NewRootCmdWithConfig builds the top-level bridge-cli cobra command with injected runtime config.
func NewRootCmdWithConfig(cfg RootCommandConfig) *cobra.Command {
	opts := &rootOptions{
		Stdout:    cfg.Stdout,
		Stderr:    cfg.Stderr,
		LookupEnv: cfg.LookupEnv,
	}

	rootCmd := &cobra.Command{
		Use:           "bridge-cli",
		Short:         "Taiko Bridge CLI (Shasta)",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.SetOut(opts.stdoutWriter())
	rootCmd.SetErr(opts.stderrWriter())

	rootCmd.PersistentFlags().StringVar(&opts.SrcRPCURL, "src-rpc", "", "Source RPC URL")
	rootCmd.PersistentFlags().StringVar(&opts.DstRPCURL, "dst-rpc", "", "Destination RPC URL")
	rootCmd.PersistentFlags().StringVar(&opts.SrcBridge, "src-bridge", "", "Source bridge address (optional override)")
	rootCmd.PersistentFlags().StringVar(&opts.DstBridge, "dst-bridge", "", "Destination bridge address (optional override)")
	rootCmd.PersistentFlags().StringVar(&opts.SrcSignal, "src-signal-service", "", "Source signal service address (optional override)")
	rootCmd.PersistentFlags().StringVar(&opts.DstSignal, "dst-signal-service", "", "Destination signal service address (optional override)")
	rootCmd.PersistentFlags().StringVar(&opts.SrcERC20Vault, "src-erc20-vault", "", "Source ERC20 vault address (optional override)")
	rootCmd.PersistentFlags().StringVar(&opts.SrcERC721Vault, "src-erc721-vault", "", "Source ERC721 vault address (optional override)")
	rootCmd.PersistentFlags().StringVar(&opts.SrcERC1155Vault, "src-erc1155-vault", "", "Source ERC1155 vault address (optional override)")
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
		newCheckReadyCmd(opts),
		newWaitReadyCmd(opts),
		newClaimCmd(opts),
		newStatusCmd(opts),
		newSchemaCmd(opts),
		newAgentCmd(opts),
	)

	return rootCmd
}

// Execute runs the command tree and exits with CLI-mapped codes.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		code := clierr.Code(err, exitcodes.Unexpected)
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
