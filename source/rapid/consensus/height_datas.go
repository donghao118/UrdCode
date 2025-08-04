package tendermint

import (
	constypes "emulator/rapid/consensus/constypes"
	"emulator/rapid/shardinfo"
	"emulator/rapid/types"
	"emulator/utils/p2p"
	crypto "emulator/utils/signer"
	"fmt"
	"log"
)

var (
	DoNothing = fmt.Errorf("")
)

type HeightDataPackage struct {
	Height    int64
	Round     int
	MyChainID string
	PassStep  int8

	ShardInfo       *shardinfo.ShardInfo
	ShardVotes      map[string]int
	AllValidatorSet map[string]*crypto.Verifier

	Proposal *constypes.Proposal

	Prevotes *constypes.VoteSet

	Precommits *constypes.VoteSet

	PartSet *types.PartSet
	Block   *types.Block

	randomHeaderBuffer []*types.Part
	p2pConn            *p2p.Sender
}

func (h *HeightDataPackage) IsIShard() bool { return h.ShardInfo.IsIShard() }
func (h *HeightDataPackage) IsBShard() bool { return h.ShardInfo.IsBShard() }

// ================================================================================

func NewHeightData(mychainid string, height int64, si *shardinfo.ShardInfo, sender *p2p.Sender) *HeightDataPackage {
	h := new(HeightDataPackage)
	h.Height, h.Round = height, 0
	h.PassStep = RoundStepNewHeight
	h.MyChainID = mychainid
	h.p2pConn = sender
	h.SetShardInfo(si)
	h.clearAll()
	h.randomHeaderBuffer = nil
	return h
}
func (h *HeightDataPackage) NextHeight() {
	h.Height++
	h.Round = 0
	h.PassStep = RoundStepNewHeight
	h.clearAll()
}
func (h *HeightDataPackage) setKeys(pubkeys []string, chain_id string) error {
	verifier, err := crypto.NewVerifier(pubkeys)
	if err != nil {
		return err
	} else if verifier == nil {
		return fmt.Errorf("HeightData initialization: empty Validator error")
	}
	h.AllValidatorSet[chain_id] = verifier
	return nil
}

func (h *HeightDataPackage) SetShardInfo(s *shardinfo.ShardInfo) error {
	h.ShardInfo = s
	h.AllValidatorSet = make(map[string]*crypto.Verifier)
	h.ShardVotes = make(map[string]int)
	//h.AllValidatorSetSize = make(map[string]int)
	for chain_id, peers := range s.PeerList {
		totalVotes := 0
		peerPubKeys := make([]string, len(peers))
		for i, peer := range peers {
			peerPubKeys[i] = peer.Pubkey
			totalVotes += int(peer.Vote)
		}
		if err := h.setKeys(peerPubKeys, chain_id); err != nil {
			return err
		}
		h.ShardVotes[chain_id] = totalVotes
	}
	return nil
}

func (h *HeightDataPackage) clearProposal() {
	h.Proposal = nil
	h.Block = nil
	h.PartSet = nil
}
func (h *HeightDataPackage) clearPrevote() {
	size := h.AllValidatorSet[h.MyChainID].Size()
	totalVotes := h.ShardVotes[h.MyChainID]
	h.Prevotes = constypes.NewVoteSet(size, int32(totalVotes))
}
func (h *HeightDataPackage) clearPrecommit() {
	size := h.AllValidatorSet[h.MyChainID].Size()
	totalVotes := h.ShardVotes[h.MyChainID]
	h.Precommits = constypes.NewVoteSet(size, int32(totalVotes))
}

func (h *HeightDataPackage) clearAll() {
	h.clearProposal()
	h.clearPrevote()
	h.clearPrecommit()
}

func (h *HeightDataPackage) FinishProposal() bool {
	if h.PassStep >= RoundStepPropose {
		return true
	}
	if h.Proposal == nil {
		return false
	}
	if h.Block == nil || !h.PartSet.IsComplete() {
		return false
	}

	h.PassStep = RoundStepPropose
	return true
}
func (h *HeightDataPackage) FinishPrevote() bool {
	if h.PassStep >= RoundStepPrevote {
		return true
	}
	if !h.FinishProposal() {
		return false
	}
	if !h.Prevotes.HasMaj23() {
		return false
	}
	h.PassStep = RoundStepPrevote
	return true
}
func (h *HeightDataPackage) FinishPrecommit() bool {
	if h.PassStep >= RoundStepPrecommit {
		return true
	}
	if !h.FinishPrevote() {
		return false
	}
	if !h.Precommits.HasMaj23() {
		return false
	}
	h.PassStep = RoundStepPrecommit
	return true
}

