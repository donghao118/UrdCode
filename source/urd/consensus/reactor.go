package consensus

import (
	"emulator/utils/p2p"
	"fmt"
)

func (cs *State) SendToShard(shardID string, bz []byte, messageType uint32) {
	if err := cs.p2p.SendToShard(shardID, p2p.ChannelIDConsensusState, bz, messageType); err != nil {
		cs.write_p2p_error(err)
	}
}
func (cs *State) SendTo(shard string, index int, bz []byte, messageType uint32) {
	if err := cs.p2p.SendToShardIndex(shard, index, p2p.ChannelIDConsensusState, bz, messageType); err != nil {
		cs.write_p2p_error(err)
	}
}

func (cs *State) write_p2p_error(err error) {
	cs.WriteCmd(fmt.Sprintf("p2p error: %v", err))
}
