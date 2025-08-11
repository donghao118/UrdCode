package consensus

import (
	"bytes"
	"emulator/core/hotstuff"
	"emulator/logger/blocklogger"
	constypes "emulator/urd/consensus/constypes"
	"emulator/urd/definition"
	inter "emulator/urd/definition"
	"emulator/urd/shardinfo"
	"emulator/urd/types"
	"emulator/utils/p2p"
	sig "emulator/utils/signer"
	"emulator/utils/store"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type State struct {
	EnablePipelineFlag bool

	HotStuffState *hotstuff.State

	mempool             inter.MempoolConn
	cross_shard_mempool inter.MempoolConn
	abci                inter.ABCIConn
	p2p                 *p2p.Sender
	store               *store.PrefixStore
	step                string

	logger blocklogger.BlockWriter

	signer        *sig.Signer
	signerIndex   int
	proposerIndex int
	verifier      *sig.Verifier
	stateLock     sync.Mutex

	block_data *BlockData

	shard_info *shardinfo.ShardInfo
	chain_id   string

	block_pool []*types.Block

	max_bytes             int
	max_cross_shard_bytes int

	cross_shard_data_bytes int
	cooperation_bytes      int
	intra_shard_bytes      int
	bytesLock              sync.Mutex
	start_time             time.Time
}

func NewState(view int64, round int32, signer *sig.Signer, signer_index int, shard_info *shardinfo.ShardInfo, chain_id string,
	mempool inter.MempoolConn, cross_shard_mempool inter.MempoolConn,
	abci inter.ABCIConn, p2p *p2p.Sender, storeDir string, logger blocklogger.BlockWriter,
	max_bytes, max_cross_shard_bytes int) (*State, error) {
	store := store.NewPrefixStore("consensus", storeDir)

	shard := shard_info.Shards[chain_id]
	var validator_keys []string
	var perVotes []int
	for _, peer := range shard.PeerList {
		perVotes = append(perVotes, int(peer.Vote))
		validator_keys = append(validator_keys, peer.Pubkey)
	}
	validators, err := sig.NewVerifier(validator_keys)
	if err != nil {
		return nil, err
	}

	hotstuff_state := hotstuff.NewState(view, round, signer, signer_index, validators, perVotes)
	lastHash := make(map[string][]byte)
	for _, id := range shard_info.ShardIDList {
		lastHash[id] = nil
	}
	block_data := NewBlockData(view, round, chain_id, lastHash, make(map[string]*types.CrossShardMessage))
	var proposer_index int = shard.LeaderIndex

	return &State{
		HotStuffState: hotstuff_state,

		mempool:             mempool,
		cross_shard_mempool: cross_shard_mempool,
		abci:                abci,
		p2p:                 p2p,
		store:               store,
		step:                "",

		logger: logger,

		signer:        signer,
		signerIndex:   signer_index,
		proposerIndex: proposer_index,
		verifier:      validators,
		stateLock:     sync.Mutex{},

		block_data: block_data,

		shard_info: shard_info,
		chain_id:   chain_id,

		block_pool: make([]*types.Block, 0),

		max_bytes:             max_bytes,
		max_cross_shard_bytes: max_cross_shard_bytes,

		bytesLock: sync.Mutex{},
	}, nil
}

func (cs *State) Start() {
	//fmt.Println(cs.Height)
	cs.stateLock.Lock()
	defer cs.stateLock.Unlock()
	cs.start_time = time.Now()
	if cs.HotStuffState.View == 0 {
		fmt.Printf("Consensus State: Starting with View %d, Round %d\n", cs.HotStuffState.View, cs.HotStuffState.Round)
		if cs.isProposer() {
			if err := cs.doPropose(); err != nil {
				panic(err)
			}
			cs.step = STEP_LEADER_VOTE
		} else {
			cs.enterNextView()
			cs.step = STEP_VALIDATOR
		}
	} else {
		cs.handle_state_transition()
	}
}
func (state *State) Stop() {
	state.store.Close()
	state.WriteCmd("Consensus State: View 100 reached, stopping")
	dur := float64(time.Since(state.start_time)) / float64(time.Second)
	fmt.Printf("Intra Shard Bandwidth: %f MB/s\n", float64(state.intra_shard_bytes)/dur/1024.0/1024.0)
	fmt.Printf("Cross Shard Bandwidth: %f MB/s\n", float64(state.cross_shard_data_bytes)/dur/1024.0/1024.0)
	fmt.Printf("Cooperation Bandwidth: %f MB/s\n", float64(state.cooperation_bytes)/dur/1024.0/1024.0)
	panic("Interupted")
}

func (state *State) fetch_block(pre_index int) *types.Block {
	if pre_index > state.block_pool_size() {
		return nil
	}
	return state.block_pool[state.block_pool_size()-pre_index]
}
func (state *State) append_block(block *types.Block) {
	state.WriteCmd(fmt.Sprintf("Append Block: %d", block.Header.View))
	if state.block_pool_size() == 7 {
		state.block_pool = append(state.block_pool[1:], block)
	} else {
		state.block_pool = append(state.block_pool, block)
	}
}
func (state *State) block_pool_size() int { return len(state.block_pool) }

func (state *State) Receive(channelID byte, bz []byte, messageType uint32) error {
	switch messageType {
	case definition.Part:
		part := types.NewPartFromBytes(bz)
		if part == nil {
			return fmt.Errorf("Part Unmarshal Error")
		}
		if err := part.ValidateBasic(); err != nil {
			return err
		}
		//state.WriteCmd(fmt.Sprintf("Received Part: %d", part.Index()))
		state.stateLock.Lock()
		defer state.stateLock.Unlock()
		err := state.doMessage(part)
		return err
	case definition.CrossShardMessage:
		csm, err := types.NewCrossShardMessageFromBytes(bz)
		if err != nil {
			return err
		}
		if err := csm.ValidateBasic(); err != nil {
			return err
		}
		state.WriteCmd(fmt.Sprintf("Received CSM: %s", csm.SourceChain))
		state.stateLock.Lock()
		defer state.stateLock.Unlock()
		err = state.doMessage(csm)
		return err
	case definition.Proposal:
		proposal := constypes.NewProposalFromBytes(bz)
		if proposal == nil {
			return fmt.Errorf("Proposal Unmarshal Error")
		}
		if err := proposal.ValidateBasic(); err != nil {
			return err
		}
		state.WriteCmd(fmt.Sprintf("Received Proposal: %d", proposal.ProposerIndex))
		state.stateLock.Lock()
		defer state.stateLock.Unlock()
		err := state.doMessage(proposal)
		return err
	case definition.Vote:
		vote := hotstuff.NewVoteFromBytes(bz)
		if vote == nil {
			return fmt.Errorf("Vote Unmarshal Error")
		}
		if err := vote.ValidateBasic(); err != nil {
			return err
		}
		//state.WriteCmd(fmt.Sprintf("Received Vote: %d", vote.ValidatorIndex))
		state.stateLock.Lock()
		defer state.stateLock.Unlock()
		err := state.doMessage(vote)
		return err
	default:
		return fmt.Errorf("Consensus State: Unknown Message Type (" + fmt.Sprint(messageType) + ")")
	}
}

func (state *State) doMessage(msg interface{}) error {
	switch msg := msg.(type) {
	case *types.Part:
		if msg.ChainID != state.chain_id {
			return fmt.Errorf("ChainID mismatch: expected %s, got %s", state.chain_id, msg.ChainID)
		}
		if err := state.block_data.addPart(msg); err != nil {
			return err
		}
	case *constypes.Proposal:
		if msg.Header.ChainID != state.chain_id {
			return fmt.Errorf("ChainID mismatch: expected %s, got %s", state.chain_id, msg.Header.ChainID)
		}
		if ok := state.verifier.Verify(msg.Signature, msg.SignBytes(), msg.ProposerIndex); !ok {
			return fmt.Errorf("Invalid signature for proposal from validator %d", msg.ProposerIndex)
		}
		if err := state.block_data.addPartSetHeader(msg.Header, msg.BlockHeaderHash); err != nil {
			return err
		}
	case *types.CrossShardMessage:
		//fmt.Println("Received CrossShardMessage:", msg.SourceChain, "Status:", len(state.block_data.finished))
		key := fmt.Sprintf("%s:%s", msg.SourceChain, msg.GetLastHash())
		if _, ok := state.shard_info.Shards[msg.SourceChain]; !ok {
			return fmt.Errorf("shard does not exists")
		} //else if !shard.VerifyAggregateSignature(string(msg.AggVote.Sign), msg.AggVote.SignBytes(), msg.AggVote.SignerIndexer.Byte()) {
		//return fmt.Errorf("bad Signature")
		//}
		if has, err := state.store.HasSpecial([]byte(key)); err != nil {
			return err
		} else if has {
			return fmt.Errorf("duplicated CrossShardMessage")
		} else if pb := msg.ToProto(); pb == nil {
			return fmt.Errorf("unmarshal CrossShardMessage Error")
		} else if err := state.store.SetSpecial([]byte(key), types.MustProtoBytes(pb)); err != nil {
			return err
		} else if _, ok := state.block_data.finished[msg.SourceChain]; !ok {
			if bytes.Equal(state.block_data.lastHash[msg.SourceChain], msg.GetLastHash()) {
				state.extendHash(msg)
				//fmt.Println("Received CrossShardMessage:", msg.SourceChain, "Status:", len(state.block_data.finished))
			}
		}
	case *hotstuff.Vote:
		if err := state.HotStuffState.AddVote(msg); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown message type: %s", msg)
	}
	return state.handle_state_transition()
}

func (state *State) extendHash(msg *types.CrossShardMessage) {
	state.block_data.finished[msg.SourceChain] = msg
	state.block_data.lastHash[msg.SourceChain] = msg.AggVote.ForHash
	state.WriteCmd(fmt.Sprintf("Extend Hash(%s): %s -> %s", msg.SourceChain, hex.EncodeToString(types.GetLastHashOfAggVote(msg.AggVote)), hex.EncodeToString(msg.AggVote.ForHash)))
}

func (state *State) redo_CrossShardMessage() error {
	for shard, hash := range state.block_data.lastHash {
		if shard == state.chain_id {
			continue
		}
		key := fmt.Sprintf("%s:%s", shard, hash)
		if bz, err := state.store.GetSpecial([]byte(key)); err != nil {
			return err
		} else if len(bz) == 0 {
			continue
		} else if csm, err := types.NewCrossShardMessageFromBytes(bz); err != nil {
			return err
		} else {
			state.WriteCmd(fmt.Sprintf("Redo CSM: %s", csm.SourceChain))
			state.extendHash(csm)
		}
	}
	return nil
}
