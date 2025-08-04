package minibank

import (
	"bytes"
	bank "emulator/proto/pyramid/abci/minibank"
	"emulator/rapid/definition"
	"emulator/rapid/types"
	"emulator/utils"
	"emulator/utils/store"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

const initBalance = 1000000

type Application struct {
	blockDBConn *store.PrefixStore
	DBConn      *store.PrefixStore

	KeyRangeTree map[string]*utils.RangeTree

	AllShardRangeTree *utils.RangeTree

	IsI       bool
	MyChainID string

	appStatus []byte

	LockedKeys map[string]uint32
	abciLock   sync.Mutex
}

func NewApplication(dbDir string, chain_id string, isI bool, keyRangeTree *utils.RangeTree, relatedIKeyRangeTree map[string]*utils.RangeTree, block_db_conn *store.PrefixStore) *Application {
	app := new(Application)
	app.DBConn = store.NewPrefixStore("abci.minibank", dbDir)
	app.blockDBConn = block_db_conn
	app.AllShardRangeTree, app.KeyRangeTree = keyRangeTree, relatedIKeyRangeTree

	app.IsI = isI
	app.MyChainID = chain_id

	app.appStatus = []byte("minibank")
	app.LockedKeys = map[string]uint32{}
	app.abciLock = sync.Mutex{}
	return app
}

var _ definition.ABCIConn = (*Application)(nil)

func (app *Application) Stop() {
	app.DBConn.Close()
}
func (app *Application) SetBlockStore(bs *store.PrefixStore) {
	app.blockDBConn = bs
}
func (app *Application) StateLock()   { app.abciLock.Lock() }
func (app *Application) StateUnlock() { app.abciLock.Unlock() }
func (app *Application) CheckLock(tx types.Tx) bool {
	ptx, err := app.unmarshalTransferTx(tx)
	if err != nil {
		return true
	}
	rs, ws := app.getRWSet(ptx)
	for r := range rs {
		if !app.requireLock(r) {
			return false
		}
	}
	for w := range ws {
		if !app.requireLock(w) {
			return false
		}
	}
	return true
}

func (app *Application) Commit() []byte {
	return app.appStatus
}
func (app *Application) AcceptCommitShard(hashes [][]byte, commits []bool, chain_id string) (*types.ABCIExecutionResponse, int, int, int) {
	out := new(types.ABCIExecutionResponse)
	app.abciLock.Lock()
	defer app.abciLock.Unlock()
	batch, err := app.DBConn.NewBatch()
	if err != nil {
		panic(err)
	}
	defer batch.Close()

	crossShard, crossTotal := 0, 0
	rt := app.KeyRangeTree[chain_id]
	for i, hs := range hashes {
		fmt.Println(i, "Execute cross shard blocks", hs)
		blockBz, _ := app.blockDBConn.GetBlockByHash(hs)
		/*
			if err != nil || len(blockBz) == 0 {
				out.Receipts = append(out.Receipts, &types.ABCIExecutionReceipt{Code: types.CodeTypeAbort})
				continue
			}
		*/
		block := types.NewBlockFromBytes(blockBz)
		if block == nil {
			out.Receipts = append(out.Receipts, &types.ABCIExecutionReceipt{Code: types.CodeTypeAbort})
			continue
		}
		if icommit, err := app.blockDBConn.GetState(hs); err != nil || !bytes.Equal(icommit, []byte("ok")) {
			out.Receipts = append(out.Receipts, &types.ABCIExecutionReceipt{Code: types.CodeTypeAbort})
			continue
		}
		readSet, writeSet := map[string]bool{}, map[string]bool{}
		for _, tx := range block.CrossShardTxs {
			if ptx, err := app.unmarshalTransferTx(tx); err != nil {
				continue
			} else {
				app.appendRWSet(ptx, readSet, writeSet)
			}
		}
		cnt := block.CrossShardTxs.Size()
		crossTotal += cnt

		if commits[i] {
			for key := range writeSet {
				if !rt.Search(key) {
					continue
				}
				app.releaseLock(key, batch)
			}
			out.Receipts = append(out.Receipts, &types.ABCIExecutionReceipt{Code: types.CodeTypeOK})
			crossShard += cnt
			fmt.Printf("(%v)Completed cross-shard block execution for B shards (%s,height=%d) with %d committed transactions\n",
				time.Now(), block.ChainID, block.Height,
				block.CrossShardTxs.Size())
		} else {
			for key := range writeSet {
				if !rt.Search(key) {
					continue
				}
				app.undoLock(key)
			}
			out.Receipts = append(out.Receipts, &types.ABCIExecutionReceipt{Code: types.CodeTypeAbort})
			fmt.Printf("(%v) completed the cross-shard block execution of B shard (%s,height=%d), and the number of aborted transactions was %d\n",
				time.Now(), block.ChainID, block.Height,
				block.CrossShardTxs.Size())
		}
	}
	if err := batch.WriteSync(); err != nil {
		panic(err)
	}
	return out, 1, 0, 0
}
func (app *Application) AcceptCrossShard(block *types.Block, chain_id string, ifcommit bool) (*types.ABCIExecutionResponse, int, int, int) {
	if block.BlockType != types.BLOCKTYPE_CrossShard {
		panic("The cross-shard block execution phase received the non-cross-shard block")
	}
	app.abciLock.Lock()
	defer app.abciLock.Unlock()
	defer func() {
		words := "lock"
		if !ifcommit {
			words = "Conflict detection"
		}
		fmt.Printf("(%v) completed a cross-shard block %s of B shards (%s,height=%d), totaling %d transactions\n",
			time.Now(), block.ChainID, block.Height,
			words, block.CrossShardTxs.Size())
		fmt.Println(block.Hash())
		app.blockDBConn.SetBlockByHeight(block.Height, block.ChainID, block)
	}()
	receipts := make([]*types.ABCIExecutionReceipt, 0, block.CrossShardTxs.Size())
	if ifcommit {
		memDB := NewMemoryDB(app.DBConn)

		rt := app.KeyRangeTree[chain_id]

		for i, tx := range block.CrossShardTxs {
			receipt := new(types.ABCIExecutionReceipt)
			receipts = append(receipts, receipt)

			csd, ok, err := NewCrossShardDataFromBytes(block.CrossShardDatas[i])
			if !ok || err != nil {
				receipt.Code = types.CodeTypeEncodingError
				if err != nil {
					receipt.Log = err.Error()
				}
				continue
			}
			csdb := NewCrossShardDB(memDB, csd)
			err = app.execute(tx, csdb)
			if err != nil {
				receipt.Code = types.CodeTypeUnknownError
				receipt.Log = err.Error()
			} else {
				receipt.Code = types.CodeTypeOK
			}
			csdb.CommitWrite(err == nil)
		}

		readSet, writeSet := map[string]bool{}, map[string]bool{}
		for i, tx := range block.CrossShardTxs {
			if !receipts[i].IsOK() {
				continue
			}
			ptx, err := app.unmarshalTransferTx(tx)
			if err != nil {
				panic("An error occurred during locking:" + err.Error())
			}
			app.appendRWSet(ptx, readSet, writeSet)
		}
		for key := range writeSet {
			if !rt.Search(key) {
				continue
			}
			u, err := memDB.Get(key)
			if err != nil {
				panic("Error fetching data during locking phase:" + err.Error())
			}
			app.setLock(key, u)
		}
	}

	app.blockDBConn.SetState(block.Hash(), []byte("ok"))
	return &types.ABCIExecutionResponse{
		Receipts: receipts,
	}, 1, 0, 0
}

func (app *Application) ExecutionInnerShard(block *types.Block) (*types.ABCIExecutionResponse, int, int, int) {
	if block.BlockType != types.BLOCKTYPE_InnerShard {
		panic("	The non-on-shard block enters the shard execution")
	}
	var commitTxsCount = 0
	app.abciLock.Lock()
	defer app.abciLock.Unlock()

	mmr := NewMemoryDB(app.DBConn)
	db := NewRangeDB(mmr, app.KeyRangeTree[block.ChainID])
	receipts := make([]*types.ABCIExecutionReceipt, block.BodyTxs.Size())
	for i, tx := range block.BodyTxs {
		abciResp := new(types.ABCIExecutionReceipt)
		err := app.execute([]byte(tx), db)
		if err != nil {
			abciResp.Code = types.CodeTypeAbort
			abciResp.Log = err.Error()
		} else {
			abciResp.Code = types.CodeTypeOK
			commitTxsCount++
		}
		db.CommitWrite(err == nil)
		receipts[i] = abciResp
	}
	if err := db.Write(); err != nil {
		panic(err)
	}
	fmt.Printf("(%v) completed the execution of the in-slice block of shard I (%s,height=%d), totaling %d transactions, and the number of committed transactions is %d\n",
		time.Now(), block.ChainID, block.Height,
		block.BodyTxs.Size(), commitTxsCount)
	app.blockDBConn.SetBlockByHeight(block.Height, block.ChainID, block)
	return &types.ABCIExecutionResponse{Receipts: receipts}, block.BodyTxs.Size(), 0, 0
}

func (app *Application) FillData(block *types.Block) *types.Block {
	if app.IsI {
		panic("The i-shard calls the B-shrad function: FillData")
	}
	//app.abciLock.Lock()
	//defer app.abciLock.Unlock()
	txs := block.CrossShardBody.CrossShardTxs
	var memoryDB = NewMemoryDB(app.DBConn)
	var rangeDB = NewRangeDB(memoryDB, app.AllShardRangeTree)
	csd := make([][]byte, txs.Size())
	for i, tx := range txs {
		err := app.execute(tx, rangeDB)
		readSet := map[string]uint32{}
		if err == nil {
			readSet = rangeDB.GetReadSet()
		}
		rangeDB.CommitWrite(err == nil)
		csd[i] = CrossShardDataBytes(readSet, err == nil)
	}
	block.CrossShardBody.CrossShardDatas = csd
	return block
}

func (app *Application) PreExecutionB(block *types.Block) *types.ABCIPreExecutionResponseB {
	app.abciLock.Lock()
	defer app.abciLock.Unlock()

	return &types.ABCIPreExecutionResponseB{
		Code:            types.CodeTypeOK,
		CrossShardDatas: block.CrossShardBody.CrossShardDatas,
		BehaviorErrors:  nil,
	}
}

func (app *Application) PreExecutionI(block *types.Block) *types.ABCIPreExecutionResponseI {
	if block.BlockType != types.BLOCKTYPE_CrossShard {
		panic("Pre-execution is invoked for the intra-shard blocks")
	}
	if !app.IsI {
		panic("The B-shard calls the i-shard function: PreExecutionI")
	}
	app.abciLock.Lock()
	defer app.abciLock.Unlock()

	readSet, writeSet := map[string]bool{}, map[string]bool{}
	for _, tx := range block.CrossShardTxs {
		ptx, err := app.unmarshalTransferTx(tx)
		if err != nil {
			return &types.ABCIPreExecutionResponseI{
				Code:           types.CodeTypeAbort,
				BehaviorErrors: []error{errors.New("There are non-serializable transactions")},
			}
		}
		app.appendRWSet(ptx, readSet, writeSet)
	}
	for key := range readSet {
		if !app.requireLock(key) {
			return &types.ABCIPreExecutionResponseI{
				Code: types.CodeTypeAbort,
				BehaviorErrors: []error{
					errors.New("Locked primary key accessed. Conflict!"),
				},
			}
		}
	}
	for key := range writeSet {
		if !app.requireLock(key) {
			return &types.ABCIPreExecutionResponseI{
				Code: types.CodeTypeAbort,
				BehaviorErrors: []error{
					errors.New("Locked primary key accessed. Conflict!！"),
				},
			}
		}
	}

	memoryDB := NewMemoryDB(app.DBConn)
	for i, tx := range block.BodyTxs {
		csdb, ok, err := NewCrossShardDataFromBytes(block.CrossShardDatas[i])
		if err != nil {
			return &types.ABCIPreExecutionResponseI{
				Code: types.CodeTypeAbort,
				BehaviorErrors: []error{
					errors.New("Problem with cross-shard data!"),
				},
			}
		}
		crossShardDataFreshFlag := true
		for key, value := range csdb {
			if app.AllShardRangeTree.Search(key) {
				if v, err := memoryDB.Get(key); err != nil || v != value {
					crossShardDataFreshFlag = false
					break
				}
			}
		}
		if !crossShardDataFreshFlag {
			return &types.ABCIPreExecutionResponseI{
				Code: types.CodeTypeAbort,
				BehaviorErrors: []error{
					fmt.Errorf("Cross-shard data is stale (%d)!", i),
				},
			}
		}
		if ok {
			db := NewCrossShardDB(memoryDB, csdb)
			err := app.execute([]byte(tx), db)
			db.CommitWrite(err == nil)
		}
	}
	return &types.ABCIPreExecutionResponseI{Code: types.CodeTypeOK}
}

func (app *Application) requireLock(key string) bool {
	if _, ok := app.LockedKeys[key]; ok {
		return false
	}
	return true
}
func (app *Application) setLock(key string, value uint32) bool {
	if _, ok := app.LockedKeys[key]; ok {
		return false
	}
	app.LockedKeys[key] = value
	return true
}
func (app *Application) releaseLock(key string, batch *store.PrefixBatch) error {
	if value, ok := app.LockedKeys[key]; !ok {
		return nil
	} else {
		delete(app.LockedKeys, key)
		if batch == nil {
			return nil
		}
		return batch.Set([]byte(key), utils.Uint32ToBytes(value))
	}
}
func (app *Application) undoLock(key string) error {
	delete(app.LockedKeys, key)
	return nil
}

func (app *Application) appendRWSet(tx *bank.TransferTx, readset, writeSet map[string]bool) {
	for _, key := range tx.From {
		writeSet[key] = true
	}
	for _, key := range tx.To {
		writeSet[key] = true
	}
}

func (app *Application) getRWSet(tx *bank.TransferTx) (map[string]bool, map[string]bool) {
	readSet, writeSet := map[string]bool{}, map[string]bool{}
	for _, key := range tx.From {
		writeSet[key] = true
		readSet[key] = true
	}
	for _, key := range tx.To {
		writeSet[key] = true
		readSet[key] = true
	}
	return readSet, writeSet
}

func (app *Application) unmarshalTransferTx(tx types.Tx) (*bank.TransferTx, error) {
	if len(tx) < 4 {
		return nil, errors.New("Given tx length less than 4, invalid format")
	}
	if txType := utils.BytesToUint32(tx[:4]); txType != definition.TxTransfer {
		return nil, errors.New("Given tx is not a transfer transaction")
	}
	return NewTransferTxFromBytes(tx)
}

func (app *Application) execute(txBytes []byte, db GetSetMemDB) error {
	if len(txBytes) < 4 {
		return errors.New("Given tx length less than 4, invalid format")
	}
	txType := utils.BytesToUint32(txBytes[:4])
	switch txType {
	case definition.TxTransfer:
		tx := new(bank.TransferTx)
		if err := proto.Unmarshal(txBytes[4:], tx); err != nil {
			return err
		}
		return app.doTransfer(tx, db)
	default:
		return errors.New("ABCI Tx does not define a type")
	}
}

func (app *Application) doTransfer(tx *bank.TransferTx, db GetSetMemDB) error {
	if err := ValidateTransferTx(tx); err != nil {
		return err
	}
	fromBalance, toBalance := make([]uint32, len(tx.From)), make([]uint32, len(tx.To))
	for i, fromKey := range tx.From {
		if _, ok := app.LockedKeys[fromKey]; ok {
			return errors.New("Locked primary key accessed")
		}
		if balance, err := db.Get(fromKey); err != nil {
			// return err
			fromBalance[i] = initBalance - tx.FromMoney[i]
		} else if balance < tx.FromMoney[i] {
			return errors.New("Balance is not Enough")
		} else {
			fromBalance[i] = balance - tx.FromMoney[i]
		}
	}
	for i, toKey := range tx.To {
		if _, ok := app.LockedKeys[toKey]; ok {
			return errors.New("Locked primary key accessed")
		}
		if balance, err := db.Get(toKey); err != nil {
			toBalance[i] = initBalance + tx.ToMoney[i]
		} else {
			toBalance[i] = balance + tx.ToMoney[i]
		}
	}

	for i, fromKey := range tx.From {
		if _, ok := app.LockedKeys[fromKey]; ok {
			return errors.New("Locked primary key accessed")
		}

		if err := db.Set(fromKey, fromBalance[i]); err != nil {
			return err
		}
	}
	for i, toKey := range tx.To {
		if _, ok := app.LockedKeys[toKey]; ok {
			return errors.New("Locked primary key accessed")
		}
		if err := db.Set(toKey, toBalance[i]); err != nil {
			return err
		}
	}
	return nil
}

func (app *Application) ValidateTx(txBytes types.Tx, isCrossShard bool) bool {
	if len(txBytes) < 4 {
		return false
	}
	var rangeTree *utils.RangeTree
	if app.IsI {
		if isCrossShard {
			return false
		}
		rangeTree = app.AllShardRangeTree
	} else {
		if isCrossShard {
			rangeTree = app.AllShardRangeTree
		} else {
			rangeTree = app.KeyRangeTree[app.MyChainID]
		}
	}
	txType := utils.BytesToUint32(txBytes[:4])
	switch txType {
	case definition.TxTransfer:
		tx, err := app.unmarshalTransferTx(txBytes)
		if err != nil {
			return false
		}
		for _, key := range tx.From {
			if !rangeTree.Search(key) {
				return false
			}
		}
		for _, key := range tx.To {
			if !rangeTree.Search(key) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
