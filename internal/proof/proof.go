package proof

import (
	"context"
	"fmt"
	"math/big"

	signalservicebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/signalservice"
	bridgetypes "github.com/davidcai/taiko-bridge-cli/internal/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// rpcCaller defines the minimal RPC caller interface needed for eth_getProof.
type rpcCaller interface {
	// CallContext performs a JSON-RPC call with context.
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

// blockReader defines block access needed to get source state root.
type blockReader interface {
	// BlockByNumber returns block data for the specified block number.
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

// BuildRequest contains all dependencies and metadata required to build a proof.
type BuildRequest struct {
	// RPC is the source chain JSON-RPC caller.
	RPC rpcCaller
	// BlockReader is used to fetch source block root hash.
	BlockReader blockReader
	// SignalService is the source signal service binding.
	SignalService *signalservicebinding.SignalService
	// SignalServiceAddr is the source signal service contract address.
	SignalServiceAddr common.Address
	// HopChainID is the destination chain id encoded in hop proof.
	HopChainID uint64
	// Message is the source message event metadata.
	Message bridgetypes.MessageSent
	// CheckpointBlockNum is the checkpoint block to prove against.
	CheckpointBlockNum uint64
}

// HopProof is the ABI model for one-hop proof tuple.
type HopProof struct {
	// ChainID is the destination chain id.
	ChainID uint64 `abi:"chainId"`
	// BlockID is the block number used for proof.
	BlockID uint64 `abi:"blockId"`
	// RootHash is the source block state root.
	RootHash [32]byte `abi:"rootHash"`
	// CacheOption controls bridge cache behavior.
	CacheOption uint8 `abi:"cacheOption"`
	// AccountProof is eth_getProof account proof nodes.
	AccountProof [][]byte `abi:"accountProof"`
	// StorageProof is eth_getProof storage proof nodes.
	StorageProof [][]byte `abi:"storageProof"`
}

var (
	// hopProofsType is the ABI tuple[] type used to pack hop proof arrays.
	hopProofsType abi.Type
)

// init pre-builds ABI type descriptors used during proof encoding.
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

// Build generates and ABI-encodes a single-hop signal proof for processMessage.
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

// getProofResponse is the partial eth_getProof response model used by this tool.
type getProofResponse struct {
	// AccountProof contains account proof node hex strings.
	AccountProof []string `json:"accountProof"`
	// StorageProof contains one storage proof entry for the signal slot.
	StorageProof []struct {
		// Value is the proved slot value.
		Value string `json:"value"`
		// Proof is the storage proof node list.
		Proof []string `json:"proof"`
	} `json:"storageProof"`
}

// getProof fetches account/storage proof for signal slot at a given block.
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

// decodeHexArray decodes a list of 0x-prefixed hex strings into byte slices.
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

// trim0x removes a leading 0x prefix when present.
func trim0x(v string) string {
	if len(v) >= 2 && v[:2] == "0x" {
		return v[2:]
	}
	return v
}
