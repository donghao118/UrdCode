package constypes

import (
	"emulator/crypto/hash"
	pbcons "emulator/proto/urd/types"
	"emulator/urd/types"
	"encoding/hex"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// ============ Proposal ========================
type Proposal struct {
	Header          *types.PartSetHeader
	ProposerIndex   int
	BlockHeaderHash []byte
	// TimeStamp     time.Time

	// POLRound  int   // Tendermint共识需要这个字段

	Signature string
}

func NewProposal(header *types.PartSetHeader, proposer_index int, blockHeaderHash []byte) *Proposal {
	return &Proposal{
		Header:          header,
		ProposerIndex:   proposer_index,
		BlockHeaderHash: blockHeaderHash,
	}
}

func (p *Proposal) ToProto() *pbcons.Proposal {
	return &pbcons.Proposal{
		Header:          p.Header.ToProto(),
		ProposerIndex:   int32(p.ProposerIndex),
		BlockHeaderHash: p.BlockHeaderHash,
		Signature:       p.Signature,
	}
}

func (p *Proposal) ProtoBytes() []byte {
	return types.MustProtoBytes(p.ToProto())
}
func (p *Proposal) SignBytes() []byte {
	return types.MustProtoBytes(
		&pbcons.Proposal{
			Header:          p.Header.ToProto(),
			ProposerIndex:   int32(p.ProposerIndex),
			BlockHeaderHash: p.BlockHeaderHash,
			Signature:       "",
		},
	)
}
func NewProposalFromProto(p *pbcons.Proposal) *Proposal {
	return &Proposal{
		Header:          types.NewPartSetHeaderFromProto(p.Header),
		ProposerIndex:   int(p.ProposerIndex),
		BlockHeaderHash: p.BlockHeaderHash,
		Signature:       p.Signature,
	}
}
func NewProposalFromBytes(bz []byte) *Proposal {
	var p = new(pbcons.Proposal)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewProposalFromProto(p)
	}
}
func (p *Proposal) ValidateBasic() error {
	if p.ProposerIndex < 0 {
		return fmt.Errorf(fmt.Sprintf("ProposerIndex为负数: %d", p.ProposerIndex))
	}
	if len(p.BlockHeaderHash) != hash.HashSize {
		return fmt.Errorf(hex.EncodeToString(p.BlockHeaderHash) + " 不是一个哈希值")
	}

	return p.Header.ValidateBasic()
}
