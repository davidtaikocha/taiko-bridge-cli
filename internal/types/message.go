package types

import (
	"encoding/hex"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	"github.com/ethereum/go-ethereum/common"
)

// MessageSent captures a Bridge.MessageSent event with source metadata.
type MessageSent struct {
	// Message is the raw bridge message payload.
	Message bridgebinding.IBridgeMessage
	// MsgHash is the bridge-computed message hash.
	MsgHash [32]byte
	// SourceBridge is the source bridge contract that emitted the event.
	SourceBridge common.Address
	// SourceTxHash is the source transaction hash containing MessageSent.
	SourceTxHash common.Hash
	// SourceBlock is the source chain block number of the event log.
	SourceBlock uint64
	// SourceLogIdx is the log index of the MessageSent event in the receipt.
	SourceLogIdx uint
}

// MsgHashHex returns the message hash in 0x-prefixed hex form.
func (m MessageSent) MsgHashHex() string {
	return "0x" + hex.EncodeToString(m.MsgHash[:])
}
