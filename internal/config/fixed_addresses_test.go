package config

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestLookupFixedAddresses_KnownChains verifies fixed address presets exist for supported chains.
func TestLookupFixedAddresses_KnownChains(t *testing.T) {
	knownChains := []uint64{
		1,      // mainnet L1
		167000, // mainnet L2
		560048, // hoodi L1
		167013, // hoodi L2
	}

	for _, chainID := range knownChains {
		addrs, ok := LookupFixedAddresses(chainID)
		if !ok {
			t.Fatalf("expected fixed addresses for chain %d", chainID)
		}
		if addrs.Bridge == (common.Address{}) {
			t.Fatalf("bridge address missing for chain %d", chainID)
		}
		if addrs.SignalService == (common.Address{}) {
			t.Fatalf("signal service address missing for chain %d", chainID)
		}
		if addrs.ERC20Vault == (common.Address{}) {
			t.Fatalf("erc20 vault address missing for chain %d", chainID)
		}
		if addrs.ERC721Vault == (common.Address{}) {
			t.Fatalf("erc721 vault address missing for chain %d", chainID)
		}
		if addrs.ERC1155Vault == (common.Address{}) {
			t.Fatalf("erc1155 vault address missing for chain %d", chainID)
		}
	}
}

// TestLookupFixedAddresses_UnknownChain verifies unknown chain ids do not resolve.
func TestLookupFixedAddresses_UnknownChain(t *testing.T) {
	if _, ok := LookupFixedAddresses(99999999); ok {
		t.Fatalf("did not expect fixed addresses for unknown chain")
	}
}
