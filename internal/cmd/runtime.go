package cmd

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	erc1155binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc1155vault"
	erc20binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc20vault"
	erc721binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc721vault"
	signalservicebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/v4/signalservice"
	"github.com/davidcai/taiko-bridge-cli/internal/config"
	"github.com/davidcai/taiko-bridge-cli/internal/outfmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type runtime struct {
	Profile *config.Profile
	Printer outfmt.Printer

	SrcClient *ethclient.Client
	DstClient *ethclient.Client

	SrcBridge *bridgebinding.Bridge
	DstBridge *bridgebinding.Bridge

	SrcERC20Vault   *erc20binding.ERC20Vault
	SrcERC721Vault  *erc721binding.ERC721Vault
	SrcERC1155Vault *erc1155binding.ERC1155Vault

	DstSignalService *signalservicebinding.SignalService
	SrcSignalService *signalservicebinding.SignalService

	PrivateKeyHex string
}

func loadRuntime(ctx context.Context, opts *rootOptions) (*runtime, error) {
	profile, err := config.LoadProfile(opts.ConfigPath, opts.Profile)
	if err != nil {
		return nil, err
	}

	srcClient, err := ethclient.DialContext(ctx, profile.Src.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial src rpc: %w", err)
	}
	dstClient, err := ethclient.DialContext(ctx, profile.Dst.RPCURL)
	if err != nil {
		srcClient.Close()
		return nil, fmt.Errorf("dial dst rpc: %w", err)
	}

	srcBridge, err := bridgebinding.NewBridge(profile.Src.BridgeAddress, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src bridge binding: %w", err)
	}
	dstBridge, err := bridgebinding.NewBridge(profile.Dst.BridgeAddress, dstClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new dst bridge binding: %w", err)
	}

	srcERC20, err := erc20binding.NewERC20Vault(profile.Src.ERC20VaultAddress, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src erc20 vault binding: %w", err)
	}
	srcERC721, err := erc721binding.NewERC721Vault(profile.Src.ERC721VaultAddress, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src erc721 vault binding: %w", err)
	}
	srcERC1155, err := erc1155binding.NewERC1155Vault(profile.Src.ERC1155VaultAddr, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src erc1155 vault binding: %w", err)
	}

	srcSignal, err := signalservicebinding.NewSignalService(profile.Src.SignalService, srcClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new src signalservice binding: %w", err)
	}
	dstSignal, err := signalservicebinding.NewSignalService(profile.Dst.SignalService, dstClient)
	if err != nil {
		srcClient.Close()
		dstClient.Close()
		return nil, fmt.Errorf("new dst signalservice binding: %w", err)
	}

	pk := strings.TrimSpace(opts.PrivateKey)
	if pk == "" && strings.TrimSpace(opts.PrivateKeyEnv) != "" {
		pk = strings.TrimSpace(os.Getenv(strings.TrimSpace(opts.PrivateKeyEnv)))
	}

	return &runtime{
		Profile:          profile,
		Printer:          outfmt.Printer{Format: opts.Format, Out: os.Stdout},
		SrcClient:        srcClient,
		DstClient:        dstClient,
		SrcBridge:        srcBridge,
		DstBridge:        dstBridge,
		SrcERC20Vault:    srcERC20,
		SrcERC721Vault:   srcERC721,
		SrcERC1155Vault:  srcERC1155,
		SrcSignalService: srcSignal,
		DstSignalService: dstSignal,
		PrivateKeyHex:    pk,
	}, nil
}

func (r *runtime) close() {
	if r.SrcClient != nil {
		r.SrcClient.Close()
	}
	if r.DstClient != nil {
		r.DstClient.Close()
	}
}

func parseAddress(v string, name string) (common.Address, error) {
	if !common.IsHexAddress(v) {
		return common.Address{}, fmt.Errorf("invalid %s", name)
	}
	return common.HexToAddress(v), nil
}

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

func parseOptionalBig(v string) (*big.Int, error) {
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	return parseBig(v, "value")
}

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

func ensureDestOwner(to common.Address, destOwner string) (common.Address, error) {
	if strings.TrimSpace(destOwner) == "" {
		return to, nil
	}
	return parseAddress(destOwner, "dest-owner")
}

func chainIDUint64(ctx context.Context, client *ethclient.Client) (uint64, error) {
	id, err := client.ChainID(ctx)
	if err != nil {
		return 0, err
	}
	return id.Uint64(), nil
}
