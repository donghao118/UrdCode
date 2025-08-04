package definition

import "google.golang.org/protobuf/reflect/protoreflect"

type ProtoMessage interface {
	protoreflect.ProtoMessage
}

// Every a new type that needs to be transfered to proto types should register its identifier here.
const (
	MessageTypeNil = iota

	Part
	Proposal
	CrossShardMessage
	Vote

	TxInsert
	TxTransfer
)
