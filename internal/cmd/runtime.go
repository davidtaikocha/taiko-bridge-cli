package cmd

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	erc1155binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc1155vault"
	erc20binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc20vault"
	erc721binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc721vault"
	signalservicebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/signalservice"
	"github.com/davidcai/taiko-bridge-cli/internal/config"
	"github.com/davidcai/taiko-bridge-cli/internal/outfmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// runtime contains initialized clients, bindings, and normalized settings for commands.
type runtime struct {
	// Printer emits command output.
	Printer outfmt.Printer

	// SrcClient is the source chain RPC client.
	SrcClient *ethclient.Client
	// DstClient is the destination chain RPC client.
	DstClient *ethclient.Client

	// SrcBridgeAddress is the source Bridge contract address.
	SrcBridgeAddress common.Address
	// DstBridgeAddress is the destination Bridge contract address.
	DstBridgeAddress common.Address
	// SrcSignalAddress is the source SignalService contract address.
	SrcSignalAddress common.Address
	// DstSignalAddress is the destination SignalService contract address.
	DstSignalAddress common.Address
	// SrcERC20Address is the source ERC20Vault contract address.
	SrcERC20Address common.Address
	// SrcERC721Address is the source ERC721Vault contract address.
	SrcERC721Address common.Address
	// SrcERC1155Address is the source ERC1155Vault contract address.
	SrcERC1155Address common.Address

	// SrcBridge is the source Bridge binding.
	SrcBridge *bridgebinding.Bridge
	// DstBridge is the destination Bridge binding.
	DstBridge *bridgebinding.Bridge

	// SrcERC20Vault is the source ERC20Vault binding.
	SrcERC20Vault *erc20binding.ERC20Vault
	// SrcERC721Vault is the source ERC721Vault binding.
	SrcERC721Vault *erc721binding.ERC721Vault
	// SrcERC1155Vault is the source ERC1155Vault binding.
	SrcERC1155Vault *erc1155binding.ERC1155Vault

	// DstSignalService is the destination SignalService binding.
	DstSignalService *signalservicebinding.SignalService
	// SrcSignalService is the source SignalService binding.
	SrcSignalService *signalservicebinding.SignalService

	// PrivateKeyHex is the normalized private key value from flags/env.
	PrivateKeyHex string
}

// loadRuntime validates root flags and creates all chain clients and bindings.
func loadRuntime(ctx context.Context, opts *rootOptions) (*runtime, error) {
	srcRPCURL, err := requireFlag(opts.SrcRPCURL, "src-rpc")
	if err != nil {
		return nil, err
	}
	dstRPCURL, err := requireFlag(opts.DstRPCURL, "dst-rpc")
	if err != nil {
		return nil, err
	}

	srcClient, err := ethclient.DialContext(ctx, srcRPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial src rpc: %w", err)
	}
	dstClient, err := ethclient.DialContext(ctx, dstRPCURL)
	if err != nil {
		srcClient.Close()
		return nil, fmt.Errorf("dial dst rpc: %w", err)
	}

	srcChainID, err := chainIDUint64(ctx, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("src chain id: %w", err)
	}
	dstChainID, err := chainIDUint64(ctx, dstClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("dst chain id: %w", err)
	}

	srcPreset, srcPresetOK := config.LookupFixedAddresses(srcChainID)
	dstPreset, dstPresetOK := config.LookupFixedAddresses(dstChainID)

	srcBridgeAddr, err := resolveAddressWithPreset(opts.SrcBridge, "src-bridge", srcPreset.Bridge, srcPresetOK, srcChainID)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, err
	}
	dstBridgeAddr, err := resolveAddressWithPreset(opts.DstBridge, "dst-bridge", dstPreset.Bridge, dstPresetOK, dstChainID)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, err
	}
	srcSignalAddr, err := resolveAddressWithPreset(opts.SrcSignal, "src-signal-service", srcPreset.SignalService, srcPresetOK, srcChainID)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, err
	}
	dstSignalAddr, err := resolveAddressWithPreset(opts.DstSignal, "dst-signal-service", dstPreset.SignalService, dstPresetOK, dstChainID)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, err
	}
	srcERC20Addr, err := resolveAddressWithPreset(opts.SrcERC20Vault, "src-erc20-vault", srcPreset.ERC20Vault, srcPresetOK, srcChainID)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, err
	}
	srcERC721Addr, err := resolveAddressWithPreset(opts.SrcERC721Vault, "src-erc721-vault", srcPreset.ERC721Vault, srcPresetOK, srcChainID)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, err
	}
	srcERC1155Addr, err := resolveAddressWithPreset(opts.SrcERC1155Vault, "src-erc1155-vault", srcPreset.ERC1155Vault, srcPresetOK, srcChainID)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, err
	}

	srcBridge, err := bridgebinding.NewBridge(srcBridgeAddr, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src bridge binding: %w", err)
	}
	dstBridge, err := bridgebinding.NewBridge(dstBridgeAddr, dstClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new dst bridge binding: %w", err)
	}

	srcERC20, err := erc20binding.NewERC20Vault(srcERC20Addr, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src erc20 vault binding: %w", err)
	}
	srcERC721, err := erc721binding.NewERC721Vault(srcERC721Addr, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src erc721 vault binding: %w", err)
	}
	srcERC1155, err := erc1155binding.NewERC1155Vault(srcERC1155Addr, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src erc1155 vault binding: %w", err)
	}

	srcSignal, err := signalservicebinding.NewSignalService(srcSignalAddr, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src signalservice binding: %w", err)
	}
	dstSignal, err := signalservicebinding.NewSignalService(dstSignalAddr, dstClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new dst signalservice binding: %w", err)
	}

	pk := strings.TrimSpace(opts.PrivateKey)
	if pk == "" && strings.TrimSpace(opts.PrivateKeyEnv) != "" {
		pk = strings.TrimSpace(opts.getEnv(strings.TrimSpace(opts.PrivateKeyEnv)))
	}

	return &runtime{
		Printer:           outfmt.Printer{Format: opts.Format, Out: opts.stdoutWriter()},
		SrcClient:         srcClient,
		DstClient:         dstClient,
		SrcBridgeAddress:  srcBridgeAddr,
		DstBridgeAddress:  dstBridgeAddr,
		SrcSignalAddress:  srcSignalAddr,
		DstSignalAddress:  dstSignalAddr,
		SrcERC20Address:   srcERC20Addr,
		SrcERC721Address:  srcERC721Addr,
		SrcERC1155Address: srcERC1155Addr,
		SrcBridge:         srcBridge,
		DstBridge:         dstBridge,
		SrcERC20Vault:     srcERC20,
		SrcERC721Vault:    srcERC721,
		SrcERC1155Vault:   srcERC1155,
		SrcSignalService:  srcSignal,
		DstSignalService:  dstSignal,
		PrivateKeyHex:     pk,
	}, nil
}

