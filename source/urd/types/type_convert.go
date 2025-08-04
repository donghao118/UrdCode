package types

import (
	"emulator/utils"
	"time"
)

func mustBytes(s any) []byte {
	switch s := s.(type) {
	case int:
		return utils.IntToBytes(s)
	case int32:
		return utils.IntToBytes(s)
	case int64:
		return utils.IntToBytes(s)
	case string:
		return []byte(s)
	case byte:
		return []byte{s}
	case []byte:
		return s
	case time.Time:
		return utils.MustProtoBytes(utils.ThirdPartyProtoTime(s))
	default:
		panic("convert mustBytes error!")
	}
}
