package minibank

import (
	"bytes"
	bank "emulator/proto/pyramid/abci/minibank"
	"emulator/pyramid/definition"
	"emulator/pyramid/types"
	"emulator/utils"
	"errors"
	"sort"
	"time"

	"google.golang.org/protobuf/proto"
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
	tos []string, toMoney []uint32) *bank.TransferTx {
	return &bank.TransferTx{
		From:      froms,
		To:        tos,
		FromMoney: fromMoney,
		ToMoney:   toMoney,
		Time:      utils.ThirdPartyProtoTime(time.Now()),
	}
}

func NewTransferTxMustLen(froms []string, fromMoney []uint32,
	tos []string, toMoney []uint32,
	mustLen int) *bank.TransferTx {
	bankProtoer := &bank.TransferTx{
		From:      froms,
		To:        tos,
		FromMoney: fromMoney,
		ToMoney:   toMoney,
		Time:      utils.ThirdPartyProtoTime(time.Now()),
	}
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
		return errors.New("From/FromMoney and To/ToMoney have different lengths")
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
		return errors.New("The sum of FromMoney and ToMoney is not equal")
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

func NewCrossShardDataFromBytes(bz []byte) (map[string]uint32, bool, error) {
	var bd = new(bank.BankData)
	if err := proto.Unmarshal(bz, bd); err != nil {
		return nil, false, err
	}
	outMap := map[string]uint32{}
	for i, key := range bd.Keys {
		outMap[key] = bd.Values[i]
	}
	return outMap, bd.OK, nil
}
func CrossShardDataBytes(data map[string]uint32, ok bool) []byte {
	keyList, valueList := make([]string, 0, len(data)), make([]uint32, 0, len(data))
	for key := range data {
		keyList = append(keyList, key)
	}
	sort.Strings(keyList)
	for _, key := range keyList {
		valueList = append(valueList, data[key])
	}
	bd := &bank.BankData{
		Keys:   keyList,
		Values: valueList,
		OK:     ok,
	}
	return types.MustProtoBytes(bd)
}

func NewTransferTxFromBytes(tx []byte) (*bank.TransferTx, error) {
	ptx := new(bank.TransferTx)
	if err := proto.Unmarshal(tx[4:], ptx); err != nil {
		return nil, err
	}
	return ptx, nil
}
