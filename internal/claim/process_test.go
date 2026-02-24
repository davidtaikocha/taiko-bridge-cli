package claim

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// fakeClient is a deterministic claim receipt client for tests.
type fakeClient struct {
	// chainID is returned by ChainID.
	chainID *big.Int
	// receipt is returned by TransactionReceipt.
	receipt *types.Receipt
}

// ChainID returns the configured chain id.
func (f *fakeClient) ChainID(ctx context.Context) (*big.Int, error) {
	return f.chainID, nil
}

// TransactionReceipt returns the configured receipt.
func (f *fakeClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return f.receipt, nil
}

// fakeBridge is a deterministic destination bridge for claim tests.
type fakeBridge struct {
	// ready controls IsMessageReceived result.
	ready bool
	// tx is returned by ProcessMessage.
	tx *types.Transaction
}

// IsMessageReceived returns configured readiness.
func (f *fakeBridge) IsMessageReceived(opts *bind.CallOpts, _message bridgebinding.IBridgeMessage, _proof []byte) (bool, error) {
	return f.ready, nil
}

// ProcessMessage returns configured transaction.
func (f *fakeBridge) ProcessMessage(opts *bind.TransactOpts, _message bridgebinding.IBridgeMessage, _proof []byte) (*types.Transaction, error) {
	return f.tx, nil
}

// TestProcess_HappyPath verifies successful claim lifecycle.
func TestProcess_HappyPath(t *testing.T) {
	key := mustKey(t)
	tx := types.NewTx(&types.LegacyTx{Nonce: 1, To: &common.Address{}, Gas: 21000, GasPrice: big.NewInt(1)})
	b := &fakeBridge{ready: true, tx: tx}
	c := &fakeClient{chainID: big.NewInt(1), receipt: &types.Receipt{Status: types.ReceiptStatusSuccessful}}

	res, err := Process(context.Background(), c, b, key, Request{Message: bridgebinding.IBridgeMessage{}, Proof: []byte{0x01}})
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	if !res.Claimed || res.TxHash != tx.Hash() {
		t.Fatalf("unexpected result: %+v", res)
	}
}

// TestProcess_Revert verifies revert classification on failed receipt status.
func TestProcess_Revert(t *testing.T) {
	key := mustKey(t)
	tx := types.NewTx(&types.LegacyTx{Nonce: 1, To: &common.Address{}, Gas: 21000, GasPrice: big.NewInt(1)})
	b := &fakeBridge{ready: true, tx: tx}
	c := &fakeClient{chainID: big.NewInt(1), receipt: &types.Receipt{Status: types.ReceiptStatusFailed}}

	_, err := Process(context.Background(), c, b, key, Request{Message: bridgebinding.IBridgeMessage{}, Proof: []byte{0x01}})
	if err != ErrReverted {
		t.Fatalf("expected ErrReverted, got %v", err)
	}
}

// TestProcess_NotReady verifies early return when IsMessageReceived is false.
func TestProcess_NotReady(t *testing.T) {
	key := mustKey(t)
	b := &fakeBridge{ready: false}
	c := &fakeClient{chainID: big.NewInt(1), receipt: &types.Receipt{Status: types.ReceiptStatusSuccessful}}

	_, err := Process(context.Background(), c, b, key, Request{Message: bridgebinding.IBridgeMessage{}, Proof: []byte{0x01}})
	if err != ErrNotReady {
		t.Fatalf("expected ErrNotReady, got %v", err)
	}
}

// mustKey generates a throwaway private key for tests.
func mustKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}
