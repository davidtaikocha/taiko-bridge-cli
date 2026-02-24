package proof

import (
	"context"
	"fmt"
	"math/big"

	signalservicebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/v4/signalservice"
	bridgetypes "github.com/davidcai/taiko-bridge-cli/internal/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type rpcCaller interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type blockReader interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

type BuildRequest struct {
	RPC                rpcCaller
	BlockReader        blockReader
	SignalService      *signalservicebinding.SignalService
	SignalServiceAddr  common.Address
	HopChainID         uint64
	Message            bridgetypes.MessageSent
	CheckpointBlockNum uint64
}

type HopProof struct {
	ChainID      uint64   `abi:"chainId"`
	BlockID      uint64   `abi:"blockId"`
	RootHash     [32]byte `abi:"rootHash"`
	CacheOption  uint8    `abi:"cacheOption"`
	AccountProof [][]byte `abi:"accountProof"`
	StorageProof [][]byte `abi:"storageProof"`
}

var (
	hopProofsType abi.Type
)

func init() {
	var err error
	hopProofsType, err = abi.NewType("tuple[]", "tuple[]", []abi.ArgumentMarshaling{
		{Name: "chainId", Type: "uint64"},
		{Name: "blockId", Type: "uint64"},
		{Name: "rootHash", Type: "bytes32"},
		{Name: "cacheOption", Type: "uint8"},
		{Name: "accountProof", Type: "bytes[]"},
		{Name: "storageProof", Type: "bytes[]"},
	})
	if err != nil {
		panic(err)
	}
}

func Build(ctx context.Context, req BuildRequest) ([]byte, error) {
	if req.RPC == nil || req.BlockReader == nil || req.SignalService == nil {
		return nil, fmt.Errorf("missing proof dependencies")
	}

	key, err := req.SignalService.GetSignalSlot(
		&bind.CallOpts{Context: ctx},
		req.Message.Message.SrcChainId,
		req.Message.SourceBridge,
		req.Message.MsgHash,
	)
	if err != nil {
		return nil, fmt.Errorf("get signal slot: %w", err)
	}

	block, err := req.BlockReader.BlockByNumber(ctx, new(big.Int).SetUint64(req.CheckpointBlockNum))
	if err != nil {
		return nil, fmt.Errorf("block by number: %w", err)
	}

	accountProof, storageProof, err := getProof(
		ctx,
		req.RPC,
		req.SignalServiceAddr,
		hexutil.Encode(key[:]),
		req.CheckpointBlockNum,
	)
	if err != nil {
		return nil, err
	}

	proof := []HopProof{{
		ChainID:      req.HopChainID,
		BlockID:      req.CheckpointBlockNum,
		RootHash:     block.Root(),
		CacheOption:  0,
		AccountProof: accountProof,
		StorageProof: storageProof,
	}}

	args := abi.Arguments{{Type: hopProofsType}}
	encoded, err := args.Pack(proof)
	if err != nil {
		return nil, fmt.Errorf("encode hop proof: %w", err)
	}
	return encoded, nil
}

type getProofResponse struct {
	AccountProof []string `json:"accountProof"`
	StorageProof []struct {
		Value string   `json:"value"`
		Proof []string `json:"proof"`
	} `json:"storageProof"`
}

func getProof(
	ctx context.Context,
	caller rpcCaller,
	signalService common.Address,
	slot string,
	blockNumber uint64,
) ([][]byte, [][]byte, error) {
	var resp getProofResponse
	if err := caller.CallContext(
		ctx,
		&resp,
		"eth_getProof",
		signalService,
		[]string{slot},
		hexutil.EncodeUint64(blockNumber),
	); err != nil {
		return nil, nil, fmt.Errorf("eth_getProof: %w", err)
	}

	if len(resp.StorageProof) == 0 {
		return nil, nil, fmt.Errorf("eth_getProof returned empty storageProof")
	}
	value, ok := new(big.Int).SetString(trim0x(resp.StorageProof[0].Value), 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid storage proof value")
	}
	if value.Sign() == 0 {
		return nil, nil, fmt.Errorf("storage proof value is zero")
	}

	accountProof, err := decodeHexArray(resp.AccountProof)
	if err != nil {
		return nil, nil, fmt.Errorf("decode account proof: %w", err)
	}
	storageProof, err := decodeHexArray(resp.StorageProof[0].Proof)
	if err != nil {
		return nil, nil, fmt.Errorf("decode storage proof: %w", err)
	}

	return accountProof, storageProof, nil
}

func decodeHexArray(in []string) ([][]byte, error) {
	out := make([][]byte, 0, len(in))
	for _, v := range in {
		b, err := hexutil.Decode(v)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func trim0x(v string) string {
	if len(v) >= 2 && v[:2] == "0x" {
		return v[2:]
	}
	return v
}
