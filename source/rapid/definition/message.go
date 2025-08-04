package definition

import "google.golang.org/protobuf/reflect/protoreflect"

type ProtoMessage interface {
	protoreflect.ProtoMessage
}

// Whenever you add the type of message to send,
// Please enter the message type here
const (
	MessageTypeNil = iota

	Part
	Block
	BlockHeader

	TendermintProposal
	TendermintPrevote
	TendermintPrecommit
	TendermintAggregatedPrecommit

	MessageAccept
	MessageCommit
	MessageOK
	CrossShardBlock

	TxTransfer
	TxInsert
	TxBlock
	TxRelay
)
