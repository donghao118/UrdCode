package definition

import (
	"emulator/pyramid/types"
	"emulator/utils/p2p"
	"emulator/utils/store"
)

type MempoolConn interface {
	AddTx(*types.Tx) error
	ReapTx(maxTxs int) (types.Txs, int, error)
	Update(txs types.Txs, commitStatus []byte) error
	RemoveTx(tx *types.Tx) error

	p2p.Reactor
}

type ABCIConn interface {
	ValidateTx(tx types.Tx, isCrossShard bool) bool

	PreExecutionB(*types.Block) *types.ABCIPreExecutionResponseB
	PreExecutionI(*types.Block) *types.ABCIPreExecutionResponseI

	ExecutionInnerShard(*types.Block) (*types.ABCIExecutionResponse, int, int, int)           
	AcceptCrossShard(*types.Block, string, bool) (*types.ABCIExecutionResponse, int, int, int) 
	AcceptCommitShard([][]byte, []bool, string) (*types.ABCIExecutionResponse, int, int, int)  

	Commit() []byte

	StateLock()
	FillData(*types.Block) *types.Block // No State Lock
	StateUnlock()
	CheckLock(tx types.Tx) bool

	Stop()

	SetBlockStore(*store.PrefixStore)
}

type ConsensusConn interface {
	p2p.Reactor
	Start()
	Stop()
}
