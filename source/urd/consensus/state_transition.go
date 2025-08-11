package consensus

import (
	"bytes"
	"emulator/core/hotstuff"
	"emulator/urd/consensus/constypes"
	"emulator/urd/definition"
	"emulator/urd/types"
	"emulator/utils/signer"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	STEP_VALIDATOR   = "validator-wait"
	STEP_LEADER_VOTE = "leader-vote"
	STEP_LEADER_WAIT = "leader-wait"
)

func (state *State) isProposer() bool {
	return state.proposerIndex == state.signerIndex
}
func (state *State) next_step() string {
	switch {
	case !state.isProposer():
		return STEP_VALIDATOR
	case state.isProposer() && state.HotStuffState.IsQuorum():
		return STEP_LEADER_WAIT
	case state.isProposer() && !state.HotStuffState.IsQuorum():
		return STEP_LEADER_VOTE
	default:
		return "UNKNOWN STEP"
	}
}

func (state *State) handle_state_transition() error {
	view, round := state.HotStuffState.View, state.HotStuffState.Round
	switch state.step {
	case STEP_VALIDATOR:
		// as a validator
		// if it has got the block and proposal
		// validate the block
		// execution block in the last round
		if state.block_data.isComplete(view, round) {
			fmt.Println("valid block", view, round)
			if block, err := state.block_data.getBlock(view, round); err != nil {
				return err
			} else if err := state.doValidate(block); err != nil {
				return err
			}
		} else {
			return nil
		}
	case STEP_LEADER_WAIT:
		// as a leader and has collected a Quorum
		// wait for CrossShardMessage
		if state.block_data.isCrossShardMessageComplete() {
			if err := state.doPropose(); err != nil {
				return err
			}
		} else {
			return nil
		}
	case STEP_LEADER_VOTE:
		// as a leader and is waiting for a Quorum
		// wait for HotStuff Votes
		if state.HotStuffState.IsQuorum() {
			if err := state.CrossShardCommunicate(); err != nil {
				return err
			}
		} else {
			return nil
		}
	}

	state.step = state.next_step()

	if state.isProposer() && (state.EnablePipelineFlag && state.HotStuffState.View > 100 || state.HotStuffState.View > 600) {
		state.Stop()
		panic("Interupted")
	}
	return state.handle_state_transition()
}

func (state *State) verify_block(block *types.Block) error {
	state.WriteLogger(fmt.Sprintf("validate block"), false, false)
	state.WriteCmd(fmt.Sprintf("validate block: (view=%d,round=%d", state.HotStuffState.View, state.HotStuffState.Round))
	if block.View < 5 {
		return nil
	}
	// 1. validate aggregated vote
	//shard := state.shard_info.Shards[state.chain_id]
	aggSig := block.AggSigVote
	/*
		if ok := shard.VerifyAggregateSignature(string(aggSig.Sign), aggSig.SignBytes(), aggSig.SignerIndexer.Byte()); !ok {
			return fmt.Errorf("error: invalid block AggSig")
		}
	*/
	if !bytes.Equal(state.fetch_block(2).Hash(), types.GetLastHashOfAggVote(aggSig)) {
		return fmt.Errorf("error: hash of j-2 block header does not comsistent")
	}
	if !bytes.Equal(state.fetch_block(1).Hash(), aggSig.ForHash) {
		return fmt.Errorf("error: hash of j-1 block header does not comsistent")
	}
	state.WriteCmd("valid aggregated signature")

	// 2. validate commitment intention
	ci := block.CI
	if len(ci.IntentionHash) != len(state.shard_info.ShardIDList) {
		return fmt.Errorf("error: not enough intention hash")
	}
	for _, id := range state.shard_info.ShardIDList {
		if id == state.chain_id {
			continue
		}
		//aggSig := ci.AggregatedSignatures[i]
		//shard := state.shard_info.Shards[id]
		/*
			if ok := shard.VerifyAggregateSignature(string(aggSig.Sign), aggSig.SignBytes(), aggSig.SignerIndexer.Byte()); !ok {
				return fmt.Errorf("error: invalid block CI")
			}
		*/

	}
	state.WriteCmd("valid Commitment Intention")

	// 3. validate commitment certificate
	cc := block.CC
	if len(cc) != len(state.shard_info.ShardIDList) {
		return fmt.Errorf("error: not enough commitment certificate core")
	}

	// 4. validate the rest of blocks
	// this part has been done when getBlock
	/*
		if err := block.ValidateBasic(); err != nil {
			return err
		}
	*/
	return nil
}

func generate_vote_for_block(block *types.Block, lastHash []byte, blockErr error, index int, sr *signer.Signer) (*hotstuff.Vote, error) {
	vote := hotstuff.NewVote(block.View, block.Round, block.Hash(), nil, index)
	if blockErr == nil {
		vote.SetOK()
	} else {
		vote.SetReject()
	}
	types.SetLastHashOfVote(vote, lastHash)
	if signStr, err := sr.SignType(vote); err != nil {
		return nil, err
	} else {
		vote.Sign = signStr
	}
	return vote, nil
}

