package p2p

import (
	protop2p "emulator/proto/utils/p2p"

	"google.golang.org/protobuf/proto"
)

type Envelop struct {
	Channel_id  byte `json:"c"`
	Message     []byte `json:"m"`
	MessageType uint32 `json:"mt"`
}

func NewEnvelop(channel_id byte, message []byte, messageType uint32) *Envelop {
	return &Envelop{
		Channel_id:  channel_id,
		Message:     message,
		MessageType: messageType,
	}
}

func (e *Envelop) ToProto() *protop2p.Envelop {
	return &protop2p.Envelop{
		ChannelId:   int32(e.Channel_id),
		Message:     e.Message,
		MessageType: e.MessageType,
	}
}

func (e *Envelop) GetMessage() []byte { return e.Message }
func (e *Envelop) GetChannelID() byte { return e.Channel_id }

func NewEnvelopFromProto(envelop *protop2p.Envelop) *Envelop {
	return &Envelop{
		Channel_id:  byte(int8(envelop.ChannelId)),
		Message:     envelop.Message,
		MessageType: envelop.MessageType,
	}
}

func NewEnvelopFromBytes(bz []byte) (*Envelop, error) {
	var e protop2p.Envelop
	if err := proto.Unmarshal(bz, &e); err != nil {
		return nil, err
	}
	return NewEnvelopFromProto(&e), nil
}
