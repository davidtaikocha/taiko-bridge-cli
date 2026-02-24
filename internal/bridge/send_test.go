package bridge

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// TestBuildETHMessage_BiDirectional verifies ETH message construction for both directions.
func TestBuildETHMessage_BiDirectional(t *testing.T) {
	cases := []struct {
		name      string
		srcChain  uint64
		destChain uint64
	}{
		{name: "l1_to_l2", srcChain: 1, destChain: 167000},
		{name: "l2_to_l1", srcChain: 167000, destChain: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			from := common.HexToAddress("0x00000000000000000000000000000000000000a1")
			to := common.HexToAddress("0x00000000000000000000000000000000000000b2")
			req := SendETHRequest{
				From:        from,
				DestChainID: tc.destChain,
				DestOwner:   to,
				To:          to,
				Value:       big.NewInt(42),
				Fee:         big.NewInt(7),
				GasLimit:    12345,
				Data:        []byte{0x01, 0x02},
			}

			msg, msgValue, err := BuildETHMessage(req, tc.srcChain)
			if err != nil {
				t.Fatalf("BuildETHMessage error: %v", err)
			}
			if msg.SrcChainId != tc.srcChain || msg.DestChainId != tc.destChain {
				t.Fatalf("unexpected chains: src=%d dest=%d", msg.SrcChainId, msg.DestChainId)
			}
			if msg.Fee != 7 {
				t.Fatalf("unexpected fee: %d", msg.Fee)
			}
			if msgValue.Cmp(big.NewInt(49)) != 0 {
				t.Fatalf("unexpected msg.value: %s", msgValue)
			}
		})
	}
}

// TestBuildERC20Op_ValidationAndPayload validates ERC20 op payload shape and input checks.
func TestBuildERC20Op_ValidationAndPayload(t *testing.T) {
	cases := []uint64{1, 167000}
	for _, chainID := range cases {
		req := SendERC20Request{
			DestChainID: chainID,
			DestOwner:   common.HexToAddress("0x00000000000000000000000000000000000000d1"),
			To:          common.HexToAddress("0x00000000000000000000000000000000000000d2"),
			Fee:         big.NewInt(1000),
			Token:       common.HexToAddress("0x00000000000000000000000000000000000000e1"),
			GasLimit:    300000,
			Amount:      big.NewInt(999),
		}

		op, msgValue, err := BuildERC20Op(req)
		if err != nil {
			t.Fatalf("BuildERC20Op error: %v", err)
		}
		if op.DestChainId != req.DestChainID || op.Amount.Cmp(req.Amount) != 0 || op.Fee != 1000 {
			t.Fatalf("unexpected erc20 op payload: %+v", op)
		}
		if msgValue.Cmp(big.NewInt(1000)) != 0 {
			t.Fatalf("unexpected msg.value: %s", msgValue)
		}
	}

	req := SendERC20Request{
		DestChainID: 167000,
		DestOwner:   common.HexToAddress("0x00000000000000000000000000000000000000d1"),
		To:          common.HexToAddress("0x00000000000000000000000000000000000000d2"),
		Fee:         big.NewInt(1000),
		Token:       common.HexToAddress("0x00000000000000000000000000000000000000e1"),
		GasLimit:    300000,
		Amount:      big.NewInt(0),
	}
	if _, _, err := BuildERC20Op(req); err == nil {
		t.Fatalf("expected validation error for zero amount")
	}
}

// TestBuildNFTAmounts validates ERC721 defaults and mismatch checks.
func TestBuildNFTAmounts(t *testing.T) {
	ids := []*big.Int{big.NewInt(1), big.NewInt(2)}
	amts, err := BuildNFTAmounts(ids, nil, true)
	if err != nil {
		t.Fatalf("BuildNFTAmounts erc721 error: %v", err)
	}
	if amts[0].Cmp(big.NewInt(1)) != 0 || amts[1].Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("erc721 default amounts mismatch: %+v", amts)
	}

	_, err = BuildNFTAmounts(ids, []*big.Int{big.NewInt(1)}, false)
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
}

// fakeMinGasBridge mocks getMessageMinGasLimit queries for gas limit resolution tests.
type fakeMinGasBridge struct {
	// MinGasLimit is the mocked minimum gas limit result.
	MinGasLimit uint32
	// Err is the optional mocked call error.
	Err error
	// LastDataLen records the last data length argument.
	LastDataLen uint64
}

// GetMessageMinGasLimit implements the bridge min gas query for tests.
func (f *fakeMinGasBridge) GetMessageMinGasLimit(_ *bind.CallOpts, dataLength *big.Int) (uint32, error) {
	f.LastDataLen = dataLength.Uint64()
	if f.Err != nil {
		return 0, f.Err
	}
	return f.MinGasLimit, nil
}

// TestResolveETHGasLimit_AdjustsWhenRequestedBelowMin verifies automatic gas bumping.
func TestResolveETHGasLimit_AdjustsWhenRequestedBelowMin(t *testing.T) {
	mockBridge := &fakeMinGasBridge{MinGasLimit: 806656}
	res, err := ResolveETHGasLimit(context.Background(), mockBridge, []byte{0x01, 0x02}, 200000)
	if err != nil {
		t.Fatalf("ResolveETHGasLimit error: %v", err)
	}
	if res.MinGasLimit != 806656 {
		t.Fatalf("unexpected min gas limit: %d", res.MinGasLimit)
	}
	if res.EffectiveGasLimit != 806656 {
		t.Fatalf("unexpected effective gas limit: %d", res.EffectiveGasLimit)
	}
	if !res.Adjusted {
		t.Fatalf("expected gas limit to be adjusted")
	}
	if mockBridge.LastDataLen != 2 {
		t.Fatalf("unexpected data length: %d", mockBridge.LastDataLen)
	}
}

// TestResolveETHGasLimit_UsesRequestedWhenAtOrAboveMin verifies no adjustment is made when safe.
func TestResolveETHGasLimit_UsesRequestedWhenAtOrAboveMin(t *testing.T) {
	mockBridge := &fakeMinGasBridge{MinGasLimit: 806656}
	res, err := ResolveETHGasLimit(context.Background(), mockBridge, nil, 1000000)
	if err != nil {
		t.Fatalf("ResolveETHGasLimit error: %v", err)
	}
	if res.MinGasLimit != 806656 {
		t.Fatalf("unexpected min gas limit: %d", res.MinGasLimit)
	}
	if res.EffectiveGasLimit != 1000000 {
		t.Fatalf("unexpected effective gas limit: %d", res.EffectiveGasLimit)
	}
	if res.Adjusted {
		t.Fatalf("did not expect gas limit adjustment")
	}
}

// TestResolveETHGasLimit_Error verifies contract query failures are returned.
func TestResolveETHGasLimit_Error(t *testing.T) {
	mockBridge := &fakeMinGasBridge{Err: errors.New("rpc failed")}
	if _, err := ResolveETHGasLimit(context.Background(), mockBridge, nil, 1000000); err == nil {
		t.Fatalf("expected query error")
	}
}

// TestResolveETHGasLimit_NilBridge verifies nil bridge dependency is rejected.
func TestResolveETHGasLimit_NilBridge(t *testing.T) {
	if _, err := ResolveETHGasLimit(context.Background(), nil, nil, 1000000); err == nil {
		t.Fatalf("expected nil bridge error")
	}
}
