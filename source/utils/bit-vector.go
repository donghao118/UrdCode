package utils

import (
	"bytes"
	"encoding/binary"

	tmmath "emulator/libs/math"
)

type BitVector struct {
	Bits  int     `json:"bits"`  // NOTE: persisted via reflect, must be exported
	Elems []uint8 `json:"elems"` // NOTE: persisted via reflect, must be exported
}

func NewBitVector(bits int) *BitVector {
	if bits < 0 {
		return nil
	}
	return &BitVector{
		Bits:  bits,
		Elems: make([]uint8, (bits+7)/8),
	}
}

func (bV *BitVector) Byte() []byte {
	if bV == nil {
		return NewBitVector(0).Byte()
	}
	buff := new(bytes.Buffer)
	uselessBits := 8*len(bV.Elems) - bV.Bits
	if err := binary.Write(buff, binary.LittleEndian, int8(uselessBits)); err != nil {
		panic(err)
	}
	for _, e := range bV.Elems {
		binary.Write(buff, binary.LittleEndian, e)
	}
	return buff.Bytes()
}

func NewBitArrayFromByte(buf []byte) *BitVector {
	if len(buf) == 0 {
		res := NewBitVector(0)
		return res
	}
	var uselessBits int8
	var temp uint8
	reader := bytes.NewBuffer(buf)
	binary.Read(reader, binary.LittleEndian, &uselessBits)

	length := len(buf) - 1
	all := length*8 - int(uselessBits)
	bv := NewBitVector(all)
	for i := 0; i < length; i++ {
		binary.Read(reader, binary.LittleEndian, &temp)
		bv.Elems[i] = temp
	}
	return bv
}

func (bA *BitVector) Size() int {
	if bA == nil {
		return 0
	}
	return bA.Bits
}

func (bA *BitVector) GetIndex(i int) bool {
	if bA == nil {
		return false
	}
	if i >= bA.Bits {
		return false
	}
	return bA.Elems[i/8]&(uint8(1)<<uint(i%8)) > 0
}

func (bA *BitVector) CountNoneZero() int {
	count := 0
	for i := 0; i < bA.Size(); i++ {
		if bA.GetIndex(i) {
			count++
		}
	}
	return count
}

func (bA *BitVector) SetIndex(i int, v bool) bool {
	if bA == nil {
		return false
	}
	if i >= bA.Bits {
		return false
	}
	if v {
		bA.Elems[i/8] |= (uint8(1) << uint(i%8))
	} else {
		bA.Elems[i/8] &= ^(uint8(1) << uint(i%8))
	}
	return true
}

func (bA *BitVector) Copy() *BitVector {
	if bA == nil {
		return NewBitVector(0)
	}
	return bA.copy()
}

func (bA *BitVector) copy() *BitVector {
	c := make([]uint8, len(bA.Elems))
	copy(c, bA.Elems)
	return &BitVector{
		Bits:  bA.Bits,
		Elems: c,
	}
}

func (bA *BitVector) copyBits(bits int) *BitVector {
	c := make([]uint8, (bits+7)/8)
	copy(c, bA.Elems)
	return &BitVector{
		Bits:  bits,
		Elems: c,
	}
}

// Or returns a bit array resulting from a bitwise OR of the two bit arrays.
// If the two bit-arrys have different lengths, Or right-pads the smaller of the two bit-arrays with zeroes.
// Thus the size of the return value is the maximum of the two provided bit arrays.
func (bA *BitVector) Or(o *BitVector) *BitVector {
	if bA == nil && o == nil {
		return nil
	}
	if bA == nil && o != nil {
		return o.Copy()
	}
	if o == nil {
		return bA.Copy()
	}
	c := bA.copyBits(tmmath.MaxInt(bA.Bits, o.Bits))
	smaller := tmmath.MinInt(len(bA.Elems), len(o.Elems))
	for i := 0; i < smaller; i++ {
		c.Elems[i] |= o.Elems[i]
	}
	return c
}

// And returns a bit array resulting from a bitwise AND of the two bit arrays.
// If the two bit-arrys have different lengths, this truncates the larger of the two bit-arrays from the right.
// Thus the size of the return value is the minimum of the two provided bit arrays.
func (bA *BitVector) And(o *BitVector) *BitVector {
	if bA == nil || o == nil {
		return nil
	}
	return bA.and(o)
}

func (bA *BitVector) and(o *BitVector) *BitVector {
	c := bA.copyBits(tmmath.MinInt(bA.Bits, o.Bits))
	for i := 0; i < len(c.Elems); i++ {
		c.Elems[i] &= o.Elems[i]
	}
	return c
}

// Not returns a bit array resulting from a bitwise Not of the provided bit array.
func (bA *BitVector) Not() *BitVector {
	if bA == nil {
		return nil // Degenerate
	}
	return bA.not()
}

func (bA *BitVector) not() *BitVector {
	c := bA.copy()
	for i := 0; i < len(c.Elems); i++ {
		c.Elems[i] = ^c.Elems[i]
	}
	return c
}

// Sub subtracts the two bit-arrays bitwise, without carrying the bits.
// Note that carryless subtraction of a - b is (a and not b).
// The output is the same as bA, regardless of o's size.
// If bA is longer than o, o is right padded with zeroes
func (bA *BitVector) Sub(o *BitVector) *BitVector {
	if bA == nil || o == nil {
		// TODO: Decide if we should do 1's complement here?
		return nil
	}
	// output is the same size as bA
	c := bA.copyBits(bA.Bits)
	// Only iterate to the minimum size between the two.
	// If o is longer, those bits are ignored.
	// If bA is longer, then skipping those iterations is equivalent
	// to right padding with 0's
	smaller := tmmath.MinInt(len(bA.Elems), len(o.Elems))
	for i := 0; i < smaller; i++ {
		// &^ is and not in golang
		c.Elems[i] &^= o.Elems[i]
	}
	return c
}
