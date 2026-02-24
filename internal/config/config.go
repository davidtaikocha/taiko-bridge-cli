package config

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/yaml.v3"
)

// EndpointYAML is the YAML representation of one bridge endpoint.
type EndpointYAML struct {
	// RPCURL is the endpoint RPC URL.
	RPCURL string `yaml:"rpc_url"`
	// Bridge is the Bridge contract address.
	Bridge string `yaml:"bridge"`
	// SignalService is the SignalService contract address.
	SignalService string `yaml:"signal_service"`
	// ERC20Vault is the ERC20Vault contract address.
	ERC20Vault string `yaml:"erc20_vault"`
	// ERC721Vault is the ERC721Vault contract address.
	ERC721Vault string `yaml:"erc721_vault"`
	// ERC1155Vault is the ERC1155Vault contract address.
	ERC1155Vault string `yaml:"erc1155_vault"`
}

// ProfileYAML represents source and destination endpoints in YAML config.
type ProfileYAML struct {
	// Src is the source endpoint config.
	Src EndpointYAML `yaml:"src"`
	// Dst is the destination endpoint config.
	Dst EndpointYAML `yaml:"dest"`
}

// FileYAML is the root YAML config model.
type FileYAML struct {
	// Profiles maps profile names to profile definitions.
	Profiles map[string]ProfileYAML `yaml:"profiles"`
}

// Endpoint is the parsed endpoint configuration with typed addresses.
type Endpoint struct {
	// RPCURL is the endpoint RPC URL.
	RPCURL string
	// BridgeAddress is the parsed Bridge contract address.
	BridgeAddress common.Address
	// SignalService is the parsed SignalService contract address.
	SignalService common.Address
	// ERC20VaultAddress is the parsed ERC20Vault contract address.
	ERC20VaultAddress common.Address
	// ERC721VaultAddress is the parsed ERC721Vault contract address.
	ERC721VaultAddress common.Address
	// ERC1155VaultAddr is the parsed ERC1155Vault contract address.
	ERC1155VaultAddr common.Address
}

// Profile is the parsed source/destination configuration.
type Profile struct {
	// Name is the selected profile name.
	Name string
	// Src is the parsed source endpoint.
	Src Endpoint
	// Dst is the parsed destination endpoint.
	Dst Endpoint
}

// LoadProfile reads YAML config and parses one named profile.
func LoadProfile(path string, profileName string) (*Profile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg FileYAML
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	p, ok := cfg.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", profileName)
	}

	src, err := parseEndpoint(p.Src)
	if err != nil {
		return nil, fmt.Errorf("parse src endpoint: %w", err)
	}

	dst, err := parseEndpoint(p.Dst)
	if err != nil {
		return nil, fmt.Errorf("parse dest endpoint: %w", err)
	}

	return &Profile{Name: profileName, Src: src, Dst: dst}, nil
}

// parseEndpoint validates a YAML endpoint and converts it into typed fields.
func parseEndpoint(in EndpointYAML) (Endpoint, error) {
	if strings.TrimSpace(in.RPCURL) == "" {
		return Endpoint{}, fmt.Errorf("rpc_url is required")
	}

	out := Endpoint{RPCURL: strings.TrimSpace(in.RPCURL)}

	var err error
	if out.BridgeAddress, err = parseAddress(in.Bridge, "bridge"); err != nil {
		return Endpoint{}, err
	}
	if out.SignalService, err = parseAddress(in.SignalService, "signal_service"); err != nil {
		return Endpoint{}, err
	}
	if out.ERC20VaultAddress, err = parseAddress(in.ERC20Vault, "erc20_vault"); err != nil {
		return Endpoint{}, err
	}
	if out.ERC721VaultAddress, err = parseAddress(in.ERC721Vault, "erc721_vault"); err != nil {
		return Endpoint{}, err
	}
	if out.ERC1155VaultAddr, err = parseAddress(in.ERC1155Vault, "erc1155_vault"); err != nil {
		return Endpoint{}, err
	}

	return out, nil
}

// parseAddress parses and validates a hex address field.
func parseAddress(v string, field string) (common.Address, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return common.Address{}, fmt.Errorf("%s is required", field)
	}
	if !common.IsHexAddress(v) {
		return common.Address{}, fmt.Errorf("%s must be a valid address", field)
	}
	return common.HexToAddress(v), nil
}

// LoadPrivateKey resolves private key from flag/env and parses ECDSA key.
func LoadPrivateKey(privateKeyFlag string, privateKeyEnv string) (*ecdsa.PrivateKey, error) {
	keyHex := strings.TrimSpace(privateKeyFlag)
	if keyHex == "" && strings.TrimSpace(privateKeyEnv) != "" {
		keyHex = strings.TrimSpace(os.Getenv(strings.TrimSpace(privateKeyEnv)))
	}
	if keyHex == "" {
		return nil, fmt.Errorf("private key required via --private-key or --private-key-env")
	}
	keyHex = strings.TrimPrefix(keyHex, "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return key, nil
}
