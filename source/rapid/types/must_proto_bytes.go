package types

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func MustProtoBytes(m protoreflect.ProtoMessage) []byte {
	if bz, err := proto.Marshal(m); err != nil {
		panic(err)
	} else {
		return bz
	}
}
