package utils

import (
	"encoding/binary"
	"fmt"
)

func IntToBytes(num any) []byte {
	switch v := num.(type) {
	case int8:
		return []byte{byte(v)}
	case uint8:
		return []byte{v}
	case int16:
		bytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(bytes, uint16(v))
		return bytes
	case uint16:
		bytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(bytes, v)
		return bytes
	case int32:
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(v))
		return bytes
	case uint32:
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, v)
		return bytes
	case int64:
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, uint64(v))
		return bytes
	case uint64:
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, v)
		return bytes
	default:
		fmt.Println("Unsupported data type")
		return nil
	}
}

func BytesToInt(data []byte, dataType any) any {
	switch dataType.(type) {
	case int8:
		return int8(data[0])
	case uint8:
		return data[0]
	case int16:
		return int16(binary.LittleEndian.Uint16(data))
	case uint16:
		return binary.LittleEndian.Uint16(data)
	case int32:
		return int32(binary.LittleEndian.Uint32(data))
	case uint32:
		return binary.LittleEndian.Uint32(data)
	case int64:
		return int64(binary.LittleEndian.Uint64(data))
	case uint64:
		return binary.LittleEndian.Uint64(data)
	default:
		fmt.Println("Unsupported data type")
		return nil
	}
}

func BytesToUint32(data []byte) uint32 {
	return binary.LittleEndian.Uint32(data)
}
func Uint32ToBytes(num uint32) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, num)
	return bytes
}
