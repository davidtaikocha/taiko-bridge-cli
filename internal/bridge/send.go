package bridge

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	erc1155binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc1155vault"
	erc20binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc20vault"
	erc721binding "github.com/davidcai/taiko-bridge-cli/internal/bindings/erc721vault"
	bridgetypes "github.com/davidcai/taiko-bridge-cli/internal/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrNoMessageSentEvent = errors.New("no MessageSent event found in receipt")
	ErrTxReverted         = errors.New("transaction reverted")
)

type receiptReader interface {
	ChainID(ctx context.Context) (*big.Int, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type SendETHRequest struct {
	From        common.Address
	DestChainID uint64
	DestOwner   common.Address
	To          common.Address
	Value       *big.Int
	Fee         *big.Int
	GasLimit    uint32
	Data        []byte
}

type SendERC20Request struct {
	DestChainID uint64
	DestOwner   common.Address
	To          common.Address
	Fee         *big.Int
	Token       common.Address
	GasLimit    uint32
	Amount      *big.Int
}

type SendNFTRequest struct {
	DestChainID uint64
	DestOwner   common.Address
	To          common.Address
	Fee         *big.Int
	Token       common.Address
	GasLimit    uint32
	TokenIDs    []*big.Int
	Amounts     []*big.Int
}

type SendResult struct {
	TxHash  common.Hash
	Receipt *types.Receipt
	Event   bridgetypes.MessageSent
}

func BuildETHMessage(req SendETHRequest, srcChainID uint64) (bridgebinding.IBridgeMessage, *big.Int, error) {
	if req.Value == nil || req.Value.Sign() < 0 {
		return bridgebinding.IBridgeMessage{}, nil, fmt.Errorf("value must be >= 0")
	}
	if req.Fee == nil || req.Fee.Sign() < 0 {
		return bridgebinding.IBridgeMessage{}, nil, fmt.Errorf("fee must be >= 0")
	}
	feeU64, err := toUint64(req.Fee, "fee")
	if err != nil {
		return bridgebinding.IBridgeMessage{}, nil, err
	}
	msgValue := new(big.Int).Add(new(big.Int).Set(req.Value), req.Fee)

	msg := bridgebinding.IBridgeMessage{
		Id:          0,
		Fee:         feeU64,
		GasLimit:    req.GasLimit,
		From:        req.From,
		SrcChainId:  srcChainID,
		SrcOwner:    req.From,
		DestChainId: req.DestChainID,
		DestOwner:   req.DestOwner,
		To:          req.To,
		Value:       new(big.Int).Set(req.Value),
		Data:        req.Data,
	}

	return msg, msgValue, nil
}

func BuildERC20Op(req SendERC20Request) (erc20binding.ERC20VaultBridgeTransferOp, *big.Int, error) {
	if req.Amount == nil || req.Amount.Sign() <= 0 {
		return erc20binding.ERC20VaultBridgeTransferOp{}, nil, fmt.Errorf("amount must be > 0")
	}
	if req.Fee == nil || req.Fee.Sign() < 0 {
		return erc20binding.ERC20VaultBridgeTransferOp{}, nil, fmt.Errorf("fee must be >= 0")
	}
	feeU64, err := toUint64(req.Fee, "fee")
	if err != nil {
		return erc20binding.ERC20VaultBridgeTransferOp{}, nil, err
	}
	op := erc20binding.ERC20VaultBridgeTransferOp{
		DestChainId: req.DestChainID,
		DestOwner:   req.DestOwner,
		To:          req.To,
		Fee:         feeU64,
		Token:       req.Token,
		GasLimit:    req.GasLimit,
		Amount:      new(big.Int).Set(req.Amount),
	}
	return op, new(big.Int).Set(req.Fee), nil
}

func BuildNFTAmounts(tokenIDs []*big.Int, amounts []*big.Int, is721 bool) ([]*big.Int, error) {
	if len(tokenIDs) == 0 {
		return nil, fmt.Errorf("token ids required")
	}
	if is721 {
		if len(amounts) == 0 {
			amounts = make([]*big.Int, len(tokenIDs))
			for i := range tokenIDs {
				amounts[i] = big.NewInt(1)
			}
		}
	}
	if len(tokenIDs) != len(amounts) {
		return nil, fmt.Errorf("token ids and amounts length mismatch")
	}
	out := make([]*big.Int, len(amounts))
	for i, amt := range amounts {
		if amt == nil || amt.Sign() <= 0 {
			return nil, fmt.Errorf("amount at index %d must be > 0", i)
		}
		out[i] = new(big.Int).Set(amt)
	}
	return out, nil
}

func SendETH(
	ctx context.Context,
	client receiptReader,
	srcBridge *bridgebinding.Bridge,
	sourceBridgeAddress common.Address,
	priv *ecdsa.PrivateKey,
	req SendETHRequest,
) (*SendResult, error) {
	srcChainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}

	msg, msgValue, err := BuildETHMessage(req, srcChainID.Uint64())
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(priv, srcChainID)
	if err != nil {
		return nil, fmt.Errorf("new transactor: %w", err)
	}
	auth.Context = ctx
	auth.Value = msgValue

	tx, err := srcBridge.SendMessage(auth, msg)
	if err != nil {
		return nil, fmt.Errorf("bridge.sendMessage: %w", err)
	}

	return waitAndExtract(ctx, client, srcBridge, sourceBridgeAddress, tx.Hash(), 0)
}

func SendERC20(
	ctx context.Context,
	client receiptReader,
	srcVault *erc20binding.ERC20Vault,
	srcBridge *bridgebinding.Bridge,
	sourceBridgeAddress common.Address,
	priv *ecdsa.PrivateKey,
	req SendERC20Request,
) (*SendResult, error) {
	srcChainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	op, msgValue, err := BuildERC20Op(req)
	if err != nil {
		return nil, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(priv, srcChainID)
	if err != nil {
		return nil, fmt.Errorf("new transactor: %w", err)
	}
	auth.Context = ctx
	auth.Value = msgValue
	tx, err := srcVault.SendToken(auth, op)
	if err != nil {
		return nil, fmt.Errorf("erc20Vault.sendToken: %w", err)
	}
	return waitAndExtract(ctx, client, srcBridge, sourceBridgeAddress, tx.Hash(), 0)
}

func SendERC721(
	ctx context.Context,
	client receiptReader,
	srcVault *erc721binding.ERC721Vault,
	srcBridge *bridgebinding.Bridge,
	sourceBridgeAddress common.Address,
	priv *ecdsa.PrivateKey,
	req SendNFTRequest,
) (*SendResult, error) {
	srcChainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	amounts, err := BuildNFTAmounts(req.TokenIDs, req.Amounts, true)
	if err != nil {
		return nil, err
	}
	feeU64, err := toUint64(req.Fee, "fee")
	if err != nil {
		return nil, err
	}
	op := erc721binding.BaseNFTVaultBridgeTransferOp{
		DestChainId: req.DestChainID,
		DestOwner:   req.DestOwner,
		To:          req.To,
		Fee:         feeU64,
		Token:       req.Token,
		GasLimit:    req.GasLimit,
		TokenIds:    cloneBigIntSlice(req.TokenIDs),
		Amounts:     amounts,
	}
	auth, err := bind.NewKeyedTransactorWithChainID(priv, srcChainID)
	if err != nil {
		return nil, fmt.Errorf("new transactor: %w", err)
	}
	auth.Context = ctx
	auth.Value = new(big.Int).Set(req.Fee)
	tx, err := srcVault.SendToken(auth, op)
	if err != nil {
		return nil, fmt.Errorf("erc721Vault.sendToken: %w", err)
	}
	return waitAndExtract(ctx, client, srcBridge, sourceBridgeAddress, tx.Hash(), 0)
}

func SendERC1155(
	ctx context.Context,
	client receiptReader,
	srcVault *erc1155binding.ERC1155Vault,
	srcBridge *bridgebinding.Bridge,
	sourceBridgeAddress common.Address,
	priv *ecdsa.PrivateKey,
	req SendNFTRequest,
) (*SendResult, error) {
	srcChainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	amounts, err := BuildNFTAmounts(req.TokenIDs, req.Amounts, false)
	if err != nil {
		return nil, err
	}
	feeU64, err := toUint64(req.Fee, "fee")
	if err != nil {
		return nil, err
	}
	op := erc1155binding.BaseNFTVaultBridgeTransferOp{
		DestChainId: req.DestChainID,
		DestOwner:   req.DestOwner,
		To:          req.To,
		Fee:         feeU64,
		Token:       req.Token,
		GasLimit:    req.GasLimit,
		TokenIds:    cloneBigIntSlice(req.TokenIDs),
		Amounts:     amounts,
	}
	auth, err := bind.NewKeyedTransactorWithChainID(priv, srcChainID)
	if err != nil {
		return nil, fmt.Errorf("new transactor: %w", err)
	}
	auth.Context = ctx
	auth.Value = new(big.Int).Set(req.Fee)
	tx, err := srcVault.SendToken(auth, op)
	if err != nil {
		return nil, fmt.Errorf("erc1155Vault.sendToken: %w", err)
	}
	return waitAndExtract(ctx, client, srcBridge, sourceBridgeAddress, tx.Hash(), 0)
}

func ReadMessageSentFromTx(
	ctx context.Context,
	client receiptReader,
	srcBridge *bridgebinding.Bridge,
	sourceBridgeAddress common.Address,
	txHash common.Hash,
	eventIndex int,
) (*bridgetypes.MessageSent, *types.Receipt, error) {
	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, nil, err
	}
	evt, err := parseMessageSentFromReceipt(receipt, srcBridge, sourceBridgeAddress, eventIndex)
	if err != nil {
		return nil, receipt, err
	}
	return evt, receipt, nil
}

func waitAndExtract(
	ctx context.Context,
	client receiptReader,
	srcBridge *bridgebinding.Bridge,
	sourceBridgeAddress common.Address,
	txHash common.Hash,
	eventIndex int,
) (*SendResult, error) {
	receipt, err := waitReceipt(ctx, client, txHash)
	if err != nil {
		return nil, err
	}
	evt, err := parseMessageSentFromReceipt(receipt, srcBridge, sourceBridgeAddress, eventIndex)
	if err != nil {
		return nil, err
	}
	return &SendResult{TxHash: txHash, Receipt: receipt, Event: *evt}, nil
}

func waitReceipt(ctx context.Context, client receiptReader, txHash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, fmt.Errorf("%w: %s", ErrTxReverted, txHash.Hex())
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

func parseMessageSentFromReceipt(
	receipt *types.Receipt,
	srcBridge *bridgebinding.Bridge,
	sourceBridgeAddress common.Address,
	eventIndex int,
) (*bridgetypes.MessageSent, error) {
	if eventIndex < 0 {
		return nil, fmt.Errorf("event index must be >= 0")
	}

	seen := 0
	for _, lg := range receipt.Logs {
		if lg.Address != sourceBridgeAddress {
			continue
		}
		evt, err := srcBridge.ParseMessageSent(*lg)
		if err != nil {
			continue
		}
		if seen == eventIndex {
			return &bridgetypes.MessageSent{
				Message:      evt.Message,
				MsgHash:      evt.MsgHash,
				SourceBridge: sourceBridgeAddress,
				SourceTxHash: receipt.TxHash,
				SourceBlock:  lg.BlockNumber,
				SourceLogIdx: lg.Index,
			}, nil
		}
		seen++
	}

	return nil, ErrNoMessageSentEvent
}

func toUint64(v *big.Int, field string) (uint64, error) {
	if v == nil || v.Sign() < 0 {
		return 0, fmt.Errorf("%s must be >= 0", field)
	}
	if !v.IsUint64() {
		return 0, fmt.Errorf("%s overflows uint64", field)
	}
	return v.Uint64(), nil
}

func cloneBigIntSlice(in []*big.Int) []*big.Int {
	out := make([]*big.Int, len(in))
	for i, v := range in {
		if v == nil {
			out[i] = big.NewInt(0)
			continue
		}
		out[i] = new(big.Int).Set(v)
	}
	return out
}
