package tendermint

import (
	"bytes"
	"emulator/crypto/merkle"
	blocklogger "emulator/logger/blocklogger"
	constypes "emulator/rapid/consensus/constypes"
	definition "emulator/rapid/definition"
	inter "emulator/rapid/definition"
	"emulator/rapid/shardinfo"
	types "emulator/rapid/types"
	p2p "emulator/utils/p2p"
	sig "emulator/utils/signer"
	store "emulator/utils/store"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type ConsensusState struct {
	Height int64 `json:"height"`
	Round  int   `json:"round"`
	Step   int8  `json:"step"`

	mempool             inter.MempoolConn
	cross_shard_mempool inter.MempoolConn
	abci                inter.ABCIConn
	p2p                 *p2p.Sender
	store               *store.PrefixStore

	signer      *sig.Signer
	signerIndex int

	heightDatas         *HeightDataPackage
	relayMessageChannel map[int64]map[int][]interface{}

	MinBlockInterval time.Duration
	LastBlockTime    time.Time
	MaxPartSize      int
	maxBlockTxNum    int

	LastBlockHash   []byte
	LastReceiptRoot []byte
	LastStateRoot   []byte

	BlockHash []byte

	logger blocklogger.BlockWriter

	stateMtx sync.Mutex

	crossShardBlockPool map[string]*constypes.CrossShardBlock
	commitPool          map[string]*constypes.MessageCommit
	committed           map[string]bool
	bcount              int
	bseen               map[string]bool

	pendingCrossShardBlockHash []byte
	pendingBlockSize           int
	maCount                    int
	maPool                     map[string]*constypes.MessageAccept
	shardHeight                map[string]int64
}

func NewConsensusState(chain_id string, si *shardinfo.ShardInfo,
	signer *sig.Signer, signerIndex int,
	mempool inter.MempoolConn, cross_shard_mempool inter.MempoolConn,
	abci inter.ABCIConn, sender *p2p.Sender,
	storeDir string, minBlockInterval time.Duration, maxPartSize int, maxBlockTxNum int,
	logger blocklogger.BlockWriter) *ConsensusState {
	cs := &ConsensusState{
		Height: 0,
		Round:  0,
		Step:   RoundStepNewHeight,

		mempool:             mempool,
		cross_shard_mempool: cross_shard_mempool,
		abci:                abci,
		p2p:                 sender,
		store:               store.NewPrefixStore("consensus", storeDir),

		signer:      signer,
		signerIndex: signerIndex,

		heightDatas:         NewHeightData(chain_id, 0, si, sender),
		relayMessageChannel: make(map[int64]map[int][]interface{}),

		MinBlockInterval: minBlockInterval,
		LastBlockTime:    time.Now(),
		MaxPartSize:      maxPartSize,
		maxBlockTxNum:    maxBlockTxNum,

		LastBlockHash: []byte{},

		LastReceiptRoot: []byte{},
		LastStateRoot:   []byte{},

		BlockHash: []byte{},

		logger: logger,

		stateMtx: sync.Mutex{},

		maCount: -1,

		crossShardBlockPool: make(map[string]*constypes.CrossShardBlock),
		commitPool:          make(map[string]*constypes.MessageCommit),
		committed:           make(map[string]bool),

		shardHeight: make(map[string]int64),
	}
	cs.abci.SetBlockStore(cs.store)
	return cs
}

func (cs *ConsensusState) Start() {
	cs.stateMtx.Lock()
	defer cs.stateMtx.Unlock()
	if cs.Height == 0 {
		cs.enterNewHeight()
	} else {
		cs.handleStateTransition()
	}
}
func (cs *ConsensusState) Stop() {
	cs.store.Close()
}
func (cs *ConsensusState) WriteLogger(msg string, is_start, is_end bool) {
	cs.logger.Write(blocklogger.NewConsensusEvent(cs.Height, int32(cs.Round), RoundStepString(cs.Step), is_start, is_end, msg))
}

func (cs *ConsensusState) IsIShard() bool { return cs.heightDatas.IsIShard() }
func (cs *ConsensusState) IsBShard() bool { return cs.heightDatas.IsBShard() }

func (cs *ConsensusState) addRelayMessageIfOverTime(height int64, round int, msg interface{}) bool {
	if cs.Height < height || cs.Height == height && cs.Round < round {
		if u, ok := cs.relayMessageChannel[height]; ok {
			u[round] = append(u[round], msg)
			return true
		} else {
			cs.relayMessageChannel[height] = map[int][]interface{}{round: {msg}}
		}
	}
	return false
}

func (cs *ConsensusState) Next() {
	if cs.Step == RoundStepCommit {
		delete(cs.relayMessageChannel, cs.Height)
		cs.Round = 0
		cs.Height++
		cs.Step = RoundStepNewHeight
	} else {
		cs.Step++
	}
}

func (cs *ConsensusState) Receive(channel_id byte, bz []byte, messageType uint32) error {
	switch messageType {
	case definition.Part:
		part := types.NewPartFromBytes(bz)
		if part == nil {
			return fmt.Errorf("Part Unmarshal Error")
		}
		if err := part.ValidateBasic(); err != nil {
			return err
		}
		cs.stateMtx.Lock()
		fmt.Printf("%v (height = % d, step = % s) received Part (height = % d, chain = % s, index = % d)\n", time.Now(), cs.Height, RoundStepString(cs.Step), part.Height, part.ChainID, part.Index())
		defer cs.stateMtx.Unlock()
		err := cs.doMessage(part)
		return err
	case definition.TendermintProposal:
		proposal := constypes.NewProposalFromBytes(bz)
		if proposal == nil {
			return fmt.Errorf("Proposal Unmarshal Error")
		}

		if err := proposal.ValidateBasic(); err != nil {
			return err
		}
		cs.stateMtx.Lock()
		fmt.Printf("%v (height=%d,step=%s) Proposal received (height=%d,chain=%s,index=%d)\n", time.Now(), cs.Height, RoundStepString(cs.Step), proposal.Header.Height, proposal.Header.ChainID, proposal.ProposerIndex)
		defer cs.stateMtx.Unlock()
		err := cs.doMessage(proposal)
		return err
	case definition.TendermintPrevote:
		prevote := constypes.NewPrevoteFromBytes(bz)
		if prevote == nil {
			return fmt.Errorf("Prevote Unmarshal Error")
		}

		if err := prevote.ValidateBasic(); err != nil {
			return err
		}
		cs.stateMtx.Lock()
		fmt.Printf("%v (height=%d,step=%s) received  Prevote(height=%d,index=%d)\n", time.Now(), cs.Height, RoundStepString(cs.Step), prevote.Height, prevote.ValidatorIndex)
		defer cs.stateMtx.Unlock()
		err := cs.doMessage(prevote)
		return err
	case definition.TendermintPrecommit:
		precommit := constypes.NewPrecommitFromBytes(bz)
		if precommit == nil {
			return fmt.Errorf("Precommit Unmarshal Error")
		}

		if err := precommit.ValidateBasic(); err != nil {
			return err
		}
		cs.stateMtx.Lock()
		fmt.Printf("%v (height=%d,step=%s) received Precommit(height=%d,index=%d)\n", time.Now(), cs.Height, RoundStepString(cs.Step), precommit.Height, precommit.ValidatorIndex)
		defer cs.stateMtx.Unlock()
		err := cs.doMessage(precommit)
		return err
	case definition.CrossShardBlock:
		csb, err := constypes.NewCrossShardBlockFromBytes(bz)
		if err != nil {
			return fmt.Errorf("CSB Unmarshal Error: " + err.Error())
		}
		if err := csb.Block.ValidateBasic(); err != nil {
			return err
		}
		if err := csb.ColleciveSignatures.ValidateBasic(); err != nil {
			return err
		}

		cs.stateMtx.Lock()
		fmt.Printf("%v (height=%d,step=%s) received CrossShardBlock(height=%d,type=%d)\n", time.Now(), cs.Height, RoundStepString(cs.Step), csb.Block.Height, csb.Block.BlockType)
		defer cs.stateMtx.Unlock()
		err = cs.doMessage(csb)
		return err
	case definition.MessageAccept:
		ma, err := constypes.NewMessageAcceptFromBytes(bz)
		if err != nil {
			return fmt.Errorf("MA Unmarshal Error: " + err.Error())
		}
		if err := ma.CollectiveSignatures.ValidateBasic(); err != nil {
			return err
		}

		cs.stateMtx.Lock()
		fmt.Printf("%v (height=%d,step=%s) MessageAccept(height=%d,code=%d)\n", time.Now(), cs.Height, RoundStepString(cs.Step), ma.CollectiveSignatures.Header.Height, ma.CollectiveSignatures.Code)
		defer cs.stateMtx.Unlock()
		err = cs.doMessage(ma)
		return err
	case definition.MessageCommit:
		mc, err := constypes.NewMessageCommitFromBytes(bz)
		if err != nil {
			return fmt.Errorf("MC Unmarshal Error: " + err.Error())
		}
		if err := mc.CollectiveSignatures.ValidateBasic(); err != nil {
			return err
		}

		cs.stateMtx.Lock()
		fmt.Printf("%v (height=%d,step=%s) MessageCommit(height=%d,code=%d,chain=%s) received\n", time.Now(), cs.Height, RoundStepString(cs.Step), mc.CollectiveSignatures.Header.Height, mc.CollectiveSignatures.Code, mc.CollectiveSignatures.Header.ChainID)
		defer cs.stateMtx.Unlock()
		err = cs.doMessage(mc)
		return err
	case definition.MessageOK:
		mo, err := constypes.NewMessageOKFromBytes(bz)
		if err != nil {
			return fmt.Errorf("MO Unmarshal Error: " + err.Error())
		}
		cs.stateMtx.Lock()
		fmt.Printf("%v (height=%d,step=%s) Received MessageOK(height=%d,chain=%s)\n", time.Now(), cs.Height, RoundStepString(cs.Step), mo.Height, mo.SrcChain)
		defer cs.stateMtx.Unlock()
		err = cs.doMessage(mo)
		return err
	default:
		return fmt.Errorf("Tendermint. ConsensusState: without recognition of the message type (" + fmt.Sprint(messageType) + ")")
	}
}

func (cs *ConsensusState) doMessage(msg interface{}) error {
	switch m := msg.(type) {
	case *types.Part:
		if cs.addRelayMessageIfOverTime(m.Height, m.Round, m) {
			return nil
		}
		if err := cs.heightDatas.AddPart(m); err != nil {
			if err == DoNothing {
				return nil
			}
			return err
		}
		return cs.handleStateTransition()
	case *constypes.Proposal:
		if cs.addRelayMessageIfOverTime(m.Header.Height, m.Header.Round, m) {
			return nil
		}
		if err := cs.heightDatas.AddProposal(m, cs.calculateProposer()); err != nil {
			if err == DoNothing {
				return nil
			}
			return err
		}
		err := cs.handleStateTransition()
		return err
	case *constypes.Prevote:
		if cs.addRelayMessageIfOverTime(m.Height, m.Round, m) {
			return nil
		}
		if err := cs.heightDatas.AddPrevote(m); err != nil {
			if err == DoNothing {
				return nil
			}
			return err
		}
		err := cs.handleStateTransition()
		return err
	case *constypes.Precommit:
		if cs.addRelayMessageIfOverTime(m.Height, m.Round, m) {
			return nil
		}
		if err := cs.heightDatas.AddPrecommit(m); err != nil {
			if err == DoNothing {
				return nil
			}
			return err
		}
		err := cs.handleStateTransition()
		return err
	case *constypes.CrossShardBlock:
		chain_id := m.ColleciveSignatures.Header.ChainID
		if !cs.heightDatas.ShardInfo.RelatedShards[chain_id] {
			return fmt.Errorf("A broadcast from a non-adjacent shard has been received")
		}
		if has, _ := cs.store.Has(m.Block.Hash()); has {
			return nil
		}
		if !m.ColleciveSignatures.VerifySignatures(cs.heightDatas.AllValidatorSet[chain_id], m.ColleciveSignatures.ValidatorBitVector.Byte()) {
			return fmt.Errorf("CrossShardBlock signature verification failed")
		}
		if cs.IsIShard() {
			cs.crossShardBlockPool[string(m.Block.Hash())] = m
		} else {
			cs.store.SetBlockByHeight(m.ColleciveSignatures.Header.Height, chain_id, m.Block)
			i0 := cs.shardHeight[chain_id] + 1
			for {
				if blockBz, err := cs.store.GetBlockByHeight(i0, chain_id); err != nil || blockBz == nil {
					break
				} else {
					iblock := types.NewBlockFromBytes(blockBz)
					_, c1, c2, c3 := cs.processIBlock(iblock, chain_id, false, m.ColleciveSignatures.IsOK())
					log.Printf("B shard synchronizes I shard state: %d intra-shard transaction,%d cross-shard transaction, commit %d\n", c1, c3, c2)
				}
				cs.shardHeight[chain_id] = i0
				i0++
			}
		}
		err := cs.handleStateTransition()
		return err
	case *constypes.MessageAccept:
		chain_id := m.CollectiveSignatures.Header.ChainID
		if cs.IsIShard() {
			return fmt.Errorf("The I shard received an Accept message")
		}
		if !bytes.Equal(m.B_BlockHash, cs.pendingCrossShardBlockHash) {
			return nil
		}
		if !m.CollectiveSignatures.VerifySignatures(cs.heightDatas.AllValidatorSet[chain_id], m.CollectiveSignatures.ValidatorBitVector.Byte()) {
			return fmt.Errorf("MessageAccept signature verification failed")
		}
		if _, ok := cs.maPool[chain_id]; ok {
			return nil
		}
		cs.maCount--
		cs.maPool[chain_id] = m
		err := cs.handleStateTransition()
		return err
	case *constypes.MessageCommit:
		chain_id := m.CollectiveSignatures.Header.ChainID
		if cs.IsBShard() {
			return fmt.Errorf("Shard B received a Commit message")
		}
		if cs.committed[string(m.B_BlockHash)] {
			return nil
		}
		if !m.CollectiveSignatures.VerifySignatures(cs.heightDatas.AllValidatorSet[chain_id], m.CollectiveSignatures.ValidatorBitVector.Byte()) {
			return fmt.Errorf("MessageCommit signature verification failed")
		}
		cs.commitPool[string(m.B_BlockHash)] = m
		err := cs.handleStateTransition()
		return err
	case *constypes.MessageOK:
		if cs.addRelayMessageIfOverTime(m.Height, 0, m) {
			return nil
		}
		if m.Height < cs.Height {
			return nil
		}
		if cs.bseen[m.SrcChain] {
			return nil
		}
		cs.bseen[m.SrcChain] = true
		cs.bcount--
		err := cs.handleStateTransition()
		return err
	default:
		return fmt.Errorf("unkonwn type")
	}
}

func (cs *ConsensusState) handleStateTransition() error {
	switch cs.Step {
	case RoundStepPropose:
		if cs.heightDatas.FinishProposal() {
			cs.WriteLogger("Finish Proposal", false, false)
			cs.doPropose()
			cs.WriteLogger("Enter Prevote", false, false)
		} else {
			fmt.Println("I didn't finish the Propose phase")
			return nil
		}
	case RoundStepPrevote:
		if cs.heightDatas.FinishPrevote() {
			cs.WriteLogger("Finish Prevote", false, false)
			cs.doPrevote()
			cs.WriteLogger("Enter Precommit", false, false)
		} else {
			return nil
		}
	case RoundStepPrecommit:
		if cs.heightDatas.FinishPrecommit() {
			cs.WriteLogger("Finish Precommit", false, false)
			cs.doPrecommit()
		} else {
			return nil
		}
	case RoundStepCommit:
		if cs.IsIShard() {
			if cs.bcount == 0 {
				cs.doCommit()
			} else {
				return nil
			}
		} else {
			for cid := range cs.heightDatas.ShardInfo.RelatedShards {
				fmt.Println(cid)
				height := cs.shardHeight[cid]
				if height < cs.Height {
					return nil
				}
			}
			cs.doCommit()
		}
	default:
		return nil
	}

	return cs.handleStateTransition()
}

func (cs *ConsensusState) calculateProposer() int {
	return (int(cs.Height) + cs.Round) % cs.heightDatas.AllValidatorSet[cs.heightDatas.MyChainID].Size()
}
func (cs *ConsensusState) isProposer() bool {
	return cs.calculateProposer() == cs.signerIndex
}

func (cs *ConsensusState) doPropose() {
	proposalBlock := cs.heightDatas.Block
	cs.BlockHash = proposalBlock.Hash()
	flag := true
	pvote := true
	if cs.IsBShard() {
		switch proposalBlock.BlockType {
		case types.BLOCKTYPE_InnerShard:
			flag = true
			fmt.Println("BlockType Inner")
		case types.BLOCKTYPE_CrossShard:
			resp := cs.abci.PreExecutionB(proposalBlock)
			flag = resp.IsOK() //&& utils.CompareBytesList(resp.CrossShardDatas, proposalBlock.CrossShardDatas)
			fmt.Println("BlockType Cross")
		case types.BLOCKTYPE_BCommitBlock:
			fmt.Println("BlockType Commit")
			if proposalBlock.BodyTxs.Size() != 1 {
				flag = false
			} else if ba, err := constypes.NewMessageAcceptSetFromBytes(proposalBlock.BodyTxs[0]); err != nil {
				flag = false
			} else if !bytes.Equal(cs.pendingCrossShardBlockHash, ba.BlockHash) {
				flag = false
			} else if len(ba.Accepts) != len(cs.heightDatas.ShardInfo.RelatedShards) {
				flag = false
			} else {
				for _, ma := range ba.Accepts {
					if !cs.heightDatas.ShardInfo.RelatedShards[ma.CollectiveSignatures.Header.ChainID] {
						flag = false
						break
					}
					if !ma.CollectiveSignatures.VerifySignatures(cs.heightDatas.AllValidatorSet[ma.CollectiveSignatures.Header.ChainID], ma.CollectiveSignatures.ValidatorBitVector.Byte()) {
						flag = false
						break
					}
					if !ma.CollectiveSignatures.IsOK() {
						pvote = false
					}
				}
			}
		default:
			flag = false
		}

	} else {
		switch proposalBlock.BlockType {
		case types.BLOCKTYPE_InnerShard:
			flag = true
		case types.BLOCKTYPE_IAcceptBlock:
			if proposalBlock.BodyTxs.Size() != 1 {
				flag = false
			} else if block, err := constypes.NewCrossShardBlockFromBytes(proposalBlock.BodyTxs[0]); err != nil {
				flag = false
			} else if sig := block.ColleciveSignatures; !cs.heightDatas.ShardInfo.RelatedShards[sig.Header.ChainID] {
				flag = false
			} else if !sig.VerifySignatures(cs.heightDatas.AllValidatorSet[sig.Header.ChainID], sig.ValidatorBitVector.Byte()) {
				flag = false
			} else {
				resp := cs.abci.PreExecutionI(block.Block)
				pvote = resp.IsOK()
			}
		case types.BLOCKTYPE_ICommitBlock:
			for _, tx := range proposalBlock.BodyTxs {
				if mc, err := constypes.NewMessageCommitFromBytes(tx); err != nil {
					flag = false
					break
				} else if !cs.heightDatas.ShardInfo.RelatedShards[mc.CollectiveSignatures.Header.ChainID] {
					flag = false
					break
				} else if !mc.CollectiveSignatures.VerifySignatures(cs.heightDatas.AllValidatorSet[mc.CollectiveSignatures.Header.ChainID], mc.CollectiveSignatures.ValidatorBitVector.Byte()) {
					flag = false
					break
				}
			}
		default:
			flag = false
		}
	}
	if !flag {

		panic("Byzantine error")
	}
	vote := constypes.NewPrevote(cs.Height, cs.Round, cs.BlockHash, cs.signerIndex)
	vote.SetOK(pvote)
	sig, err := cs.signer.SignType(vote)
	if err != nil {
		panic(err)
	}
	vote.Signature = sig
	cs.heightDatas.AddPrevote(vote)
	cs.SendInternal(vote.ProtoBytes(), inter.TendermintPrevote)
	cs.Next()
}

func (cs *ConsensusState) doPrevote() {
	bz, ok := cs.heightDatas.Prevotes.GetMaj23()
	vote := constypes.NewPrecommit(cs.Height, cs.Round, bz, cs.signerIndex)
	vote.SetOK(ok)
	sig, err := cs.signer.SignType(vote)
	if err != nil {
		panic(err)
	}
	vote.Signature = sig
	cs.heightDatas.AddPrecommit(vote)
	cs.SendInternal(vote.ProtoBytes(), inter.TendermintPrecommit)
	cs.Next()
}

func (cs *ConsensusState) doPrecommit() {
	block := cs.heightDatas.Block

	var resp *types.ABCIExecutionResponse
	var innerShardCount, crossShardCommit, crossShardCount int
	if cs.IsIShard() {
		resp, innerShardCount, crossShardCommit, crossShardCount = cs.processIBlock(block, cs.heightDatas.MyChainID, true, false)
	} else {
		resp, innerShardCount, crossShardCommit, crossShardCount = cs.processBBlock(block, cs.heightDatas.MyChainID, true)
	}

	bzz := make([][]byte, 0, len(resp.Receipts))
	for _, receipt := range resp.Receipts {
		bzz = append(bzz, receipt.ProtoBytes())
	}
	respRoot, _ := merkle.ProofsFromByteSlices(bzz)
	cs.LastReceiptRoot = respRoot
	cs.LastBlockHash = block.Hash()
	cs.LastStateRoot = cs.abci.Commit()
	cs.LastBlockTime = block.Time

	cs.store.SetBlockByHeight(block.Height, block.ChainID, block)

	if innerShardCount != 0 || crossShardCount != 0 || len(cs.pendingCrossShardBlockHash) == 0 {
		if cs.IsIShard() && innerShardCount <= 1 {
			cs.WriteLogger(fmt.Sprintf("finish[%d,%d,%d]", innerShardCount, crossShardCommit, crossShardCount), false, true)
		} else if cs.IsIShard() {
			cs.WriteLogger(fmt.Sprintf("finish[%d,%d,%d]", innerShardCount*2/3, crossShardCommit, crossShardCount), false, true)
		} else {
			cs.WriteLogger(fmt.Sprintf("finish[%d,%d,%d]", innerShardCount, crossShardCommit/2, crossShardCount/2), false, true)
		}
	} else {
		cs.WriteLogger(fmt.Sprintf("finish[%d,%d,%d]", 1, crossShardCommit, crossShardCount), false, true)
	}

	cs.Next()
}

func (cs *ConsensusState) doCommit() {
	messageOK := &constypes.MessageOK{
		Height:   cs.Height,
		SrcChain: cs.heightDatas.MyChainID,
	}
	if cs.isProposer() && cs.IsBShard() {
		for chain_id := range cs.heightDatas.ShardInfo.RelatedShards {
			fmt.Printf("Broadcast MessageOK to the %s shard\n", chain_id)
			cs.SendTo(chain_id, messageOK.ProtoBytes(), definition.MessageOK)
		}

	}
	cs.enterNewHeight()
}

func (cs *ConsensusState) updateIState(block *types.Block, cross_shard_block_hash []byte) {
	switch block.BlockType {
	case types.BLOCKTYPE_InnerShard:
		cs.mempool.Update(block.BodyTxs, nil)
	case types.BLOCKTYPE_IAcceptBlock:
		delete(cs.crossShardBlockPool, string(cross_shard_block_hash))
	case types.BLOCKTYPE_ICommitBlock:
		delete(cs.commitPool, string(cross_shard_block_hash))
		cs.committed[string(cross_shard_block_hash)] = true
	default:
		return
	}
}
func (cs *ConsensusState) updateBState(block *types.Block, cross_shard_block_hash []byte, ifcommit bool) {
	switch block.BlockType {
	case types.BLOCKTYPE_InnerShard:
		cs.mempool.Update(block.BodyTxs, nil)
	case types.BLOCKTYPE_CrossShard:
		cs.maCount = len(cs.heightDatas.ShardInfo.RelatedShards)
		cs.maPool = make(map[string]*constypes.MessageAccept)
		cs.pendingCrossShardBlockHash = cross_shard_block_hash
		cs.pendingBlockSize = block.CrossShardTxs.Size()
	case types.BLOCKTYPE_BCommitBlock:
		if ifcommit {
			ublockBz, err := cs.store.GetBlockByHash(cross_shard_block_hash)
			if err != nil {
				panic(err)
			}
			ublock := types.NewBlockFromBytes(ublockBz)
			cs.cross_shard_mempool.Update(ublock.CrossShardTxs, nil)
		}
		cs.maCount = -1
		cs.maPool = nil
		cs.pendingCrossShardBlockHash = nil
	default:
		return
	}
}

func (cs *ConsensusState) processIBlock(proposalBlock *types.Block, chain_id string, isConsensusRound bool, isok bool) (resp *types.ABCIExecutionResponse, innerShardCount int, crossShardCommit int, crossShardCount int) {
	resp = new(types.ABCIExecutionResponse)
	var aggSig *constypes.PrecommitAggregated
	if isConsensusRound {
		aggSig = cs.heightDatas.GenerateAggregatePrecommitSignature()
		go func(deliver bool, mySig *constypes.PrecommitAggregated, myBlock *types.Block) {
			if !deliver {
				return
			}
			myCsBlock := &constypes.CrossShardBlock{
				Block:               myBlock,
				ColleciveSignatures: aggSig,
			}
			cs.SendToRelatedShards(myCsBlock.ProtoBytes(), definition.CrossShardBlock)

		}(cs.isProposer() && isConsensusRound, aggSig, proposalBlock)
	}

	switch proposalBlock.BlockType {
	case types.BLOCKTYPE_InnerShard:
		log.Println("Ishard performs intra-shard blocks")
		resp, innerShardCount, crossShardCommit, crossShardCount = cs.abci.ExecutionInnerShard(proposalBlock)
		if isConsensusRound {
			cs.updateIState(proposalBlock, proposalBlock.Hash())
		}
		return
	case types.BLOCKTYPE_IAcceptBlock:
		log.Println("I shard process the Acceptblock")
		blocks, err := constypes.NewCrossShardBlockFromBytes(proposalBlock.BodyTxs[0])
		if err != nil {
			panic(err)
		}
		if isConsensusRound {
			resp, innerShardCount, crossShardCommit, crossShardCount = cs.abci.AcceptCrossShard(blocks.Block, chain_id, aggSig.IsOK())
		} else {
			resp, innerShardCount, crossShardCommit, crossShardCount = cs.abci.AcceptCrossShard(blocks.Block, chain_id, isok)
		}
		ma := &constypes.MessageAccept{
			B_BlockHash:          blocks.Block.Hash(),
			CollectiveSignatures: aggSig,
		}
		if cs.isProposer() && isConsensusRound {
			cs.SendTo(blocks.Block.ChainID, ma.ProtoBytes(), definition.MessageAccept)
		}
		if isConsensusRound {
			cs.updateIState(proposalBlock, ma.B_BlockHash)
		}
		return
	case types.BLOCKTYPE_ICommitBlock:
		log.Println("I shards process Commitblock")
		hashes, commits := [][]byte{}, []bool{}
		for _, tx := range proposalBlock.BodyTxs {
			if mc, err := constypes.NewMessageCommitFromBytes(tx); err != nil {
				panic(err)
			} else {
				hashes = append(hashes, mc.B_BlockHash)
				commits = append(commits, mc.CollectiveSignatures.IsOK())
			}
		}
		resp, innerShardCount, crossShardCommit, crossShardCount = cs.abci.AcceptCommitShard(hashes, commits, chain_id)
		if isConsensusRound {
			for _, hs := range hashes {
				cs.updateIState(proposalBlock, hs)
			}
		}
		return
	default:
		panic("Process dose not define block type")
	}
}

func (cs *ConsensusState) processBBlock(proposalBlock *types.Block, chain_id string, isConsensusRound bool) (resp *types.ABCIExecutionResponse, innerShardCount int, crossShardCommit int, crossShardCount int) {
	resp = new(types.ABCIExecutionResponse)
	switch proposalBlock.BlockType {
	case types.BLOCKTYPE_InnerShard:
		resp, innerShardCount, crossShardCommit, crossShardCount = cs.abci.ExecutionInnerShard(proposalBlock)
		if isConsensusRound {
			cs.updateBState(proposalBlock, proposalBlock.Hash(), false)
		}
		return
	case types.BLOCKTYPE_CrossShard:
		aggSig := cs.heightDatas.GenerateAggregatePrecommitSignature()
		csProposal := &constypes.CrossShardBlock{
			Block:               proposalBlock,
			ColleciveSignatures: aggSig,
		}
		if cs.isProposer() && isConsensusRound {
			cs.SendToRelatedShards(csProposal.ProtoBytes(), definition.CrossShardBlock)
		}
		resp.Receipts = []*types.ABCIExecutionReceipt{
			&types.ABCIExecutionReceipt{
				Code: types.CodeTypeOK,
			},
		}
		if isConsensusRound {
			cs.updateBState(proposalBlock, proposalBlock.Hash(), false)
		}
		innerShardCount = 1
		return
	case types.BLOCKTYPE_BCommitBlock:
		aggSig := cs.heightDatas.GenerateAggregatePrecommitSignature()
		mc := &constypes.MessageCommit{
			B_BlockHash:          cs.pendingCrossShardBlockHash,
			CollectiveSignatures: aggSig,
		}
		if cs.isProposer() && isConsensusRound {
			cs.SendToRelatedShards(mc.ProtoBytes(), definition.MessageCommit)
		}
		resp.Receipts = []*types.ABCIExecutionReceipt{
			&types.ABCIExecutionReceipt{
				Code: types.CodeTypeOK,
			},
		}
		if isConsensusRound {
			cs.updateBState(proposalBlock, mc.B_BlockHash, true)
		}
		innerShardCount = 1
		if mc.CollectiveSignatures.IsOK() {
			crossShardCommit = cs.pendingBlockSize
		} else {
			crossShardCommit = 0
		}
		crossShardCount = cs.pendingBlockSize
		cs.pendingBlockSize = 0
		return
	default:
		panic("Process does not confirm the block type")
	}
}

func (cs *ConsensusState) enterNewHeight() {
	cs.Step = RoundStepCommit
	if cs.IsIShard() {
		cs.bcount = len(cs.heightDatas.ShardInfo.RelatedShards)
		cs.bseen = map[string]bool{}
	}
	cs.Next()
	cs.WriteLogger("enter New Height", true, false)
	cs.heightDatas.NextHeight()

	cs.enterNewRound()
}
func (cs *ConsensusState) enterNewRound() {
	cs.Step = RoundStepPropose
	if cs.isProposer() {
		cs.createBlockAndBroadcast()
	}
	if relay_height, ok := cs.relayMessageChannel[cs.Height]; ok {
		if relayRound, ok := relay_height[cs.Round]; ok {
			for _, msg := range relayRound {
				cs.doMessage(msg)
			}
		}
	}
}
func (cs *ConsensusState) createBlockAndBroadcast() {
	log.Println("Start generating blocks and broadcast them")
	var block *types.Block
	if cs.IsBShard() {
		block = cs.createBBlockTxs()
	} else {
		block = cs.createIBlockTxs()
	}
	block.Header = types.Header{
		BlockType:   block.BlockType,
		HashPointer: cs.LastBlockHash,

		ChainID: cs.heightDatas.MyChainID,
		Height:  cs.Height,
		Time:    time.Now(),

		StateRoot:   cs.LastStateRoot,
		ReceiptRoot: cs.LastReceiptRoot,
	}

	interval_dst := cs.LastBlockTime.Add(cs.MinBlockInterval)
	if t := time.Now(); t.Before(interval_dst) {
		block.Header.Time = interval_dst
	} else {
		block.Header.Time = t
	}
	part_set := types.PartSetFromBlock(block, cs.MaxPartSize, cs.Round)
	proposal := constypes.NewProposal(part_set.Header, cs.signerIndex, part_set.BlockHeaderHash)
	if sig, err := cs.signer.SignType(proposal); err != nil {
		panic(err)
	} else {
		proposal.Signature = sig
	}
	log.Println("PartSize", len(part_set.Parts))
	log.Println("Proposal", *proposal)

	cs.heightDatas.Block = block
	cs.heightDatas.PartSet = part_set
	cs.heightDatas.Proposal = proposal

	go func() {
		time.Sleep(interval_dst.Sub(time.Now()))

		cs.SendInternal(proposal.ProtoBytes(), inter.TendermintProposal)
		for _, part := range part_set.Parts {
			cs.SendInternal(part.ProtoBytes(), inter.Part)
		}
	}()
}
func (cs *ConsensusState) createIBlockTxs() *types.Block {
	var block = new(types.Block)
	if b := len(cs.crossShardBlockPool); b > 0 {
		block.BlockType = types.BLOCKTYPE_IAcceptBlock
		keys := make([]string, 0, b)
		for key := range cs.crossShardBlockPool {
			keys = append(keys, key)
		}
		csb := cs.crossShardBlockPool[keys[rand.Intn(b)]]
		block.BodyTxs = append(block.BodyTxs, csb.ProtoBytes())
	} else if c := len(cs.commitPool); c > 0 {
		block.BlockType = types.BLOCKTYPE_ICommitBlock
		block.BodyTxs = make(types.Txs, 0, c)
		for _, mc := range cs.commitPool {
			block.BodyTxs = append(block.BodyTxs, mc.ProtoBytes())
		}
	} else {
		block.BlockType = types.BLOCKTYPE_InnerShard
		block.BodyTxs, _, _ = cs.mempool.ReapTx(cs.maxBlockTxNum)
	}
	return block
}
func (cs *ConsensusState) createBBlockTxs() *types.Block {
	cs.abci.StateLock()
	defer cs.abci.StateUnlock()
	var block = new(types.Block)
	if cs.maCount == -1 {
		txs, n, _ := cs.cross_shard_mempool.ReapTx(cs.maxBlockTxNum)
		if n > 0 {
			block.BlockType = types.BLOCKTYPE_CrossShard
			block.CrossShardTxs = txs
			block = cs.abci.FillData(block)
			return block
		}
	}
	if cs.maCount == 0 {
		block.BlockType = types.BLOCKTYPE_BCommitBlock
		mas := make([]*constypes.MessageAccept, 0, len(cs.maPool))
		for _, ma := range cs.maPool {
			mas = append(mas, ma)
		}
		mc := &constypes.MessageAcceptSet{
			Accepts:   mas,
			BlockHash: cs.pendingCrossShardBlockHash,
		}
		block.BodyTxs = append(block.BodyTxs, mc.ProtoBytes())
		return block
	}
	block.BlockType = types.BLOCKTYPE_InnerShard
	block.BodyTxs, _, _ = cs.mempool.ReapTx(cs.maxBlockTxNum)
	return block
}

// ============================================
func (cs *ConsensusState) SendTo(shardID string, bz []byte, messageType uint32) {
	cs.p2p.SendToShard(shardID, p2p.ChannelIDConsensusState, bz, messageType)
}
func (cs *ConsensusState) SendInternal(bz []byte, messageType uint32) {
	cs.p2p.SendToShard(cs.heightDatas.MyChainID, p2p.ChannelIDConsensusState, bz, messageType)
}
func (cs *ConsensusState) SendToRelatedShards(bz []byte, messageType uint32) {
	relatedShards := cs.heightDatas.ShardInfo.RelatedShards
	for shard := range relatedShards {
		cs.SendTo(shard, bz, messageType)
	}
}
func (cs *ConsensusState) SendToRelatedShardsExcept(bz []byte, messageType uint32, exception string) {
	relatedShards := cs.heightDatas.ShardInfo.RelatedShards
	for shard := range relatedShards {
		if shard == exception {
			continue
		}
		cs.SendTo(shard, bz, messageType)
	}
}
