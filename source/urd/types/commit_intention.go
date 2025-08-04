package types

import (
	"emulator/core/hotstuff"
	"emulator/crypto/merkle"
	"emulator/utils"

	prototypes "emulator/proto/urd/types"

	"google.golang.org/protobuf/proto"
)

type IntentionBrief [][]byte

func (ib IntentionBrief) Hash() []byte { return merkle.HashFromByteSlices(ib) }

type CommitIntention struct {
	IntentionHash        IntentionBrief
	AggregatedSignatures []*hotstuff.AggregatedVote

	rightHash, leftHash []byte
}

func (ci *CommitIntention) ToProto() *prototypes.CommitIntention {
	if ci == nil {
		return nil
	}
	pci := new(prototypes.CommitIntention)
	pci.IntentionHash = ci.IntentionHash
	for _, as := range ci.AggregatedSignatures {
		pci.AggregatedSignatures = append(pci.AggregatedSignatures, as.ToProto())
	}
	return pci
}

func NewCommitIntentionFromProto(pci *prototypes.CommitIntention) *CommitIntention {
	if pci == nil {
		return nil
	}
	ci := new(CommitIntention)
	ci.IntentionHash = pci.IntentionHash
	for _, as := range pci.AggregatedSignatures {
		ci.AggregatedSignatures = append(ci.AggregatedSignatures, hotstuff.NewAggregatedVoteFromProto(as))
	}
	return ci
}

func NewCommitIntentionFromBytes(data []byte) *CommitIntention {
	pci := new(prototypes.CommitIntention)
	if err := proto.Unmarshal(data, pci); err != nil {
		return nil
	}
	return NewCommitIntentionFromProto(pci)
}

func (ci *CommitIntention) ProtoBytes() []byte {
	return utils.MustProtoBytes(ci.ToProto())
}

func (ci *CommitIntention) Hash() []byte {
	return merkle.HashFromByteSlices([][]byte{ci.LeftHash(), ci.RrightHash()})
}
func (ci *CommitIntention) LeftHash() []byte {
	if len(ci.leftHash) == 0 {
		ci.leftHash = ci.IntentionHash.Hash()
	}
	return ci.leftHash
}
func (ci *CommitIntention) RrightHash() []byte {
	if len(ci.rightHash) == 0 {
		lst2 := make([][]byte, len(ci.AggregatedSignatures))
		for i, as := range ci.AggregatedSignatures {
			lst2[i] = as.Hash()
		}
		ci.rightHash = merkle.HashFromByteSlices(lst2)
	}
	return ci.rightHash
}

func (ci *CommitIntention) GetBrief() IntentionBrief {
	return ci.IntentionHash
}

type ProofOfIntention struct {
	IntentionHashProof *merkle.Proof
	RightHash          []byte
}

func (poi *ProofOfIntention) Verify(root, leftHash []byte) error {
	if poi == nil {
		return nil
	}
	ciHash := merkle.HashFromByteSlices([][]byte{leftHash, poi.RightHash})
	if err := poi.IntentionHashProof.Verify(root, ciHash); err != nil {
		return err
	}
	return nil
}

func (poi *ProofOfIntention) ToProto() *prototypes.ProofOfIntention {
	if poi == nil {
		return nil
	}
	return &prototypes.ProofOfIntention{
		IntentionHashProof: poi.IntentionHashProof.ToProto(),
		RightHash:          poi.RightHash,
	}
}
func (poi *ProofOfIntention) ProtoBytes() []byte {
	return utils.MustProtoBytes(poi.ToProto())
}
func NewProofOfIntentionFromProto(ppoi *prototypes.ProofOfIntention) *ProofOfIntention {
	if ppoi == nil {
		return nil
	}
	return &ProofOfIntention{
		IntentionHashProof: merkle.NewProofFromProto(ppoi.IntentionHashProof),
		RightHash:          ppoi.RightHash,
	}
}
func NewProofOfIntentionFromBytes(data []byte) *ProofOfIntention {
	ppoi := new(prototypes.ProofOfIntention)
	if err := proto.Unmarshal(data, ppoi); err != nil {
		return nil
	}
	return NewProofOfIntentionFromProto(ppoi)
}

func (poi *ProofOfIntention) Hash() []byte {
	return merkle.HashFromByteSlices(
		[][]byte{
			poi.IntentionHashProof.ContentHash(),
			poi.RightHash,
		},
	)
}