// close tears down RPC client connections.
func (r *runtime) close() {
	if r.SrcClient != nil {
		r.SrcClient.Close()
	}
	if r.DstClient != nil {
		r.DstClient.Close()
	}
}

// parseAddress validates and parses an EVM address string.
func parseAddress(v string, name string) (common.Address, error) {
	v = strings.TrimSpace(v)
	if !common.IsHexAddress(v) {
		return common.Address{}, fmt.Errorf("invalid %s", name)
	}
	return common.HexToAddress(v), nil
}

// resolveAddressWithPreset resolves an address from explicit flag override or fixed chain preset.
func resolveAddressWithPreset(
	override string,
	flagName string,
	preset common.Address,
	hasPreset bool,
	chainID uint64,
) (common.Address, error) {
	if strings.TrimSpace(override) != "" {
		return parseAddress(override, flagName)
	}
	if hasPreset {
		return preset, nil
	}
	return common.Address{}, fmt.Errorf("%s is required for unsupported chain id %d", flagName, chainID)
}

// parseBig parses decimal or 0x-prefixed hexadecimal integer values.
func parseBig(v string, name string) (*big.Int, error) {
	if strings.TrimSpace(v) == "" {
		return nil, fmt.Errorf("%s required", name)
	}
	if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
		out, ok := new(big.Int).SetString(v[2:], 16)
		if !ok {
			return nil, fmt.Errorf("invalid %s", name)
		}
		return out, nil
	}
	out, ok := new(big.Int).SetString(v, 10)
	if !ok {
		return nil, fmt.Errorf("invalid %s", name)
	}
	return out, nil
}

// parseOptionalBig parses a bigint value and allows empty input.
func parseOptionalBig(v string) (*big.Int, error) {
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	return parseBig(v, "value")
}

// parseBytesHex parses optional 0x-prefixed byte input.
func parseBytesHex(v string) ([]byte, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	if !strings.HasPrefix(v, "0x") {
		return nil, fmt.Errorf("data must be 0x-prefixed hex")
	}
	return common.FromHex(v), nil
}

// parseCSVBigInts parses comma-separated big integer values.
func parseCSVBigInts(v string, field string) ([]*big.Int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, fmt.Errorf("%s required", field)
	}
	parts := strings.Split(v, ",")
	out := make([]*big.Int, 0, len(parts))
	for _, p := range parts {
		b, err := parseBig(strings.TrimSpace(p), field)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// parseCSVBigIntsOptional parses comma-separated big integer values and allows empty input.
func parseCSVBigIntsOptional(v string) ([]*big.Int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	parts := strings.Split(v, ",")
	out := make([]*big.Int, 0, len(parts))
	for _, p := range parts {
		b, err := parseBig(strings.TrimSpace(p), "amounts")
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// ensureDestOwner returns explicit dest owner, defaulting to recipient address.
func ensureDestOwner(to common.Address, destOwner string) (common.Address, error) {
	if strings.TrimSpace(destOwner) == "" {
		return to, nil
	}
	return parseAddress(destOwner, "dest-owner")
}

// chainIDUint64 reads and converts chain id from RPC client.
func chainIDUint64(ctx context.Context, client *ethclient.Client) (uint64, error) {
	id, err := client.ChainID(ctx)
	if err != nil {
		return 0, err
	}
	return id.Uint64(), nil
}

// requireFlag enforces non-empty root flag values.
func requireFlag(v string, name string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return v, nil
}
