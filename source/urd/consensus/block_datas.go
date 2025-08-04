package consensus

import (
	"emulator/logger/blocklogger"
	"emulator/urd/types"
	"fmt"
	"time"
)

type BlockData struct {
	retainView       int64
	retainRound      int32
	blocks           map[int64]map[int32]*types.PartSet
	retainBlockParts map[int64]map[int32][]*types.Part
	lastHash         map[string][]byte
	finished         map[string]*types.CrossShardMessage

	j_1finished         map[string]*types.CrossShardMessage
	j_2finished         map[string]*types.CrossShardMessage
	j_cross_shard_txs   []types.Txs
	j_1_cross_shard_txs []types.Txs
	j_2_cross_shard_txs []types.Txs
}

func NewBlockData(view int64, round int32, chain_id string, lastHash map[string][]byte, finished map[string]*types.CrossShardMessage) *BlockData {
	return &BlockData{
		retainView:       view,
		retainRound:      round,
		blocks:           make(map[int64]map[int32]*types.PartSet),
		retainBlockParts: make(map[int64]map[int32][]*types.Part),
		lastHash:         lastHash,
		finished:         finished,
	}
}

func (bd *BlockData) old(v int64, r int32) bool {
	return v < bd.retainView || (v == bd.retainView && r < bd.retainRound)
}

func (bd *BlockData) addPartSetHeader(header *types.PartSetHeader, headerHash []byte) error {
	if bd.old(header.View, header.Round) {
		return fmt.Errorf("PartSetHeader is old: view %d, round %d", header.View, header.Round)
	}
	// Check if the header is nil
	if bd.blocks == nil {
		bd.blocks = make(map[int64]map[int32]*types.PartSet)
	}
	if _, ok := bd.blocks[header.View]; !ok {
		bd.blocks[header.View] = make(map[int32]*types.PartSet)
	}
	if _, ok := bd.blocks[header.View][header.Round]; !ok {
		bd.blocks[header.View][header.Round] = types.NewPartSet(header, headerHash)
		// Check if there are parts arrive in advance
		if bd.retainBlockParts != nil {
			if _, ok := bd.retainBlockParts[header.View]; ok {
				if _, ok := bd.retainBlockParts[header.View][header.Round]; ok {
					for _, part := range bd.retainBlockParts[header.View][header.Round] {
						if err := bd.addPart(part); err != nil {
							fmt.Println("Error: ", err)
						}
					}
				}
			}
		}
		return nil
	} else {
		return fmt.Errorf("PartSet already exists for view %d and round %d", header.View, header.Round)
	}
}

func (bd *BlockData) addPartSet(parts *types.PartSet) error {
	if bd.old(parts.Header.View, parts.Header.Round) {
		return fmt.Errorf("PartSet is old: view %d, round %d", parts.Header.View, parts.Header.Round)
	}
	if bd.blocks == nil {
		bd.blocks = make(map[int64]map[int32]*types.PartSet)
	}
	if _, ok := bd.blocks[parts.Header.View]; !ok {
		bd.blocks[parts.Header.View] = make(map[int32]*types.PartSet)
	}
	if _, ok := bd.blocks[parts.Header.View][parts.Header.Round]; !ok {
		bd.blocks[parts.Header.View][parts.Header.Round] = parts
		return nil
	} else {
		return fmt.Errorf("PartSet already exists for view %d and round %d", parts.Header.View, parts.Header.Round)
	}
}

func (bd *BlockData) addBlock(block *types.Block, size int, round int32) (*types.PartSet, error) {
	parts := types.PartSetFromBlock(block, size, round)
	return parts, bd.addPartSet(parts)
}

func (bd *BlockData) addPart(part *types.Part) error {
	if bd.old(part.View, part.Round) {
		return fmt.Errorf("Part is old: view %d, round %d", part.View, part.Round)
	}
	if bd.blocks == nil {
		bd.append_part(part)
		return fmt.Errorf("PartSet not exists for view %d and round %d", part.View, part.Round)
	} else if _, ok := bd.blocks[part.View]; !ok {
		bd.append_part(part)
		return fmt.Errorf("PartSet not exists for view %d and round %d", part.View, part.Round)
	} else if _, ok := bd.blocks[part.View][part.Round]; !ok {
		bd.append_part(part)
		return fmt.Errorf("PartSet not exists for view %d and round %d", part.View, part.Round)
	}
	return bd.blocks[part.View][part.Round].AddPart(part)
}

func (bd *BlockData) isCrossShardMessageComplete() bool {
	return len(bd.finished) == len(bd.lastHash)
}

func (bd *BlockData) isComplete(view int64, round int32) bool {
	if bd.blocks == nil {
		return false
	} else if _, ok := bd.blocks[view]; !ok {
		return false
	} else if _, ok := bd.blocks[view][round]; !ok {
		return false
	}
	return bd.blocks[view][round].IsComplete()
}
func (bd *BlockData) getBlock(view int64, round int32) (*types.Block, error) {
	if !bd.isComplete(view, round) {
		return nil, fmt.Errorf("Block not exists for view %d and round %d", view, round)
	}
	return bd.blocks[view][round].GenBlock()
}

func (bd *BlockData) next(view int64, round int32) error {
	if err := bd.prune(view, round); err != nil {
		return err
	}
	bd.j_2finished = bd.j_1finished
	bd.j_1finished = bd.finished
	bd.finished = make(map[string]*types.CrossShardMessage)
	bd.j_2_cross_shard_txs = bd.j_1_cross_shard_txs
	bd.j_1_cross_shard_txs = bd.j_cross_shard_txs
	bd.j_cross_shard_txs = make([]types.Txs, 0)
	bd.retainView, bd.retainRound = view, round
	return nil
}
func (bd *BlockData) prune(view int64, round int32) error {
	if bd.blocks == nil {
		return nil
	}
	// delete old blocks
	for v := range bd.blocks {
		if v < view {
			delete(bd.blocks, v)
		} else if v == view {
			for r := range bd.blocks[v] {
				if r < round {
					delete(bd.blocks[v], r)
				}
			}
			if len(bd.blocks[v]) == 0 {
				delete(bd.blocks, v)
			}
		}
	}
	// delete old blockparts
	for v := range bd.retainBlockParts {
		if v < view {
			delete(bd.retainBlockParts, v)
		} else if v == view {
			for r := range bd.retainBlockParts[v] {
				if r < round {
					delete(bd.retainBlockParts[v], r)
				}
			}
			if len(bd.retainBlockParts[v]) == 0 {
				delete(bd.retainBlockParts, v)
			}
		}
	}
	return nil
}

func (bd *BlockData) append_part(part *types.Part) {
	if bd.old(part.View, part.Round) {
		return
	}
	// Check if the header is nil
	if bd.retainBlockParts == nil {
		bd.retainBlockParts = make(map[int64]map[int32][]*types.Part)
	}
	if _, ok := bd.retainBlockParts[part.View]; !ok {
		bd.retainBlockParts[part.View] = make(map[int32][]*types.Part)
	}
	bd.retainBlockParts[part.View][part.Round] = append(bd.retainBlockParts[part.View][part.Round], part)
}

func (state *State) WriteLogger(msg string, is_start, is_end bool) {
	state.logger.Write(blocklogger.NewConsensusEvent(state.HotStuffState.View, state.HotStuffState.Round, state.step, is_start, is_end, msg))
}
func (state *State) WriteCmd(msg string) {
	fmt.Printf("[%v] (view=%d,round=%d,step=%s) %s\n", time.Now(), state.HotStuffState.View, state.HotStuffState.Round, state.step, msg)
}
