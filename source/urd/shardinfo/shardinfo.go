package shardinfo

import (
	"emulator/crypto/merkle"
	protoinfo "emulator/proto/urd/shardinfo"
	"emulator/utils/p2p"
	"encoding/json"
	"fmt"
	"sort"
)

type ShardInfo struct {
	Shards      map[string]*Shard
	ShardIDList []string
}

func NewShardInfo(peers map[string][]*p2p.Peer, leader_index int, keyRanges map[string]string) (si *ShardInfo) {
	shards := make(map[string]*Shard)
	for shard, ps := range peers {
		var totalVotes int32 = 0
		for _, p := range ps {
			totalVotes += p.Vote
		}
		shards[shard] = &Shard{
			PeerList:    ps,
			TotalVotes:  totalVotes,
			LeaderIndex: leader_index,
			KeyRange:    keyRanges[shard],
		}
	}
	si = &ShardInfo{
		Shards: shards,
	}
	if err := si.init(); err != nil {
		panic(err)
	}
	return si
}

func (si *ShardInfo) init() error {
	si.ShardIDList = []string{}
	for id, shard := range si.Shards {
		if err := shard.generateVerifier(); err != nil {
			return err
		}
		fmt.Println("Generated Verifier for Shard:", id)
		si.ShardIDList = append(si.ShardIDList, id)
	}
	sort.Strings(si.ShardIDList)
	return nil
}

func (si *ShardInfo) ToProto() *protoinfo.ShardInfo {
	shards := make(map[string]*protoinfo.Shard)
	for key, shard := range si.Shards {
		shards[key] = shard.ToProto()
	}
	return &protoinfo.ShardInfo{
		ShardList: shards,
	}
}

func NewShardInfoFromProto(p *protoinfo.ShardInfo) *ShardInfo {
	shards := make(map[string]*Shard)
	for key, shard := range p.ShardList {
		shards[key] = NewShardeFromProto(shard)
	}
	si := &ShardInfo{
		Shards: shards,
	}
	if err := si.init(); err != nil {
		fmt.Println("ShardInfo init error:", err)
		return nil
	}
	return si
}
func (si *ShardInfo) Hash() []byte {
	shardBasic := [][]byte{}
	keys := []string{}
	for k := range si.Shards {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		shard := si.Shards[key]
		shardBasic = append(shardBasic, shard.Hash())
	}
	return merkle.HashFromByteSlices(shardBasic)
}

func (si *ShardInfo) MarshalJson() ([]byte, error) {
	return json.MarshalIndent(si, "", "  ")
}
func (si *ShardInfo) UnmarshalJson(data []byte) error {
	err := json.Unmarshal(data, si)
	if err != nil {
		return err
	}
	if err := si.init(); err != nil {
		return err
	}
	return nil
}
