package mempool

// 非常轻量化的Mempool设计
// 这个Mempool是不安全的，它无法拒绝重复事务，仅停留在实验室基础上
import (
	"emulator/libs/clist"
	"emulator/urd/definition"
	"emulator/urd/types"
	"fmt"
	"sync"
)

type Mempool struct {
	txs    *clist.CList
	txsMap sync.Map

	abci                definition.ABCIConn
	isCrossShardMempool bool
}

func NewMempool(isCrossShardMempool bool, abci definition.ABCIConn) *Mempool {
	return &Mempool{
		txs:    clist.New(),
		txsMap: sync.Map{},

		abci:                abci,
		isCrossShardMempool: isCrossShardMempool,
	}
}

var _ definition.MempoolConn = (*Mempool)(nil)

func (mpl *Mempool) Receive(chID byte, message []byte, messageType uint32) error {
	tx := message
	return mpl.AddTx(tx)
}

func (mpl *Mempool) AddTx(tx []byte) error {
	if !mpl.abci.ValidateTx(tx, mpl.isCrossShardMempool) {
		return fmt.Errorf("An invalid transaction")
	}
	e := mpl.txs.PushBack(&tx)
	mpl.txsMap.Store(types.TxKey(tx), e)
	return nil
}

func (mpl *Mempool) RemoveTx(tx []byte) error {
	if e, ok := mpl.txsMap.Load(types.TxKey(tx)); ok {
		if u, ok := e.(*clist.CElement); ok {
			mpl.txs.Remove(u)
			mpl.txsMap.Delete(types.TxKey(tx))
		}
	}
	return nil
}

func (mpl *Mempool) ReapTx(maxTxsBz int) (types.Txs, int, error) {
	var (
		txs = make(types.Txs, 0)
	)
	fmt.Println("ReapTx called, maxTxs:", maxTxsBz)
	fmt.Println("Current mempool size:", mpl.txs.Len())
	current := 0
	for e := mpl.txs.Front(); e != nil; e = e.Next() {
		memTx := *e.Value.(*[]byte)
		current += len(memTx)
		if current >= maxTxsBz {
			break
		}
		txs = append(txs, memTx)
	}
	return txs, len(txs), nil
}

func (mpl *Mempool) Update(txs types.Txs, commitStatus []byte) error {
	if len(txs) == 0 {
		return nil
	}
	for _, tx := range txs {
		mpl.RemoveTx(tx)
	}
	return nil
}
