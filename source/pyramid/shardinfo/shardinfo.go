package shardinfo

import (
	"emulator/utils/p2p"

	protoshardinfo "emulator/proto/pyramid/shardinfo"
	protop2p "emulator/proto/utils/p2p"

	"emulator/pyramid/types"

	"google.golang.org/protobuf/proto"
)

type ShardInfo struct {
	PeerList       map[string][]*p2p.Peer `json:"peer_list"`
	RelatedShards  map[string]bool        `json:"related_shards"`
	PeerRelatedMap map[string][]string    `json:"peer_related_map"`
	ShardIdentity  int8                   `json:"shard_identity(I=1 B=2)"`
	KeyRangeMap    map[string]string      `json:"key_range"`
}

func (s *ShardInfo) SetIShard()     { s.ShardIdentity = types.BLOCKTYPE_I }
func (s *ShardInfo) SetBShard()     { s.ShardIdentity = types.BLOCKTYPE_B }
func (s *ShardInfo) IsIShard() bool { return s.ShardIdentity == types.BLOCKTYPE_I }
func (s *ShardInfo) IsBShard() bool { return s.ShardIdentity == types.BLOCKTYPE_B }
func (s *ShardInfo) ToProto() *protoshardinfo.ShardInfo {
	plMap := make(map[string]*protoshardinfo.RepeatedPeers)
	for key, value := range s.PeerList {
		proto_value := make([]*protop2p.Peer, len(value))
		for i, v := range value {
			proto_value[i] = v.ToProto()
		}
		plMap[key] = &protoshardinfo.RepeatedPeers{Peers: proto_value}
	}
	prMap := make(map[string]*protoshardinfo.RepeatedString)
	for key, value := range s.PeerRelatedMap {
		prMap[key] = &protoshardinfo.RepeatedString{Str: value}
	}
	return &protoshardinfo.ShardInfo{
		PeerList:       plMap,
		RelatedShards:  s.RelatedShards,
		PeerRelatedMap: prMap,
		ShardIdentity:  int32(s.ShardIdentity),
	}
}

func (s *ShardInfo) ProtoBytes() []byte {
	return types.MustProtoBytes(s.ToProto())
}

func NewShardInfoFromProto(p *protoshardinfo.ShardInfo) *ShardInfo {
	plMap := make(map[string][]*p2p.Peer)
	for key, value := range p.PeerList {
		peers := make([]*p2p.Peer, len(value.Peers))
		for i, v := range value.Peers {
			upeer, err := p2p.NewPeerFromProto(v)
			if err != nil {
				panic(err)
			} else {
				peers[i] = upeer
			}
		}
		plMap[key] = peers
	}
	prMap := make(map[string][]string)
	for key, value := range p.PeerRelatedMap {
		prMap[key] = value.GetStr()
	}
	return &ShardInfo{
		PeerList:       plMap,
		RelatedShards:  p.RelatedShards,
		PeerRelatedMap: prMap,
		ShardIdentity:  int8(p.ShardIdentity),
	}
}

func NewShardInfoFromBytes(bz []byte) *ShardInfo {
	var p = new(protoshardinfo.ShardInfo)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewShardInfoFromProto(p)
	}
}
