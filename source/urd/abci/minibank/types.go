package minibank

import (
	"bytes"
	bank "emulator/proto/urd/abci/minibank"
	"emulator/urd/definition"
	"emulator/urd/types"
	"emulator/utils"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"
)

const (
	RLockedIdentifier = '0'
	WLockedIdentifier = '1'
	FreeIdentifier    = '2'
)

func TransferTxSize(tx *bank.TransferTx) int {
	return len(TransferBytes(tx))
}
func InsertTxSize(tx *bank.InsertTx) int {
	return len(InsertBytes(tx))
}

func TransferBytes(tx *bank.TransferTx) []byte {
	return bytes.Join([][]byte{utils.Uint32ToBytes(definition.TxTransfer), types.MustProtoBytes(tx)}, nil)
}
func InsertBytes(tx *bank.InsertTx) []byte {
	return bytes.Join([][]byte{utils.Uint32ToBytes(definition.TxInsert), types.MustProtoBytes(tx)}, nil)
}

func NewTransferTx(froms []string, fromMoney []uint32,
	tos []string, toMoney []uint32, related_shards []string) *bank.TransferTx {
	return &bank.TransferTx{
		From:      froms,
		To:        tos,
		FromMoney: fromMoney,
		ToMoney:   toMoney,
		Time:      utils.ThirdPartyProtoTime(time.Now()),
		Shards:    related_shards,
	}
}

func NewTransferTxMustLen(froms []string, fromMoney []uint32,
	tos []string, toMoney []uint32, related_shards []string,
	mustLen int) *bank.TransferTx {
	bankProtoer := NewTransferTx(froms, fromMoney, tos, toMoney, related_shards)
	delta := mustLen - TransferTxSize(bankProtoer)
	if delta > 0 {
		bankProtoer.Buffer = GenerateRandomBytes(delta)
	}
	return bankProtoer
}
func NewInsertTx(account string, money uint32) *bank.InsertTx {
	return &bank.InsertTx{
		Account: account,
		Money:   money,
		Time:    utils.ThirdPartyProtoTime(time.Now()),
	}
}
func NewInsertTxMustLen(account string, money uint32, mustLen int) *bank.InsertTx {
	bankProtoer := &bank.InsertTx{
		Account: account,
		Money:   money,
		Time:    utils.ThirdPartyProtoTime(time.Now()),
	}
	delta := mustLen - InsertTxSize(bankProtoer)
	if delta > 0 {
		bankProtoer.Buffer = GenerateRandomBytes(delta)
	}
	return bankProtoer
}

func ValidateTransferTx(tx *bank.TransferTx) error {
	if len(tx.From) != len(tx.FromMoney) || len(tx.To) != len(tx.ToMoney) {
		return errors.New("From/FromMoney and To/ToMoney is not the same length")
	}
	if len(tx.From) == 0 || len(tx.To) == 0 {
		return errors.New("From/To is empty")
	}

	fromMoneySum := uint32(0)
	toMoneySum := uint32(0)

	for _, money := range tx.FromMoney {
		fromMoneySum += money
	}

	for _, money := range tx.ToMoney {
		toMoneySum += money
	}

	if fromMoneySum != toMoneySum {
		return errors.New("FromMoney and ToMoney are not equal")
	}

	return nil
}

func GenerateRandomBytes(n int) []byte {
	var out = make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = '0'
	}
	return out
}

func NewBankDataFromBytes(bz []byte) (*bank.BankData, error) {
	var bd = new(bank.BankData)
	if err := proto.Unmarshal(bz, bd); err != nil {
		return nil, err
	}
	return bd, nil
}
func BankDataBytes(data *bank.BankData) ([]byte, error) {
	return types.MustProtoBytes(data), nil
}

func NewTransferTxFromBytes(txBytes []byte) (*bank.TransferTx, error) {
	tx := new(bank.TransferTx)
	if err := proto.Unmarshal(txBytes[4:], tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func MarshalValue(money uint32, locked byte) ([]byte, error) {
	var out []byte = []byte{locked}
	out = append(out, utils.Uint32ToBytes(money)...)
	return out, nil
}
func UnmarshalValue(bz []byte) (money uint32, locked byte, err error) {
	locked = bz[0]
	money = utils.BytesToUint32(bz[1:])
	return
}
func SetValueUnlock(bz []byte) []byte {
	bz[0] = FreeIdentifier
	return bz
}
func SetValueRLock(bz []byte) []byte {
	bz[0] = RLockedIdentifier
	return bz
}
func SetValueWLock(bz []byte) []byte {
	bz[0] = WLockedIdentifier
	return bz
}

func RelayTransferTxBytes(tx *bank.RelayTransferTx) ([]byte, error) {
	return types.MustProtoBytes(tx), nil
}
func NewRelayTransferTxFromBytes(bz []byte) (*bank.RelayTransferTx, error) {
	var tx = new(bank.RelayTransferTx)
	if err := proto.Unmarshal(bz, tx); err != nil {
		return nil, err
	} else {
		return tx, nil
	}
}

func RelayTransferTxSetBytes(tx *bank.RelayTransferTxSet) ([]byte, error) {
	return types.MustProtoBytes(tx), nil
}
func NewRelayTransferTxSetFromBytes(bz []byte) (*bank.RelayTransferTxSet, error) {
	var tx = new(bank.RelayTransferTxSet)
	if err := proto.Unmarshal(bz, tx); err != nil {
		return nil, err
	} else {
		return tx, nil
	}
}

func isRelayTransferTxSetFinish(tx *bank.RelayTransferTxSet) bool {
	for _, bd := range tx.Datas {
		if bd == nil {
			return false
		}
	}
	return true
}

func RelayTransferTxListBytes(txs map[string]*bank.BankData) ([]byte, error) {
	r := &bank.RelayTransferTxList{
		Txs: txs,
	}
	return types.MustProtoBytes(r), nil
}
func RelayTransferTxListFromBytes(bz []byte) (map[string]*bank.BankData, error) {
	var txs = new(bank.RelayTransferTxList)
	if err := proto.Unmarshal(bz, txs); err != nil {
		return nil, err
	}
	return txs.Txs, nil
}
