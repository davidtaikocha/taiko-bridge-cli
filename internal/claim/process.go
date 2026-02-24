package claim

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// ErrNotReady indicates message proof is not yet processable on destination bridge.
	ErrNotReady = errors.New("message not ready")
	// ErrReverted indicates processMessage transaction reverted on-chain.
	ErrReverted = errors.New("claim transaction reverted")
)

// bridgeContract is the destination bridge interface used by the claimer.
type bridgeContract interface {
	// ProcessMessage executes the claim on destination chain.
	ProcessMessage(opts *bind.TransactOpts, _message bridgebinding.IBridgeMessage, _proof []byte) (*types.Transaction, error)
	// IsMessageReceived checks if the message proof is currently valid.
	IsMessageReceived(opts *bind.CallOpts, _message bridgebinding.IBridgeMessage, _proof []byte) (bool, error)
}

// receiptClient is the chain client surface needed by claim processing.
type receiptClient interface {
	// ChainID returns the target chain id used for signing.
	ChainID(ctx context.Context) (*big.Int, error)
	// TransactionReceipt returns receipt for submitted claim transaction.
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// Request contains claim inputs for bridge.processMessage.
type Request struct {
	// Message is the decoded bridge message.
	Message bridgebinding.IBridgeMessage
	// Proof is the ABI-encoded hop proof payload.
	Proof []byte
	// Value is optional tx value to send with processMessage.
	Value *big.Int
}

// Result describes claim execution status.
type Result struct {
	// TxHash is the submitted claim transaction hash.
	TxHash common.Hash `json:"tx_hash"`
	// Ready reflects the pre-flight readiness check result.
	Ready bool `json:"ready"`
	// Claimed indicates whether claim completed successfully.
	Claimed bool `json:"claimed"`
	// Reverted indicates whether claim tx reverted on-chain.
	Reverted bool `json:"reverted"`
}

// Process validates readiness, sends processMessage, and waits for tx confirmation.
func Process(
	ctx context.Context,
	client receiptClient,
	destBridge bridgeContract,
	privateKey *ecdsa.PrivateKey,
	req Request,
) (*Result, error) {
	ready, err := destBridge.IsMessageReceived(&bind.CallOpts{Context: ctx}, req.Message, req.Proof)
	if err != nil {
		return nil, fmt.Errorf("isMessageReceived: %w", err)
	}
	if !ready {
		return &Result{Ready: false, Claimed: false}, ErrNotReady
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("new transactor: %w", err)
	}
	auth.Context = ctx
	if req.Value != nil {
		auth.Value = new(big.Int).Set(req.Value)
	}

	tx, err := destBridge.ProcessMessage(auth, req.Message, req.Proof)
	if err != nil {
		if IsNotReadyError(err) {
			return &Result{Ready: false, Claimed: false}, ErrNotReady
		}
		return nil, fmt.Errorf("processMessage: %w", err)
	}

	receipt, err := waitReceipt(ctx, client, tx.Hash())
	if err != nil {
		if errors.Is(err, ErrReverted) {
			return &Result{TxHash: tx.Hash(), Ready: true, Claimed: false, Reverted: true}, err
		}
		return nil, err
	}

	_ = receipt
	return &Result{TxHash: tx.Hash(), Ready: true, Claimed: true}, nil
}

// waitReceipt polls transaction receipt until mined, reverted, or context timeout.
func waitReceipt(ctx context.Context, client receiptClient, txHash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, ErrReverted
			}
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// IsNotReadyError classifies on-chain errors that indicate claim is not ready yet.
func IsNotReadyError(err error) bool {
	if err == nil {
		return false
	}
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "b_signal_not_received") || strings.Contains(e, "message not received") || strings.Contains(e, "not ready")
}
