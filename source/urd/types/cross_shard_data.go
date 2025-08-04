package types

import (
	"emulator/core/hotstuff"
	prototypes "emulator/proto/urd/types"

	"google.golang.org/protobuf/proto"
)

type CrossShardMessage struct {
	SourceChain      string                   // making round j block
	AggVote          *hotstuff.AggregatedVote // of j - 1
	IntentionBrief   IntentionBrief           // of j - 2
	ProofOfIntention *ProofOfIntention        // of j - 2
	OPTXs            Txs                      // of j - 2
	OutputTxsProof   *OutputTxsProof          // of j - 2
}

func (csm *CrossShardMessage) ValidateBasic() error {
	if err := csm.ProofOfIntention.Verify(csm.GetLastHash(), csm.IntentionBrief.Hash()); err != nil {
		return err
	}
	if err := csm.OutputTxsProof.Verify(csm.GetLastHash(), csm.OPTXs); err != nil {
		return err
	}
	return nil
}

func (csm *CrossShardMessage) ToProto() *prototypes.CrossShardMessage {
	return &prototypes.CrossShardMessage{
		SourceChain: csm.SourceChain,
		AggVote:     csm.AggVote.ToProto(),

		IntentionBrief:   csm.IntentionBrief,
		ProofOfIntention: csm.ProofOfIntention.ToProto(),

		OPTTxs:         csm.OPTXs.ToProto(),
		OutputTxsProof: csm.OutputTxsProof.ToProto(),
	}
}

func (csm *CrossShardMessage) ProtoBytes() []byte { return MustProtoBytes(csm.ToProto()) }

func NewCrossShardMessageFromProto(pb *prototypes.CrossShardMessage) (*CrossShardMessage, error) {
	return &CrossShardMessage{
		SourceChain: pb.SourceChain,
		AggVote:     hotstuff.NewAggregatedVoteFromProto(pb.AggVote),

		IntentionBrief:   pb.IntentionBrief,
		ProofOfIntention: NewProofOfIntentionFromProto(pb.ProofOfIntention),

		OPTXs:          NewTxsFromProto(pb.OPTTxs),
		OutputTxsProof: NewOutputTxsProofFromProto(pb.OutputTxsProof),
	}, nil
}

func NewCrossShardMessageFromBytes(bz []byte) (*CrossShardMessage, error) {
	var pb = new(prototypes.CrossShardMessage)
	if err := proto.Unmarshal(bz, pb); err != nil {
		return nil, err
	}
	return NewCrossShardMessageFromProto(pb)
}

func (csm *CrossShardMessage) GetLastHash() []byte {
	return GetLastHashOfAggVote(csm.AggVote)
}
func (csm *CrossShardMessage) SetLastHash(h []byte) {
	SetLastHashOfAggVote(csm.AggVote, h)
}

func GetLastHashOfAggVote(aggVote *hotstuff.AggregatedVote) []byte {
	if len(aggVote.ForNecessaryData) < 1 {
		return nil
	}
	return aggVote.ForNecessaryData[0]
}
func SetLastHashOfAggVote(aggVote *hotstuff.AggregatedVote, hash []byte) {
	if len(aggVote.ForNecessaryData) < 1 {
		aggVote.ForNecessaryData = [][]byte{hash}
	} else {
		aggVote.ForNecessaryData[0] = hash
	}
}

func GetLastHashOfVote(aggVote *hotstuff.Vote) []byte {
	if len(aggVote.ForNecessaryData) < 1 {
		return nil
	}
	return aggVote.ForNecessaryData[0]
}
func SetLastHashOfVote(aggVote *hotstuff.Vote, hash []byte) {
	if len(aggVote.ForNecessaryData) < 1 {
		aggVote.ForNecessaryData = [][]byte{hash}
	} else {
		aggVote.ForNecessaryData[0] = hash
	}
}
