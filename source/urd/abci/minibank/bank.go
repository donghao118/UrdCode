package minibank

import (
	"bytes"
	"emulator/crypto/merkle"
	bank "emulator/proto/urd/abci/minibank"
	"emulator/urd/definition"
	"emulator/urd/shardinfo"
	"emulator/urd/types"
	"emulator/utils"
	"emulator/utils/store"
	"errors"
	"fmt"
	"sort"

	"google.golang.org/protobuf/proto"
)

const initBalance = 1000000

var prefix_of_undo_relay = []byte("undo")

func toRelayKey(key []byte) []byte {
	return bytes.Join([][]byte{prefix_of_undo_relay, key}, nil)
}

type Application struct {
	db *store.PrefixStore

	KeyRangeTrees   map[string]*utils.RangeList
	chain_id        string
	shards_to_index map[string]int
	shard_info      *shardinfo.ShardInfo

	appStatus []byte
}

func NewApplication(dbDir string, chain_id string, keyRangeTrees map[string]*utils.RangeList, shard_info *shardinfo.ShardInfo) *Application {
	app := new(Application)
	app.db = store.NewPrefixStore("abci.minibank", dbDir)

	app.KeyRangeTrees = keyRangeTrees
	app.chain_id = chain_id
	app.shards_to_index = make(map[string]int)
	for i, shard := range shard_info.ShardIDList {
		app.shards_to_index[shard] = i
	}
	app.shard_info = shard_info

	app.appStatus = merkle.HashFromByteSlices(nil)

	return app
}

func (app *Application) Stop() {
	app.db.Close()
}

var _ definition.ABCIConn = (*Application)(nil)

