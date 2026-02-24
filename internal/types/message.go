package types

import (
	"encoding/hex"

	bridgebinding "github.com/davidcai/taiko-bridge-cli/internal/bindings/bridge"
	"github.com/ethereum/go-ethereum/common"
)

// MessageSent captures a Bridge.MessageSent event with source metadata.
type MessageSent struct {
	Message      bridgebinding.IBridgeMessage
	MsgHash      [32]byte
	SourceBridge common.Address
	SourceTxHash common.Hash
	SourceBlock  uint64
	SourceLogIdx uint
}

func (m MessageSent) MsgHashHex() string {
	return "0x" + hex.EncodeToString(m.MsgHash[:])
}
