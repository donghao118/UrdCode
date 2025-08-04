package hotstuff

import (
	"emulator/crypto/merkle"
	protovote "emulator/proto/core/hotstuff"
	"emulator/utils"
	"emulator/utils/signer"

	"google.golang.org/protobuf/proto"
)

type Vote struct {
	Code int8

	View             int64
	Round            int32
	ForHash          []byte
	ForNecessaryData [][]byte

	Sign           string
	ValidatorIndex int
}

var _ signer.VerifiableType = (*Vote)(nil)
var _ signer.VerifiableType = (*AggregatedVote)(nil)

func NewVote(view int64, round int32, hash []byte, necessary_data [][]byte, index int) *Vote {
	return &Vote{
		Code: utils.CodeTypeOK,

		View:             view,
		Round:            round,
		ForHash:          hash,
		ForNecessaryData: necessary_data,

		Sign:           "",
		ValidatorIndex: index,
	}
}
func (v *Vote) IsOK() bool {
	return v.Code == utils.CodeTypeOK
}
func (v *Vote) SetOK() {
	v.Code = utils.CodeTypeOK
}
func (v *Vote) SetReject() {
	v.Code = utils.CodeTypeReject
}

func (v *Vote) ToProto() *protovote.Vote {
	if v == nil {
		return nil
	}
	return &protovote.Vote{
		Code: int32(v.Code),

		View:             v.View,
		Round:            v.Round,
		ForHash:          v.ForHash,
		ForNecessaryData: v.ForNecessaryData,

		Sign:           v.Sign,
		ValidatorIndex: int32(v.ValidatorIndex),
	}
}

func (v *Vote) ProtoBytes() []byte {
	return utils.MustProtoBytes(v.ToProto())
}
func (v *Vote) SignBytes() []byte {
	return utils.MustProtoBytes(
		&protovote.Vote{
			Code: int32(v.Code),

			Round:            v.Round,
			View:             v.View,
			ForHash:          v.ForHash,
			ForNecessaryData: v.ForNecessaryData,

			Sign:           "",
			ValidatorIndex: -1,
		},
	)
}

func NewVoteFromProto(p *protovote.Vote) *Vote {
	return &Vote{
		Code: int8(p.Code),

		Round:            p.Round,
		View:             p.View,
		ForHash:          p.ForHash,
		ForNecessaryData: p.ForNecessaryData,

		Sign:           p.Sign,
		ValidatorIndex: int(p.ValidatorIndex),
	}
}
func NewVoteFromBytes(bz []byte) *Vote {
	var p = new(protovote.Vote)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewVoteFromProto(p)
	}
}

func (v *Vote) GetSign() string {
	return v.Sign
}
func (v *Vote) ValidateBasic() error {
	if v.Code != utils.CodeTypeOK && v.Code != utils.CodeTypeReject {
		return utils.ErrInvalidVoteCode
	}
	if v.ValidatorIndex < 0 {
		return utils.ErrInvalidValidatorIndex
	}
	if v.Round < 0 {
		return utils.ErrInvalidRound
	}
	if v.View < 0 {
		return utils.ErrInvalidView
	}
	return nil
}

func (v *Vote) Hash() []byte {
	return merkle.HashFromByteSlices([][]byte{v.ProtoBytes()})
}

type AggregatedVote struct {
	Code int8

	View    int64
	Round   int32
	ForHash []byte

	ForNecessaryData [][]byte

	Sign          string
	SignerIndexer *utils.BitVector
}

func (v *Vote) GetAggregated(bv *utils.BitVector) *AggregatedVote {
	return &AggregatedVote{
		Code: v.Code,

		View:             v.View,
		Round:            v.Round,
		ForHash:          v.ForHash,
		ForNecessaryData: v.ForNecessaryData,

		Sign:          v.Sign,
		SignerIndexer: bv,
	}
}

func (v *AggregatedVote) IsOK() bool {
	return v.Code == utils.CodeTypeOK
}

func (v *AggregatedVote) ToProto() *protovote.AggregatedVote {
	if v == nil {
		return nil
	}
	return &protovote.AggregatedVote{
		Code: int32(v.Code),

		View:             v.View,
		Round:            v.Round,
		ForHash:          v.ForHash,
		ForNecessaryData: v.ForNecessaryData,

		Sign:        v.Sign,
		SignerIndex: v.SignerIndexer.Byte(),
	}
}

func (v *AggregatedVote) ProtoBytes() []byte {
	return utils.MustProtoBytes(v.ToProto())
}
func (v *AggregatedVote) Hash() []byte {
	return merkle.HashFromByteSlices([][]byte{v.ProtoBytes()})
}
func (v *AggregatedVote) SignBytes() []byte {
	return utils.MustProtoBytes(
		&protovote.AggregatedVote{
			Code: int32(v.Code),

			Round:            v.Round,
			View:             v.View,
			ForHash:          v.ForHash,
			ForNecessaryData: v.ForNecessaryData,

			Sign:        "",
			SignerIndex: nil,
		},
	)
}

func NewAggregatedVoteFromProto(p *protovote.AggregatedVote) *AggregatedVote {
	if p == nil {
		return nil
	}
	return &AggregatedVote{
		Code: int8(p.Code),

		Round:            p.Round,
		View:             p.View,
		ForHash:          p.ForHash,
		ForNecessaryData: p.ForNecessaryData,

		Sign:          p.Sign,
		SignerIndexer: utils.NewBitArrayFromByte(p.SignerIndex),
	}
}
func NewAggregatedVoteFromBytes(bz []byte) *AggregatedVote {
	var p = new(protovote.AggregatedVote)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewAggregatedVoteFromProto(p)
	}
}

func (v *AggregatedVote) GetSign() string {
	return v.Sign
}
func (v *AggregatedVote) ValidateBasic() error {
	if v.Code != utils.CodeTypeOK && v.Code != utils.CodeTypeReject {
		return utils.ErrInvalidVoteCode
	}
	if v.SignerIndexer.Size() <= 0 {
		return utils.ErrInvalidValidatorIndex
	}
	if v.Round < 0 {
		return utils.ErrInvalidRound
	}
	if v.View < 0 {
		return utils.ErrInvalidView
	}
	return nil
}
