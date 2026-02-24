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
	ErrNotReady = errors.New("message not ready")
	ErrReverted = errors.New("claim transaction reverted")
)

type bridgeContract interface {
	ProcessMessage(opts *bind.TransactOpts, _message bridgebinding.IBridgeMessage, _proof []byte) (*types.Transaction, error)
	IsMessageReceived(opts *bind.CallOpts, _message bridgebinding.IBridgeMessage, _proof []byte) (bool, error)
}

type receiptClient interface {
	ChainID(ctx context.Context) (*big.Int, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type Request struct {
	Message bridgebinding.IBridgeMessage
	Proof   []byte
	Value   *big.Int
}

type Result struct {
	TxHash   common.Hash `json:"tx_hash"`
	Ready    bool        `json:"ready"`
	Claimed  bool        `json:"claimed"`
	Reverted bool        `json:"reverted"`
}

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

func IsNotReadyError(err error) bool {
	if err == nil {
		return false
	}
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "b_signal_not_received") || strings.Contains(e, "message not received") || strings.Contains(e, "not ready")
}
