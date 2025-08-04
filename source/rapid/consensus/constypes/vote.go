package constypes

import (
	pbcons "emulator/proto/pyramid/consensus/types"
	"emulator/rapid/types"
	"emulator/utils"
	"emulator/utils/signer"
	"encoding/hex"
	"fmt"

	"emulator/crypto/hash"

	"google.golang.org/protobuf/proto"
)

const (
	VoteTypeNil = iota
	VoteTypeProposal
	VoteTypePrevote
	VoteTypePrecommit
	VoteTypePrecommitAggregated
	VoteTypeCrossShardProposal
	VoteTypeCrossShardAccept
	VoteTypeCrossShardCommit

	ShardTypeI
	ShardTypeB
)

// ============ Proposal ========================
type Proposal struct {
	VoteType uint32 // VoteTypeProposal

	Header          *types.PartSetHeader
	ProposerIndex   int
	BlockHeaderHash []byte

	Signature string
}

func NewProposal(header *types.PartSetHeader, proposer_index int, blockHeaderHash []byte) *Proposal {
	return &Proposal{
		VoteType:        VoteTypeProposal,
		Header:          header,
		ProposerIndex:   proposer_index,
		BlockHeaderHash: blockHeaderHash,
	}
}

func (p *Proposal) ToProto() *pbcons.Proposal {
	return &pbcons.Proposal{
		VoteType:        p.VoteType,
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
			VoteType:        p.VoteType,
			Header:          p.Header.ToProto(),
			ProposerIndex:   int32(p.ProposerIndex),
			BlockHeaderHash: p.BlockHeaderHash,
			Signature:       "",
		},
	)
}
func NewProposalFromProto(p *pbcons.Proposal) *Proposal {
	return &Proposal{
		VoteType:        p.VoteType,
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
	if p.VoteType != VoteTypeProposal {
		return fmt.Errorf(fmt.Sprintf("VoteTypeError: %d is not the VoteType for the Proposal", p.VoteType))
	}
	if p.ProposerIndex < 0 {
		return fmt.Errorf(fmt.Sprintf("The ProposerIndex is negative: %d", p.ProposerIndex))
	}
	if len(p.BlockHeaderHash) != hash.HashSize {
		return fmt.Errorf(hex.EncodeToString(p.BlockHeaderHash) + " It's not a hash")
	}

	return p.Header.ValidateBasic()
}

// ============ Prevote =================
type Prevote struct {
	VoteType uint32

	Code   int8
	Height int64
	Round  int
	// TimeStamp       time.Time
	BlockHeaderHash []byte

	Signature      string
	ValidatorIndex int
}

var _ Vote = (*Prevote)(nil)

func (p *Prevote) IsOK() bool { return p.Code == types.CodeTypeOK }
func (p *Prevote) SetOK(ok bool) {
	if ok {
		p.Code = types.CodeTypeOK
	} else {
		p.Code = types.CodeTypeAbort
	}
}
func (p *Prevote) GetBlockHeaderHash() []byte {
	return p.BlockHeaderHash
}
func (p *Prevote) GetSignature() string { return p.Signature }
func (p *Prevote) GetIndex() int        { return p.ValidatorIndex }
func (p *Prevote) ValidateBasic() error {
	if p.VoteType != VoteTypePrevote {
		return fmt.Errorf(fmt.Sprintf("VoteTypeError: %d is not a VoteType for Prevote", p.VoteType))
	}
	if len(p.BlockHeaderHash) != hash.HashSize {
		return fmt.Errorf(fmt.Sprintf("Prevote malformed: BlockHeaderHash is not a hash value"))
	}
	return nil
}

func NewPrevote(height int64, round int, blockHeaderHash []byte, validatorIndex int) *Prevote {
	return &Prevote{
		Code:            types.CodeTypeOK,
		VoteType:        VoteTypePrevote,
		Height:          height,
		Round:           round,
		BlockHeaderHash: blockHeaderHash,
		ValidatorIndex:  validatorIndex,
	}
}

func (p *Prevote) ToProto() *pbcons.Prevote {
	return &pbcons.Prevote{
		Code:            int32(p.Code),
		VoteType:        p.VoteType,
		Height:          p.Height,
		Round:           int32(p.Round),
		BlockHeaderHash: p.BlockHeaderHash,
		Signature:       p.Signature,
		ValidatorIndex:  int32(p.ValidatorIndex),
	}
}

func (p *Prevote) ProtoBytes() []byte {
	return types.MustProtoBytes(p.ToProto())
}

func (p *Prevote) SignBytes() []byte {
	return types.MustProtoBytes(
		&pbcons.Prevote{
			Code:            int32(p.Code),
			VoteType:        p.VoteType,
			Height:          p.Height,
			Round:           int32(p.Round),
			BlockHeaderHash: p.BlockHeaderHash,
			Signature:       "",
			ValidatorIndex:  -1,
		},
	)
}

func NewPrevoteFromProto(p *pbcons.Prevote) *Prevote {
	return &Prevote{
		Code:            int8(p.Code),
		VoteType:        p.VoteType,
		Height:          p.Height,
		Round:           int(p.Round),
		BlockHeaderHash: p.BlockHeaderHash,
		Signature:       p.Signature,
		ValidatorIndex:  int(p.ValidatorIndex),
	}
}

func NewPrevoteFromBytes(bz []byte) *Prevote {
	var p = new(pbcons.Prevote)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewPrevoteFromProto(p)
	}
}

