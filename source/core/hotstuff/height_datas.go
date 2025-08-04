package hotstuff

import (
	"bytes"
	"emulator/utils"
	crypto "emulator/utils/signer"
	"fmt"
)

type HeightDataPackage struct {
	ProposalHash []byte
	View         int64
	Round        int32

	Votes          []*Vote
	VotesBitVector *utils.BitVector
	VotesTotal     int
	RejectTotal    int
	VotesNeeded    int
	PerVote        []int
	ValidatorSet   *crypto.Verifier

	AggVote *AggregatedVote
}

func NewHeightDataPackage(validatorSet *crypto.Verifier, perVote []int, view int64, round int32) *HeightDataPackage {
	votesNeeded := 0
	for _, vote := range perVote {
		votesNeeded += vote
	}

	return &HeightDataPackage{
		View:           view,
		Round:          round,
		Votes:          make([]*Vote, validatorSet.Size()),
		VotesBitVector: utils.NewBitVector(validatorSet.Size()),
		VotesTotal:     0,
		VotesNeeded:    votesNeeded,
		ValidatorSet:   validatorSet,
		PerVote:        perVote,
	}
}
func (hdp *HeightDataPackage) addVote(vote *Vote) error {
	if !hdp.ValidatorSet.Verify(string(vote.GetSign()), vote.SignBytes(), vote.ValidatorIndex) {
		return utils.ErrInvalidSign
	}
	if hdp.View != vote.View ||
		hdp.Round != vote.Round ||
		!bytes.Equal(hdp.ProposalHash, vote.ForHash) {
		return utils.ErrInvalidVoteCode
	}
	if hdp.Votes[vote.ValidatorIndex] == nil {
		if vote.IsOK() {
			hdp.VotesTotal += hdp.PerVote[vote.ValidatorIndex]
		} else {
			hdp.RejectTotal += hdp.PerVote[vote.ValidatorIndex]
		}
		hdp.Votes[vote.ValidatorIndex] = vote
		hdp.VotesBitVector.SetIndex(vote.ValidatorIndex, true)
	} else {
		return utils.ErrDuplicatedVote
	}
	return nil
}

func (hdp *HeightDataPackage) isQuorum() bool {
	return hdp.VotesTotal >= 2*hdp.VotesNeeded/3 || hdp.RejectTotal >= 2*hdp.VotesNeeded/3
}

func (hdp *HeightDataPackage) getMaj23() (*AggregatedVote, error) {
	if hdp.HasAggVote() {
		return hdp.AggVote, nil
	}
	bv := utils.NewBitVector(hdp.ValidatorSet.Size())
	aggVote := &Vote{
		Round:   hdp.Round,
		View:    hdp.View,
		ForHash: hdp.ProposalHash,
	}
	votes := []string{}

	if hdp.VotesTotal >= 2*hdp.VotesNeeded/3 {
		aggVote.SetOK()
	} else if hdp.RejectTotal >= 2*hdp.VotesNeeded/3 {
		aggVote.SetReject()
	} else {
		return nil, fmt.Errorf("error no maj23")
	}
	for index, vote := range hdp.Votes {
		if vote == nil {
			continue
		}
		if vote.IsOK() == aggVote.IsOK() {
			bv.SetIndex(index, true)
		}
		votes = append(votes, string(vote.GetSign()))
		aggVote.ForNecessaryData = vote.ForNecessaryData
	}
	aggSig, err := crypto.AggregateSignatures(votes)
	if err != nil {
		return nil, err
	}
	aggVote.Sign = aggSig
	vote := aggVote.GetAggregated(bv)
	hdp.AggVote = vote
	return vote, nil
}

func (hdp *HeightDataPackage) validate_aggregated(aggVote *AggregatedVote) error {
	if !hdp.ValidatorSet.VerifyAggregateSignature(string(aggVote.Sign), aggVote.SignBytes(), aggVote.SignerIndexer.Byte()) {
		return utils.ErrInvalidSign
	}
	if hdp.View != aggVote.View ||
		hdp.Round != aggVote.Round {
		return utils.ErrInvalidVoteCode
	}

	hdp.AggVote = aggVote
	return nil
}

func (hdp *HeightDataPackage) HasAggVote() bool {
	return hdp.AggVote != nil
}
