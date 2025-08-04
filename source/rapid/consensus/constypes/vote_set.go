package constypes

import (
	"bytes"
	"emulator/utils"
	crypto "emulator/utils/signer"
	"fmt"
)

type Vote interface {
	GetBlockHeaderHash() []byte
	GetIndex() int
	IsOK() bool
	GetSignature() string
}

type VoteSet struct {
	validatorNum       int
	totalVotes         int32
	votes              []Vote
	blockHeaderHashCnt map[string][]int32
	bitVector          *utils.BitVector

	has_mj23  bool
	mj23_key  string
	mj23_code bool
}

func NewVoteSet(n int, totalVotes int32) *VoteSet {
	return &VoteSet{
		validatorNum:       n,
		totalVotes:         totalVotes,
		votes:              make([]Vote, n),
		blockHeaderHashCnt: map[string][]int32{},
		bitVector:          utils.NewBitVector(n),

		has_mj23:  false,
		mj23_key:  "",
		mj23_code: false,
	}
}

func voteIndex(vote Vote) int {
	if vote.IsOK() {
		return 0
	} else {
		return 1
	}
}

func (vs *VoteSet) AddVote(vote Vote, valVote int32) error {
	key := string(vote.GetBlockHeaderHash())

	if vs.bitVector.GetIndex(vote.GetIndex()) {
		rawVote := vs.votes[vote.GetIndex()]
		if bytes.Equal(vote.GetBlockHeaderHash(), rawVote.GetBlockHeaderHash()) ||
			vote.IsOK() != rawVote.IsOK() {
			return fmt.Errorf("The voter casts at least two ambiguous votes")
		}
		return nil
	}

	vs.bitVector.SetIndex(vote.GetIndex(), true)
	vs.votes[vote.GetIndex()] = vote
	if _, ok := vs.blockHeaderHashCnt[key]; !ok {
		u := []int32{0, 0}
		u[voteIndex(vote)] += valVote
		vs.blockHeaderHashCnt[key] = u
	} else {
		vs.blockHeaderHashCnt[key][voteIndex(vote)] += valVote
	}

	if vs.has_mj23 {
		return nil
	}

	if vs.blockHeaderHashCnt[key][voteIndex(vote)] > 2*vs.totalVotes/3 {
		vs.has_mj23 = true
		vs.mj23_key = key
		vs.mj23_code = vote.IsOK()
	}
	return nil
}

func (vs *VoteSet) HasMaj23() bool { return vs.has_mj23 }
func (vs *VoteSet) GetMaj23() ([]byte, bool) {
	return []byte(vs.mj23_key), vs.mj23_code
}
func (vs *VoteSet) GetMaj23AggreSig() (string, []byte) {
	aggSigs := []string{}
	bv := utils.NewBitVector(vs.validatorNum)
	bz, ok := vs.GetMaj23()
	for i := 0; i < vs.validatorNum; i++ {
		if vs.bitVector.GetIndex(i) {
			if vote := vs.votes[i]; vote.IsOK() == ok && bytes.Equal(bz, vote.GetBlockHeaderHash()) {
				aggSigs = append(aggSigs, vote.GetSignature())
				bv.SetIndex(i, true)
			}
		}
	}
	aggSig, err := crypto.AggregateSignatures(aggSigs)
	if err != nil {
		panic(err)
	}
	return aggSig, bv.Byte()
}