// ============ Precommit ====================
type Precommit struct {
	VoteType uint32
	Code     int8

	Height int64
	Round  int
	// TimeStamp       time.Time
	BlockHeaderHash []byte

	Signature      string
	ValidatorIndex int
}

var _ Vote = (*Precommit)(nil)

func (p *Precommit) IsOK() bool { return p.Code == types.CodeTypeOK }
func (p *Precommit) GetBlockHeaderHash() []byte {
	return p.BlockHeaderHash
}
func (p *Precommit) GetSignature() string { return p.Signature }
func (p *Precommit) GetIndex() int        { return p.ValidatorIndex }
func (p *Precommit) SetOK(ok bool) {
	if ok {
		p.Code = types.CodeTypeOK
	} else {
		p.Code = types.CodeTypeAbort
	}
}
func (p *Precommit) ValidateBasic() error {
	if p.VoteType != VoteTypePrecommit {
		return fmt.Errorf(fmt.Sprintf("VoteTypeError: %d is not a Precommit VoteType", p.VoteType))
	}
	if len(p.BlockHeaderHash) != hash.HashSize {
		return fmt.Errorf(fmt.Sprintf("Precommit malformed: BlockHeaderHash is not a hash value"))
	}

	return nil
}

func NewPrecommit(height int64, round int, blockHeaderHash []byte, validatorIndex int) *Precommit {
	return &Precommit{
		Code:            types.CodeTypeOK,
		VoteType:        VoteTypePrecommit,
		Height:          height,
		Round:           round,
		BlockHeaderHash: blockHeaderHash,
		ValidatorIndex:  validatorIndex,
	}
}

func (p *Precommit) ToProto() *pbcons.Precommit {
	return &pbcons.Precommit{
		Code:            int32(p.Code),
		VoteType:        p.VoteType,
		Height:          p.Height,
		Round:           int32(p.Round),
		BlockHeaderHash: p.BlockHeaderHash,
		Signature:       p.Signature,
		ValidatorIndex:  int32(p.ValidatorIndex),
	}
}

func (p *Precommit) ProtoBytes() []byte {
	return types.MustProtoBytes(p.ToProto())
}

func (p *Precommit) SignBytes() []byte {
	return types.MustProtoBytes(
		&pbcons.Precommit{
			Code:            int32(p.Code),
			VoteType:        p.VoteType,
			Height:          p.Height,
			Round:           int32(p.Round),
			BlockHeaderHash: p.BlockHeaderHash,
			Signature:       "",
			ValidatorIndex:  -1,
		},
	)
}

func NewPrecommitFromProto(p *pbcons.Precommit) *Precommit {
	return &Precommit{
		Code:            int8(p.Code),
		VoteType:        p.VoteType,
		Height:          p.Height,
		Round:           int(p.Round),
		BlockHeaderHash: p.BlockHeaderHash,
		Signature:       p.Signature,
		ValidatorIndex:  int(p.ValidatorIndex),
	}
}