func (h *HeightDataPackage) AddProposal(p *constypes.Proposal, index int) error {
	if h.Proposal != nil {
		return DoNothing
	}
	log.Println("Start processing the Proposal")
	if h.Height != p.Header.Height || h.Round != p.Header.Round {
		return DoNothing
	}
	if p.ProposerIndex != index {
		return fmt.Errorf(fmt.Sprintf("Proposer error: Should have been %d, received %d", index, p.ProposerIndex))
	}
	if p.Header.ChainID != h.MyChainID {
		return fmt.Errorf(fmt.Sprintf("Proposal validation failed: I am a %s shard and do not accept the Proposal for %s shard", h.MyChainID, p.Header.ChainID))
	}
	if !h.AllValidatorSet[h.MyChainID].Verify(p.Signature, p.SignBytes(), index) {
		return fmt.Errorf(fmt.Sprintf("Proposal signature verification failed"))
	}
	h.Proposal = p
	h.PartSet = types.NewPartSet(p.Header, p.BlockHeaderHash)
	for _, part := range h.randomHeaderBuffer {
		h.AddPart(part)
	}
	return nil
}

func (h *HeightDataPackage) AddPart(p *types.Part) error {
	if h.Height != p.Height || h.Round != p.Round {
		return DoNothing
	}
	if p.ChainID != h.MyChainID {
		return fmt.Errorf("Received parts from other shards")
	}
	if h.PartSet == nil {
		h.randomHeaderBuffer = append(h.randomHeaderBuffer, p)
		log.Println("Part joins the cache pool")
		return DoNothing
	} else if err := h.PartSet.AddPart(p); err != nil && err != types.DuplicatedPartPassError {
		return err
	} else if err == types.DuplicatedPartPassError {
		return DoNothing
	} else {
		if h.PartSet.IsComplete() {
			block, err := h.PartSet.GenBlock()
			if err != nil {
				return err
			}
			log.Printf("(height=%d) shard %s block has been collected\n", h.Height, p.ChainID)
			h.Block = block
		}
	}
	return nil
}

func (h *HeightDataPackage) AddPrevote(p *constypes.Prevote) error {
	if h.Height != p.Height || h.Round != p.Round {
		return DoNothing
	}
	if maxSize := h.AllValidatorSet[h.MyChainID].Size(); p.ValidatorIndex < 0 || p.ValidatorIndex >= maxSize {
		return fmt.Errorf(fmt.Sprintf("Proposer error: Should have been at [0,%d], received %d", maxSize-1, p.ValidatorIndex))
	}

	if !h.AllValidatorSet[h.MyChainID].Verify(p.Signature, p.SignBytes(), p.ValidatorIndex) {
		return fmt.Errorf(fmt.Sprintf("The Prevote signature verification failed"))
	}
	valVote := h.ShardInfo.PeerList[h.MyChainID][p.ValidatorIndex].Vote
	return h.Prevotes.AddVote(p, valVote)
}

func (h *HeightDataPackage) AddPrecommit(p *constypes.Precommit) error {
	if h.Height != p.Height || h.Round != p.Round {
		return DoNothing
	}
	if maxSize := h.AllValidatorSet[h.MyChainID].Size(); p.ValidatorIndex < 0 || p.ValidatorIndex >= maxSize {
		return fmt.Errorf(fmt.Sprintf("Proposer error: Should have been at [0,%d], received %d", maxSize-1, p.ValidatorIndex))
	}
	if !h.AllValidatorSet[h.MyChainID].Verify(p.Signature, p.SignBytes(), p.ValidatorIndex) {
		return fmt.Errorf(fmt.Sprintf("Precommit signature validation failed"))
	}
	valVote := h.ShardInfo.PeerList[h.MyChainID][p.ValidatorIndex].Vote
	return h.Precommits.AddVote(p, valVote)
}

// ================================================================================

func (h *HeightDataPackage) GenerateAggregatePrecommitSignature() *constypes.PrecommitAggregated {
	hash, ok := h.Precommits.GetMaj23()
	sig, bv := h.Precommits.GetMaj23AggreSig()
	pa := constypes.NewPrecommitAggregated(bv, h.Proposal.Header, hash, sig)
	pa.SetCode(ok)
	return pa
}
