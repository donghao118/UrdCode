package types

import (
	prototypes "emulator/proto/urd/types"
	"sort"

	"google.golang.org/protobuf/proto"
)

const (
	CodeTypeNone = iota
	CodeTypeOK
	CodeTypeEncodingError
	CodeTypeAbort
	CodeTypeUnknownError
)

type ABCIExecutionResponse struct {
	Responses           []*ABCIExecutionReceipt
	OPTxs               []Txs
	CrossShardResponses []*ABCIExecutionReceipt
}

type ABCIPreExecutionResponse struct {
	NewCrossShardTxs [][]byte
	OPTs             []Txs
	Code             int8
}

type ABCIExecutionReceipt struct {
	Code    int8
	Log     string
	Info    string
	Options map[string]string
}

func (r *ABCIExecutionReceipt) SetRawTx(tx []byte) {
	if r.Options == nil {
		r.Options = make(map[string]string)
	}
	r.Options["rawTx"] = string(tx)
}
func (r *ABCIExecutionReceipt) GetRawTx() []byte {
	if r.Options == nil {
		return nil
	}
	tx := r.Options["rawTx"]
	return []byte(tx)
}

func (r *ABCIExecutionReceipt) IsOK() bool { return r.Code == CodeTypeOK }

func (r *ABCIExecutionReceipt) ToProto() *prototypes.ABCIExecutionReceipt {
	opKeyList, opValueList := make([]string, 0, len(r.Options)), make([]string, 0, len(r.Options))
	for key := range r.Options {
		opKeyList = append(opKeyList, key)
	}
	sort.Strings(opKeyList)
	for _, key := range opKeyList {
		opKeyList = append(opKeyList, r.Options[key])
	}
	return &prototypes.ABCIExecutionReceipt{
		Code:        int32(r.Code),
		Log:         r.Log,
		Info:        r.Info,
		OptionKey:   opKeyList,
		OptionValue: opValueList,
	}
}
func NewABCIReceiptFromProto(p *prototypes.ABCIExecutionReceipt) *ABCIExecutionReceipt {
	opMap := make(map[string]string, len(p.OptionKey))
	for i, key := range p.OptionKey {
		opMap[key] = p.OptionValue[i]
	}
	return &ABCIExecutionReceipt{
		Code:    int8(p.Code),
		Log:     p.Log,
		Info:    p.Info,
		Options: opMap,
	}
}
func (r *ABCIExecutionReceipt) ProtoBytes() []byte {
	return MustProtoBytes(r.ToProto())
}
func NewABCIReceiptFromBytes(bz []byte) (*ABCIExecutionReceipt, error) {
	p := new(prototypes.ABCIExecutionReceipt)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil, err
	} else {
		return NewABCIReceiptFromProto(p), nil
	}
}