func NewPrecommitFromBytes(bz []byte) *Precommit {
	var p = new(pbcons.Precommit)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewPrecommitFromProto(p)
	}
}

// ========== PrecommitAggregated =======================
type PrecommitAggregated struct {
	VoteType uint32

	Code int8

	ValidatorBitVector *utils.BitVector

	Header          *types.PartSetHeader
	BlockHeaderHash []byte
	// TimeStamps []time.Time

	AggregatedSignatures string
}

func (p *PrecommitAggregated) ValidateBasic() error {
	if p.VoteType != VoteTypePrecommitAggregated {
		return fmt.Errorf(fmt.Sprintf("VoteTypeError: %d is not a PrecommitAggregated VoteType", p.VoteType))
	}
	if p.ValidatorBitVector == nil {
		return fmt.Errorf(fmt.Sprintf("PrecommitAggregated BitVector must not be empty"))
	}
	if err := p.Header.ValidateBasic(); err != nil {
		return err
	}
	if len(p.BlockHeaderHash) != hash.HashSize {
		return fmt.Errorf(fmt.Sprintf("Precommit malformed: BlockHeaderHash is not a hash."))
	}
	return nil
}
func NewPrecommitAggregated(validatorBitVector []byte, header *types.PartSetHeader, blockHeaderHash []byte, aggregatedSignatures string) *PrecommitAggregated {
	return &PrecommitAggregated{
		VoteType:             VoteTypePrecommitAggregated,
		Code:                 types.CodeTypeOK,
		ValidatorBitVector:   utils.NewBitArrayFromByte(validatorBitVector),
		Header:               header,
		BlockHeaderHash:      blockHeaderHash,
		AggregatedSignatures: aggregatedSignatures,
	}
}
func (p *PrecommitAggregated) IsOK() bool { return p.Code == types.CodeTypeOK }
func (p *PrecommitAggregated) ToProto() *pbcons.PrecommitAggregated {
	return &pbcons.PrecommitAggregated{
		VoteType:             p.VoteType,
		Code:                 int32(p.Code),
		ValidatorBitVector:   p.ValidatorBitVector.Byte(),
		Header:               p.Header.ToProto(),
		BlockHeaderHash:      p.BlockHeaderHash,
		AggregatedSignatures: p.AggregatedSignatures,
	}
}

func (p *PrecommitAggregated) ProtoBytes() []byte {
	return types.MustProtoBytes(p.ToProto())
}

func NewPrecommitAggregatedFromProto(p *pbcons.PrecommitAggregated) *PrecommitAggregated {
	return &PrecommitAggregated{
		VoteType:             p.VoteType,
		Code:                 int8(p.Code),
		ValidatorBitVector:   utils.NewBitArrayFromByte(p.ValidatorBitVector),
		Header:               types.NewPartSetHeaderFromProto(p.Header),
		BlockHeaderHash:      p.BlockHeaderHash,
		AggregatedSignatures: p.AggregatedSignatures,
	}
}

func NewPrecommitAggregatedFromBytes(bz []byte) *PrecommitAggregated {
	var p = new(pbcons.PrecommitAggregated)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewPrecommitAggregatedFromProto(p)
	}
}

func (p *PrecommitAggregated) Precommit() *Precommit {
	precommit := NewPrecommit(p.Header.Height, p.Header.Round, p.BlockHeaderHash, -1)
	precommit.Code = p.Code
	return precommit
}

func (p *PrecommitAggregated) VerifySignatures(verifier *signer.Verifier, bitVectorBytes []byte) bool {
	if verifier == nil {
		return false
	}
	precommit := p.Precommit()
	return verifier.VerifyAggregateSignature(p.AggregatedSignatures, precommit.SignBytes(), bitVectorBytes)
}
func (p *PrecommitAggregated) SetCode(ok bool) {
	if ok {
		p.Code = types.CodeTypeOK
	} else {
		p.Code = types.CodeTypeAbort
	}
}

func isNotHash(bz []byte) bool { return len(bz) != hash.HashSize }