func (state *State) doValidate(block *types.Block) error {
	blockErr := state.verify_block(block)
	if blockErr != nil {
		state.WriteCmd(fmt.Sprintf("block validation failed: %s", blockErr))
	}
	// block of j-1
	lastBlock := state.fetch_block(1)
	lastHash := lastBlock.Hash()

	vote, err := generate_vote_for_block(block, lastHash, blockErr, state.signerIndex, state.signer)
	if err != nil {
		return err
	}

	if vote.IsOK() {
		if _, err := state.commit_and_execution_j_2(); err != nil {
			return err
		}
		state.enterNextView()
		state.append_block(block)
	}

	state.bytesLock.Lock()
	state.intra_shard_bytes += len(vote.ProtoBytes())
	state.bytesLock.Unlock()

	state.SendTo(state.chain_id, state.proposerIndex, vote.ProtoBytes(), definition.Vote)

	return nil
}

func (state *State) commit_and_execution_j_2() (types.ABCIExecutionResponse, error) {
	if block_j_2 := state.fetch_block(2); block_j_2 != nil {
		// execution TXs of voting round j-2
		// execution CTXs of voting round j-6, whose merkle root is included in block j-2 as a Commitment Certificate
		state.WriteCmd(fmt.Sprintf("start to execute block for view %d", block_j_2.View))
		resp := state.abci.Execution(block_j_2.PTXS, block_j_2.CrossShardTxs, block_j_2.CTXS)
		state.WriteLogger(fmt.Sprintf("finish[%d,%d,%d]", block_j_2.PTXS.Size(), block_j_2.CrossShardTxs.Size()/2, block_j_2.CrossShardTxs.Size()/2), false, true)
		return *resp, nil
	}
	return types.ABCIExecutionResponse{}, nil
}

func (state *State) enterNextView() {
	state.block_data.next(state.HotStuffState.View+1, 0)
	state.HotStuffState.EnterNewView()
	state.WriteLogger("START", true, false)
}

func (state *State) doPropose() error {
	resp, err := state.commit_and_execution_j_2()
	if err != nil {
		return err
	}
	state.enterNextView()

	if state.HotStuffState.View == 1 {
		time.Sleep(10 * time.Second)
	}

	new_block := state.make_block(resp)
	partset, err := state.block_data.addBlock(new_block, 40960, state.HotStuffState.Round)
	if err != nil {
		return err
	}
	state.HotStuffState.SetHash(new_block.Hash())

	proposal := constypes.NewProposal(partset.Header, state.signerIndex, partset.BlockHeaderHash)
	sig, err := state.signer.SignType(proposal)
	if err != nil {
		panic(err)
	}
	proposal.Signature = sig
	go func() {
		//time.Sleep(interval_dst.Sub(time.Now()))
		state.bytesLock.Lock()
		defer state.bytesLock.Unlock()
		proposalBz := proposal.ProtoBytes()
		coo_bytes := len(types.MustProtoBytes(new_block.CI.ToProto())) + len(types.MustProtoBytes(new_block.CC.ToProto()))
		cross_shard_bytes := 0
		for i := range new_block.CTXS {
			cross_shard_bytes += len(types.MustProtoBytes(new_block.CTXS[i].ToProto())) + len(types.MustProtoBytes(new_block.CTXSProof[i].ToProto()))
		}
		state.cross_shard_data_bytes += cross_shard_bytes
		state.intra_shard_bytes += len(proposalBz) - cross_shard_bytes - coo_bytes
		state.cooperation_bytes += coo_bytes
		state.SendToShard(state.chain_id, proposalBz, definition.Proposal)
		for _, part := range partset.Parts {
			state.SendToShard(state.chain_id, part.ProtoBytes(), definition.Part)
			state.intra_shard_bytes += len(part.ProtoBytes())
		}
	}()

	vote, err := generate_vote_for_block(new_block, new_block.HashPointer, nil, state.signerIndex, state.signer)
	if err != nil {
		return err
	}
	if err := state.HotStuffState.AddVote(vote); err != nil {
		return err
	}

	state.append_block(new_block)
	// remove txs from mempool
	state.mempool.Update(new_block.PTXS, nil)
	state.cross_shard_mempool.Update(new_block.CrossShardTxs, nil)
	return nil
}

