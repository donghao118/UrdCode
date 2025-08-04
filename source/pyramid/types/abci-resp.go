package types

import (
	prototypes "emulator/proto/pyramid/types"

	"google.golang.org/protobuf/proto"
)

const (
	CodeTypeNone = iota
	CodeTypeOK
	CodeTypeEncodingError
	CodeTypeAbort
	CodeTypeUnknownError
)

type ABCIPreExecutionResponseB struct {
	Code            int8
	CrossShardDatas [][]byte
	BehaviorErrors  []error
}

func (resp *ABCIPreExecutionResponseB) IsOK() bool { return resp.Code == CodeTypeOK }

type ABCIPreExecutionResponseI struct {
	Code           int8
	BehaviorErrors []error
}

func (resp *ABCIPreExecutionResponseI) IsOK() bool { return resp.Code == CodeTypeOK }

type ABCIExecutionResponse struct {
	Receipts []*ABCIExecutionReceipt
}

type ABCIExecutionReceipt struct {
	Code    int8
	Log     string
	Info    string
	Options map[string]string
}

func (r *ABCIExecutionReceipt) IsOK() bool { return r.Code == CodeTypeOK }

func (r *ABCIExecutionReceipt) ToProto() *prototypes.ABCIExecutionReceipt {
	return &prototypes.ABCIExecutionReceipt{
		Code:    int32(r.Code),
		Log:     r.Log,
		Info:    r.Info,
		Options: r.Options,
	}
}
func NewABCIReceiptFromProto(p *prototypes.ABCIExecutionReceipt) *ABCIExecutionReceipt {
	return &ABCIExecutionReceipt{
		Code:    int8(p.Code),
		Log:     p.Log,
		Info:    p.Info,
		Options: p.Options,
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
