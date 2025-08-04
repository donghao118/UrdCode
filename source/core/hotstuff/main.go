package hotstuff

import (
	crypto "emulator/utils/signer"
	sig "emulator/utils/signer"
)

type State struct {
	View  int64
	Round int32

	signer      *sig.Signer
	signerIndex int

	heightDatas  *HeightDataPackage
	ValidatorSet *crypto.Verifier
	PerVotes     []int
}

func NewState(view int64, round int32, signer *sig.Signer, index int, validatorSet *crypto.Verifier, perVotes []int) *State {
	return &State{
		View:         view,
		Round:        round,
		signer:       signer,
		signerIndex:  index,
		heightDatas:  nil,
		ValidatorSet: validatorSet,
		PerVotes:     perVotes,
	}
}

func (state *State) SetHash(hash []byte) {
	if state.heightDatas == nil {
		state.heightDatas = NewHeightDataPackage(state.ValidatorSet, state.PerVotes, state.View, state.Round)
	}
	state.heightDatas.ProposalHash = hash
}

func (state *State) UpdateValidators(validatorSet *crypto.Verifier, perVotes []int) {
	state.ValidatorSet = validatorSet
	state.PerVotes = perVotes
}

func (state *State) EnterNewRound() {
	state.Round++
	state.heightDatas = NewHeightDataPackage(state.ValidatorSet, state.PerVotes, state.View, state.Round)
	state.heightDatas.View, state.heightDatas.Round = state.View, state.Round
}

func (state *State) EnterNewView() {
	state.View++
	state.Round = 0
	state.heightDatas = NewHeightDataPackage(state.ValidatorSet, state.PerVotes, state.View, state.Round)
	state.heightDatas.View, state.heightDatas.Round = state.View, state.Round
}

func (state *State) AddVote(vote *Vote) error {
	if err := state.heightDatas.addVote(vote); err != nil {
		return err
	}
	return nil
}

func (state *State) IsQuorum() bool {
	return state.heightDatas.isQuorum()
}

func (state *State) GetMaj23() (*AggregatedVote, error) {
	aggVote, err := state.heightDatas.getMaj23()
	return aggVote, err
}

func (state *State) ValidateAggregated(aggVote *AggregatedVote) error {
	return state.heightDatas.validate_aggregated(aggVote)
}
