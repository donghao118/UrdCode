package utils

import (
	"fmt"
	"testing"
)

func TestBV(t *testing.T) {
	byte1 := []byte{0, 255, 255, 23}
	byte2 := []byte{0, 128, 31, 255}

	bv := NewBitArrayFromByte(byte1)
	bv2 := NewBitArrayFromByte(byte2)
	bv3 := bv.And(bv2)
	fmt.Println(bv3.Byte())
}