func (app *Application) Commit() []byte {
	return app.appStatus
}
func (app *Application) ValidateTx(tx []byte, isCrossShard bool) bool {
	if err := app.validateTx(tx); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (app *Application) Execution(txs types.Txs, cross_shard_txs types.Txs, CTXS []types.Txs) *types.ABCIExecutionResponse {
	resp := new(types.ABCIExecutionResponse)
	db := NewDB(app.db, app.KeyRangeTrees[app.chain_id])
	for i, ctxs := range CTXS {
		chain := app.shard_info.ShardIDList[i]
		for _, opt := range ctxs {
			app.executeRelay(opt, chain, db)
		}
	}
	for _, tx := range txs {
		receipt := new(types.ABCIExecutionReceipt)
		err := app.execute(tx, db)
		if err != nil {
			receipt.Code = types.CodeTypeAbort
			receipt.Info = err.Error()
		} else {
			receipt.Code = types.CodeTypeOK
		}
		receipt.SetRawTx(tx)
		resp.Responses = append(resp.Responses, receipt)
	}
	resp.OPTxs = app.preExecution(cross_shard_txs)
	return resp
}

// =======================================================================================

func (app *Application) preExecution(input types.Txs) []types.Txs {
	relayTxs := make([]types.Txs, len(app.shards_to_index))
	db := NewDB(app.db, app.KeyRangeTrees[app.chain_id])
	wlocks, rlocks := make(map[string]bool), make(map[string]bool)
	for _, txBytes := range input {
		tx, err := NewTransferTxFromBytes(txBytes)
		if err != nil {
			continue
		} else if len(tx.Shards) <= 1 {
			continue
		}
		relayTx, dstShards, err := app.pre_doTransfer(tx, txBytes, wlocks, rlocks, db)
		if err != nil {
			//fmt.Println("pre do transfer error:", err)
			continue
		}
		for _, shard := range dstShards {
			relayTxs[app.shards_to_index[shard]] = append(relayTxs[app.shards_to_index[shard]], relayTx)
		}
		if err := app.executeCrossShard(txBytes, db); err != nil {
			//fmt.Println("execute cross shard error:", err)
			continue
		}
	}
	return relayTxs
}

func (app *Application) executeRelay(txBytes []byte, chain string, db DB) error {
	tx, err := NewRelayTransferTxFromBytes(txBytes)
	if err != nil {
		return err
	}
	return app.unlockTransfer(tx, chain, db)
}
func (app *Application) executeCrossShard(txBytes []byte, db DB) error {
	if len(txBytes) < 4 {
		return errors.New("Invalid Transaction Type")
	}
	txType := utils.BytesToUint32(txBytes[:4])
	switch txType {
	case definition.TxTransfer:
		tx, err := NewTransferTxFromBytes(txBytes)
		if err != nil {
			return err
		}
		if len(tx.Shards) > 1 {
			return app.lockTransfer(tx, txBytes, db)
		} else {
			return fmt.Errorf("tx for invalid shards")
		}
	default:
		return errors.New("ABCI Tx unknown type")
	}
}
func (app *Application) execute(txBytes []byte, db DB) error {
	if len(txBytes) < 4 {
		return errors.New("Invalid Transaction Type")
	}
	txType := utils.BytesToUint32(txBytes[:4])
	switch txType {
	case definition.TxTransfer:
		tx, err := NewTransferTxFromBytes(txBytes)
		if err != nil {
			return err
		}
		switch len(tx.Shards) {
		case 1:
			if tx.Shards[0] != app.chain_id {
				return fmt.Errorf("tx does not belong to this shard")
			}
			return app.doTransfer(tx, db)
		default:
			return fmt.Errorf("tx for invalid shards")
		}
	default:
		return errors.New("ABCI Tx unknown type")
	}
}

func (app *Application) unlockTransfer(tx *bank.RelayTransferTx, chain string, db DB) error {
	hash := tx.TxHash
	var relayTxSet *bank.RelayTransferTxSet
	if bz, err := app.db.GetSpecial(hash); err != nil {
		return err
	} else if len(bz) == 0 {
		var bankdatas map[string]*bank.BankData
		if bank_datas_bz, err := app.db.GetSpecial(toRelayKey(hash)); err != nil {
			return err
		} else if len(bank_datas_bz) == 0 {
			bankdatas = map[string]*bank.BankData{}
		} else if bankdatas, err = RelayTransferTxListFromBytes(bank_datas_bz); err != nil {
			return err
		}
		bankdatas[chain] = tx.Datas
		if rbz, err := RelayTransferTxListBytes(bankdatas); err != nil {
			return err
		} else if err := app.db.SetSpecial(toRelayKey(hash), rbz); err != nil {
			return err
		}
		return nil
	} else if relayTxSet, err = NewRelayTransferTxSetFromBytes(bz); err != nil {
		return err
	} else {
		relayTxSet = app.insertBankDataToRelayTxSet(tx.Datas, chain, relayTxSet)
	}

	if !isRelayTransferTxSetFinish(relayTxSet) {
		return nil
	}
	defer db.Clear()
	for i, data := range relayTxSet.Datas {
		if relayTxSet.Shards[i] == app.chain_id {
			for _, key := range data.Keys {
				if err := db.WUnlock(key); err != nil {
					return err
				}
			}
		}
		for i := range data.Keys {
			db.LoadData(data.Keys[i], data.Values[i])
		}
	}
	rawTx, err := NewTransferTxFromBytes(relayTxSet.RawTx)
	if err != nil {
		return err
	}
	return app.doTransfer(rawTx, db)
}

func (app *Application) insertBankDataToRelayTxSet(bank_data *bank.BankData, chain string, relayTxSet *bank.RelayTransferTxSet) *bank.RelayTransferTxSet {
	for i, shard := range relayTxSet.Shards {
		if shard == chain {
			relayTxSet.Datas[i] = bank_data
		}
	}
	return relayTxSet
}

func (app *Application) lockTransfer(tx *bank.TransferTx, raw_tx []byte, db DB) error {
	// we have validated this transaction when pre_execution
	// the only thing todo is lock those txs, which should have been done in pre_execution phase
	hash := types.TxHash(raw_tx)

	relayTxSet := &bank.RelayTransferTxSet{
		Shards: tx.Shards,
		Datas:  make([]*bank.BankData, len(tx.Shards)),
		RawTx:  raw_tx,
	}
	if bz, err := app.db.GetSpecial(toRelayKey(hash)); err != nil {
		return err
	} else if len(bz) > 0 {
		if datas, err := RelayTransferTxListFromBytes(bz); err != nil {
			return err
		} else {
			for shard, bank_data := range datas {
				app.insertBankDataToRelayTxSet(bank_data, shard, relayTxSet)
			}
		}
	}

	setBz, err := RelayTransferTxSetBytes(relayTxSet)
	if err != nil {
		return err
	}
	if err := app.db.SetSpecial(hash, setBz); err != nil {
		return err
	}

	for _, key := range append(tx.From, tx.To...) {
		if !app.search_key_intra_shard(key) {
			continue
		}
		db.WLock(key)
	}
	return nil
}

func (app *Application) pre_doTransfer(tx *bank.TransferTx, raw_tx []byte, wlocks, rlocks map[string]bool, db DB) ([]byte, []string, error) {
	if !utils.StrIn(app.chain_id, tx.Shards) {
		return nil, nil, fmt.Errorf("tx is not included in related shards")
	}
	hash := types.TxHash(raw_tx)
	if ok, err := app.db.HasSpecial(hash); err != nil {
		return nil, nil, err
	} else if ok {
		return nil, nil, fmt.Errorf("tx already committed")
	}
	if err := ValidateTransferTx(tx); err != nil {
		return nil, nil, err
	}
	var relayTx = new(bank.RelayTransferTx)
	var bankData = new(bank.BankData)
	relayTx.Datas = bankData
	relayTx.TxHash = hash
	related_shards := []string{}
	related_shards_map := map[string]bool{}
	for _, key := range append(tx.From, tx.To...) {
		if shard, err := app.search_key_shard(key); err != nil {
			return nil, nil, err
		} else {
			related_shards_map[shard] = true
			if shard != app.chain_id {
				continue
			}
		}
		money, locked, err := db.Get(key)
		if err != nil {
			return nil, nil, err
		}
		if !isFree(locked) || wlocks[key] || rlocks[key] {
			return nil, nil, fmt.Errorf("Abort due to lock conflict")
		}
		wlocks[key] = true
		bankData.Keys = append(bankData.Keys, key)
		bankData.Values = append(bankData.Values, money)
	}
	relayTxBz, err := RelayTransferTxBytes(relayTx)
	if err != nil {
		return nil, nil, err
	}
	for k := range related_shards_map {
		related_shards = append(related_shards, k)
	}
	sort.Strings(related_shards)
	if !utils.StrEqual(tx.Shards, related_shards) {
		return nil, nil, fmt.Errorf("invalid cross shard fields: %v != %v", tx.Shards, related_shards)
	}
	return relayTxBz, related_shards, nil
}

func (app *Application) doTransfer(tx *bank.TransferTx, db DB) error {
	if !utils.StrIn(app.chain_id, tx.Shards) {
		return fmt.Errorf("tx is not included in related shards")
	}
	if err := ValidateTransferTx(tx); err != nil {
		return err
	}
	fromBalance, toBalance := make([]uint32, len(tx.From)), make([]uint32, len(tx.To))
	// 1. read Balance
	for i, fromKey := range tx.From {
		if balance, locked, err := db.Get(fromKey); err != nil {
			return err
			//fromBalance[i] = initBalance - tx.FromMoney[i]
		} else if !isFree(locked) {
			return fmt.Errorf("one of its keys is locked")
		} else if balance < tx.FromMoney[i] {
			return errors.New("Balance is not Enough")
		} else {
			fromBalance[i] = balance - tx.FromMoney[i]
		}
	}
	for i, toKey := range tx.To {
		if balance, locked, err := db.Get(toKey); err != nil {
			return err
			//toBalance[i] = initBalance + tx.ToMoney[i]
		} else if !isFree(locked) {
			return fmt.Errorf("one of its keys is locked")
		} else {
			toBalance[i] = balance + tx.ToMoney[i]
		}
	}
	// 2. write Balance
	for i, fromKey := range tx.From {
		if err := db.Set(fromKey, fromBalance[i], FreeIdentifier); err != nil {
			return err
		}
	}
	for i, toKey := range tx.To {
		if err := db.Set(toKey, toBalance[i], FreeIdentifier); err != nil {
			return err
		}
	}
	return nil
}

func (app *Application) validateTx(txBytes []byte) (err error) {
	if len(txBytes) < 4 {
		return fmt.Errorf("TxLength is not enough")

	}
	txType := utils.BytesToUint32(txBytes[:4])
	switch txType {
	case definition.TxTransfer:
		tx := new(bank.TransferTx)
		if err := proto.Unmarshal(txBytes[4:], tx); err != nil {
			return fmt.Errorf("fail to unmarshsal")
		}
		return nil
	case definition.TxInsert:
		tx := new(bank.InsertTx)
		if err := proto.Unmarshal(txBytes[4:], tx); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown transaction type")
	}
}

func (app *Application) search_key_shard(key string) (string, error) {
	for shard, rangeTree := range app.KeyRangeTrees {
		if rangeTree.Search(key) {
			return shard, nil
		}
	}
	return "", fmt.Errorf("key %s not exist", key)
}

func (app *Application) search_key_intra_shard(key string) bool {
	return app.KeyRangeTrees[app.chain_id].Search(key)
}
