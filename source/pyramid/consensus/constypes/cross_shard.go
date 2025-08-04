package constypes

import (
	pb "emulator/proto/pyramid/consensus/types"
	"emulator/pyramid/types"

	"google.golang.org/protobuf/proto"
)

type CrossShardBlock struct {
	Block               *types.Block
	ColleciveSignatures *PrecommitAggregated
}

func (csb *CrossShardBlock) ToProto() *pb.CrossShardBlock {
	return &pb.CrossShardBlock{
		Block:                csb.Block.ToProto(),
		CollectiveSignatures: csb.ColleciveSignatures.ToProto(),
	}
}
func (csb *CrossShardBlock) ProtoBytes() []byte {
	return types.MustProtoBytes(csb.ToProto())
}
func NewCrossShardBlockFromProto(p *pb.CrossShardBlock) *CrossShardBlock {
	return &CrossShardBlock{
		Block:               types.NewBlockFromProto(p.Block),
		ColleciveSignatures: NewPrecommitAggregatedFromProto(p.CollectiveSignatures),
	}
}
func NewCrossShardBlockFromBytes(bz []byte) (*CrossShardBlock, error) {
	var csb = new(pb.CrossShardBlock)
	if err := proto.Unmarshal(bz, csb); err != nil {
		return nil, err
	}
	return NewCrossShardBlockFromProto(csb), nil
}

type MessageAccept struct {
	B_BlockHash          []byte
	CollectiveSignatures *PrecommitAggregated
}

func (ma *MessageAccept) ToProto() *pb.MessageAccept {
	return &pb.MessageAccept{
		BBlockHash:           ma.B_BlockHash,
		CollectiveSignatures: ma.CollectiveSignatures.ToProto(),
	}
}

func (ma *MessageAccept) ProtoBytes() []byte {
	return types.MustProtoBytes(ma.ToProto())
}

func NewMessageAcceptFromProto(p *pb.MessageAccept) *MessageAccept {
	return &MessageAccept{
		B_BlockHash:          p.BBlockHash,
		CollectiveSignatures: NewPrecommitAggregatedFromProto(p.CollectiveSignatures),
	}
}

func NewMessageAcceptFromBytes(bz []byte) (*MessageAccept, error) {
	var ma = new(pb.MessageAccept)
	if err := proto.Unmarshal(bz, ma); err != nil {
		return nil, err
	}
	return NewMessageAcceptFromProto(ma), nil
}

type MessageAcceptSet struct {
	Accepts   []*MessageAccept
	BlockHash []byte
}

func (mas *MessageAcceptSet) ToProto() *pb.MessageAcceptSet {
	accepts := make([]*pb.MessageAccept, len(mas.Accepts))
	for i, accept := range mas.Accepts {
		accepts[i] = accept.ToProto()
	}
	return &pb.MessageAcceptSet{
		BlockHash: mas.BlockHash,
		Accepts:   accepts,
	}
}

func (mas *MessageAcceptSet) ProtoBytes() []byte {
	return types.MustProtoBytes(mas.ToProto())
}

func NewMessageAcceptSetFromProto(p *pb.MessageAcceptSet) *MessageAcceptSet {
	accepts := make([]*MessageAccept, len(p.Accepts))
	for i, accept := range p.Accepts {
		accepts[i] = NewMessageAcceptFromProto(accept)
	}
	return &MessageAcceptSet{
		BlockHash: p.BlockHash,
		Accepts:   accepts,
	}
}

func NewMessageAcceptSetFromBytes(bz []byte) (*MessageAcceptSet, error) {
	var mas = new(pb.MessageAcceptSet)
	if err := proto.Unmarshal(bz, mas); err != nil {
		return nil, err
	}
	accepts := NewMessageAcceptSetFromProto(mas)
	return accepts, nil
}

type MessageCommit struct {
	B_BlockHash          []byte
	CollectiveSignatures *PrecommitAggregated
}

func (mc *MessageCommit) ToProto() *pb.MessageCommit {
	return &pb.MessageCommit{
		BBlockHash:           mc.B_BlockHash,
		CollectiveSignatures: mc.CollectiveSignatures.ToProto(),
	}
}

func (mc *MessageCommit) ProtoBytes() []byte {
	return types.MustProtoBytes(mc.ToProto())
}

func NewMessageCommitFromProto(p *pb.MessageCommit) *MessageCommit {
	return &MessageCommit{
		B_BlockHash:          p.BBlockHash,
		CollectiveSignatures: NewPrecommitAggregatedFromProto(p.CollectiveSignatures),
	}
}

func NewMessageCommitFromBytes(bz []byte) (*MessageCommit, error) {
	var ma = new(pb.MessageCommit)
	if err := proto.Unmarshal(bz, ma); err != nil {
		return nil, err
	}
	return NewMessageCommitFromProto(ma), nil
}

type MessageOK struct {
	Height   int64
	SrcChain string
}

func (m *MessageOK) ToProto() *pb.MessageOK {
	return &pb.MessageOK{
		Chain_ID: m.SrcChain,
		Height:   m.Height,
	}
}
func (m *MessageOK) ProtoBytes() []byte {
	return types.MustProtoBytes(m.ToProto())
}
func NewMessageOKFromProto(p *pb.MessageOK) *MessageOK {
	return &MessageOK{
		Height:   p.Height,
		SrcChain: p.Chain_ID,
	}
}
func NewMessageOKFromBytes(bz []byte) (*MessageOK, error) {
	var mo = new(pb.MessageOK)
	if err := proto.Unmarshal(bz, mo); err != nil {
		return nil, err
	}
	return NewMessageOKFromProto(mo), nil
}
