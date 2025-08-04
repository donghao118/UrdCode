package p2p

const (
	ChannelIDConsensusState    = 0x21
	ChannelIDMempool           = 0x31
	ChannelIDCrossShardMempool = 0x32
)

type Reactor interface {
	//AddPeer(peer Peer) error

	Receive(chID byte, message []byte, messageType uint32)error
}
