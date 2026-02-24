package config

import "github.com/ethereum/go-ethereum/common"

// FixedAddresses is the canonical bridge contract address set for one chain id.
type FixedAddresses struct {
	// Bridge is the bridge proxy contract address.
	Bridge common.Address
	// SignalService is the signal service proxy contract address.
	SignalService common.Address
	// ERC20Vault is the ERC20 vault proxy contract address.
	ERC20Vault common.Address
	// ERC721Vault is the ERC721 vault proxy contract address.
	ERC721Vault common.Address
	// ERC1155Vault is the ERC1155 vault proxy contract address.
	ERC1155Vault common.Address
}

var (
	// fixedAddressByChainID contains canonical Taiko deployment addresses keyed by chain id.
	//
	// Source of truth:
	// - packages/protocol/deployments/taiko-hoodi-contract-logs.md
	// - packages/protocol/deployments/mainnet-contract-logs-L1.md
	// - packages/protocol/deployments/mainnet-contract-logs-L2.md
	fixedAddressByChainID = map[uint64]FixedAddresses{
		// Ethereum Mainnet L1.
		1: {
			Bridge:        common.HexToAddress("0xd60247c6848B7Ca29eDdF63AA924E53dB6Ddd8EC"),
			SignalService: common.HexToAddress("0x9e0a24964e5397B566c1ed39258e21aB5E35C77C"),
			ERC20Vault:    common.HexToAddress("0x996282cA11E5DEb6B5D122CC3B9A1FcAAD4415Ab"),
			ERC721Vault:   common.HexToAddress("0x0b470dd3A0e1C41228856Fb319649E7c08f419Aa"),
			ERC1155Vault:  common.HexToAddress("0xaf145913EA4a56BE22E120ED9C24589659881702"),
		},
		// Taiko Mainnet L2.
		167000: {
			Bridge:        common.HexToAddress("0x1670000000000000000000000000000000000001"),
			SignalService: common.HexToAddress("0x1670000000000000000000000000000000000005"),
			ERC20Vault:    common.HexToAddress("0x1670000000000000000000000000000000000002"),
			ERC721Vault:   common.HexToAddress("0x1670000000000000000000000000000000000003"),
			ERC1155Vault:  common.HexToAddress("0x1670000000000000000000000000000000000004"),
		},
		// Ethereum Hoodi L1.
		560048: {
			Bridge:        common.HexToAddress("0x6a4cf607DaC2C4784B7D934Bcb3AD7F2ED18Ed80"),
			SignalService: common.HexToAddress("0x4c70b7F5E153D497faFa0476575903F9299ed811"),
			ERC20Vault:    common.HexToAddress("0x0857cd029937E7a119e492434c71CB9a9Bb59aB0"),
			ERC721Vault:   common.HexToAddress("0x4876e7993dD40C22526c8B01F2D52AD8FdbdF768"),
			ERC1155Vault:  common.HexToAddress("0x81Ff6CcE1e5cFd6ebE83922F5A9608d1752C92c6"),
		},
		// Taiko Hoodi L2.
		167013: {
			Bridge:        common.HexToAddress("0x1670130000000000000000000000000000000001"),
			SignalService: common.HexToAddress("0x1670130000000000000000000000000000000005"),
			ERC20Vault:    common.HexToAddress("0x1670130000000000000000000000000000000002"),
			ERC721Vault:   common.HexToAddress("0x1670130000000000000000000000000000000003"),
			ERC1155Vault:  common.HexToAddress("0x1670130000000000000000000000000000000004"),
		},
	}
)

// LookupFixedAddresses returns canonical addresses for a known chain id.
func LookupFixedAddresses(chainID uint64) (FixedAddresses, bool) {
	addrs, ok := fixedAddressByChainID[chainID]
	return addrs, ok
}
