package definition

import (
	"emulator/urd/types"
	"emulator/utils/p2p"
)

type MempoolConn interface {
	AddTx([]byte) error
	ReapTx(maxBytes int) (types.Txs, int, error)
	Update(txs types.Txs, commitStatus []byte) error
	RemoveTx(tx []byte) error

	p2p.Reactor
}

type ABCIConn interface {
	ValidateTx(tx []byte, isCrossShard bool) bool

	// execution and commit
	Execution(types.Txs, types.Txs, []types.Txs) *types.ABCIExecutionResponse
	Commit() []byte

	Stop()
}

type ConsensusConn interface {
	p2p.Reactor
	Start()
	Stop()
}
