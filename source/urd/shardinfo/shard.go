package shardinfo

import (
	"emulator/crypto/merkle"
	"emulator/utils"
	"emulator/utils/p2p"
	"emulator/utils/signer"

	protoinfo "emulator/proto/urd/shardinfo"
	protop2p "emulator/proto/utils/p2p"
)

type Shard struct {
	PeerList    []*p2p.Peer `json:"peer_list"`
	TotalVotes  int32       `json:"total_votes"`
	LeaderIndex int         `json:"leader_index"`
	KeyRange    string      `json:"key_range"`

	verifier *signer.Verifier
}

func (shard *Shard) ToProto() *protoinfo.Shard {
	peers := []*protop2p.Peer{}
	for _, peer := range shard.PeerList {
		peers = append(peers, peer.ToProto())
	}
	return &protoinfo.Shard{
		PeerList:   peers,
		TotalVotes: shard.TotalVotes,
	}
}

func NewShardeFromProto(p *protoinfo.Shard) *Shard {
	peers := []*p2p.Peer{}
	for _, peer := range p.PeerList {
		upeer, err := p2p.NewPeerFromProto(peer)
		if err != nil {
			panic(err)
		}
		peers = append(peers, upeer)
	}
	return &Shard{
		PeerList:   peers,
		TotalVotes: p.TotalVotes,
	}
}

func (shard *Shard) Hash() []byte {
	shardBasic := [][]byte{utils.IntToBytes(shard.TotalVotes)}
	for _, peer := range shard.PeerList {
		shardBasic = append(shardBasic,
			merkle.HashFromByteSlices(
				[][]byte{
					[]byte(peer.IP),
					[]byte(peer.Pubkey),
					utils.IntToBytes(peer.Vote),
				},
			),
		)
	}
	return merkle.HashFromByteSlices(shardBasic)
}

func (shard *Shard) generateVerifier() error {
	pubkeys := []string{}
	for _, peer := range shard.PeerList {
		pubkeys = append(pubkeys, peer.Pubkey)
	}
	verifier, err := signer.NewVerifier(pubkeys)
	if err != nil {
		return err
	}
	shard.verifier = verifier
	return nil
}

func (shard *Shard) Size() int {
	return shard.verifier.Size()
}
func (shard *Shard) Verify(sig string, msg []byte, index int) bool {
	return shard.verifier.Verify(sig, msg, index)
}
func (shard *Shard) VerifyAggregateSignature(aggSig string, msg []byte, bitMapBytes []byte) bool {
	return shard.verifier.VerifyAggregateSignature(aggSig, msg, bitMapBytes)
}
