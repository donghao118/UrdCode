package p2p

import (
	"google.golang.org/protobuf/proto"

	protop2p "emulator/proto/utils/p2p"
	"emulator/utils"
)

type Peer struct {
	IP     string          `json:"ip"`
	Chain  map[string]bool `json:"chain"`
	Pubkey string          `json:"pubkey"`
	Vote   int32           `json:"vote"`
}

func NewPeer(ip string, chains map[string]bool, publicKey string, vote int32) (*Peer, error) {
	return &Peer{
		IP:     ip,
		Chain:  chains,
		Pubkey: publicKey,
		Vote:   vote,
	}, nil
}

func (peer *Peer) Equal(other *Peer) bool {
	return peer.GetIP() == other.GetIP()
}

func (peer *Peer) GetIP() string {
	return peer.IP
}

func (peer *Peer) ChainID() map[string]bool {
	return peer.Chain
}

func (peer *Peer) BelongTo(s string) bool {
	return peer.Chain[s]
}

func (peer *Peer) PubkeyStr() string { return peer.Pubkey }

func (peer *Peer) ToProto() *protop2p.Peer {
	return &protop2p.Peer{
		IP:     peer.GetIP(),
		Chain:  peer.ChainID(),
		Pubkey: peer.PubkeyStr(),
		Vote:   peer.Vote,
	}
}

func (peer *Peer) ProtoBytes() []byte {
	return utils.MustProtoBytes(peer.ToProto())
}

func NewPeerFromProto(p *protop2p.Peer) (*Peer, error) {
	return NewPeer(p.IP, p.Chain, p.Pubkey, p.Vote)
}

func NewPeerFromBytes(bz []byte) (*Peer, error) {
	p := new(protop2p.Peer)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil, err
	} else {
		return NewPeerFromProto(p)
	}
}