func (state *State) make_block(execution_result types.ABCIExecutionResponse) *types.Block {
	state.WriteCmd(fmt.Sprintf("start to generate a block for view %d", state.HotStuffState.View))
	var block types.Block
	lastBlock := state.fetch_block(1)
	lastHash := lastBlock.Hash()

	// Header
	block.Header = types.Header{
		HashPointer: lastHash,

		ChainID: state.chain_id,
		View:    state.HotStuffState.View,
		Round:   state.HotStuffState.Round,
		Time:    time.Now(),
	}

	if state.block_pool_size() >= 1 {
		// AggSig
		block.AggSigVote = state.block_data.j_1finished[state.chain_id].AggVote
	}

	state.WriteCmd(fmt.Sprintf("max_intra_shard: %d KB | max_cross_shard: %d KB", state.max_bytes/1024, state.max_cross_shard_bytes/1024))
	// PTXS
	if state.EnablePipelineFlag || state.HotStuffState.View%6 == 0 {
		if txs, _, err := state.mempool.ReapTx(state.max_bytes); err != nil {
			return nil
		} else {
			block.PTXS = txs
		}
		if txs, _, err := state.cross_shard_mempool.ReapTx(state.max_cross_shard_bytes); err != nil {
			return nil
		} else {
			block.CrossShardTxs = txs
		}
	}

	// OPT
	for _, respTxs := range execution_result.OPTxs {
		block.OPT = append(block.OPT, respTxs.Hash())
	}
	state.block_data.j_1_cross_shard_txs = execution_result.OPTxs

	if state.block_pool_size() >= 2 {
		// CI
		var ci types.CommitIntention
		for _, chain_id := range state.shard_info.ShardIDList {
			csm := state.block_data.j_1finished[chain_id]
			ci.AggregatedSignatures = append(ci.AggregatedSignatures, csm.AggVote)
			ci.IntentionHash = append(ci.IntentionHash, csm.GetLastHash())
		}
		block.CI = &ci
	}

	if state.block_pool_size() >= 4 {
		// CC
		var cc types.CommitCertificate
		for _, chain_id := range state.shard_info.ShardIDList {
			csm := state.block_data.j_1finished[chain_id]
			var core types.CommitCertificateCore
			core.Hash = csm.GetLastHash()
			core.MKProof = *csm.ProofOfIntention
			core.IntentionBrief = csm.IntentionBrief
			cc = append(cc, &core)
		}
		block.CC = cc
	}

	if state.block_pool_size() >= 4 {
		// CTXs and its proof
		var ctxs []types.Txs
		var ctxs_proof []*types.OutputTxsProof
		for _, chain_id := range state.shard_info.ShardIDList {
			csm := state.block_data.j_1finished[chain_id]
			ctxs = append(ctxs, csm.OPTXs)
			ctxs_proof = append(ctxs_proof, csm.OutputTxsProof)
		}
		block.CTXS = ctxs
		block.CTXSProof = ctxs_proof
	}

	// must call block.Hash() to ensure to fill in all the hashes
	state.WriteCmd(fmt.Sprintf("successfully generate a block(hash=%s,itxs=%d,ctxs=%d)", hex.EncodeToString(block.Hash()), block.PTXS.Size(), block.CrossShardTxs.Size()))
	return &block
}
func (state *State) CrossShardCommunicate() error {
	if err := state.redo_CrossShardMessage(); err != nil {
		return err
	}
	sig, err := state.HotStuffState.GetMaj23()
	if err != nil {
		return err
	}
	/*
		if !state.verifier.VerifyAggregateSignature(string(sig.GetSign()), sig.SignBytes(), sig.SignerIndexer.Byte()) {
			panic(errors.New("invalid signature"))
		}
	*/
	var model_csm types.CrossShardMessage = types.CrossShardMessage{
		SourceChain: state.chain_id,
		AggVote:     sig,
	}
	var block_j_2 *types.Block
	if state.block_pool_size() >= 4 {
		block_j_2 = state.fetch_block(2)
		model_csm.IntentionBrief = block_j_2.CI.GetBrief()
		model_csm.ProofOfIntention = &types.ProofOfIntention{
			IntentionHashProof: block_j_2.GetCommitIntentionProof(),
			RightHash:          block_j_2.CI.RrightHash(),
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(state.shard_info.ShardIDList))

	broadcastFunc := func(index int) {
		csm := types.CrossShardMessage{
			SourceChain:      model_csm.SourceChain,
			AggVote:          model_csm.AggVote,
			IntentionBrief:   model_csm.IntentionBrief,
			ProofOfIntention: model_csm.ProofOfIntention,
		}
		if state.block_pool_size() >= 4 {
			txs := state.block_data.j_2_cross_shard_txs[index]
			proof := block_j_2.GetOutputTxsProofOf(index)
			csm.OPTXs = txs
			csm.OutputTxsProof = proof
		}
		if id := state.shard_info.ShardIDList[index]; id == state.chain_id {
			state.extendHash(&csm)
		} else {
			bz := csm.ProtoBytes()
			state.bytesLock.Lock()
			cs_bytes := len(types.MustProtoBytes(csm.OPTXs.ToProto()))
			coo_byte := len(bz) - cs_bytes
			fmt.Println("Broadcasting CrossShardMessage to shard", id, "with bytes:", len(bz), "cross_shard_data_bytes:", cs_bytes, "cooperation_bytes:", coo_byte)
			fmt.Println("CrossShardMessage OPTXs size:", csm.OPTXs.Size())
			state.cross_shard_data_bytes += cs_bytes
			state.cooperation_bytes += coo_byte
			state.bytesLock.Unlock()
			state.SendTo(id, state.shard_info.Shards[id].LeaderIndex, bz, definition.CrossShardMessage)
		}
		wg.Done()
	}

	for i := range state.shard_info.ShardIDList {
		go broadcastFunc(i)
	}
	wg.Wait()

	return nil
}
