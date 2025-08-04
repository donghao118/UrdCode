package types

import (
	"bytes"
	"emulator/crypto/hash"
	"emulator/crypto/merkle"
	pbtypes "emulator/proto/pyramid/types"
	"emulator/utils"
	"encoding/hex"
	"fmt"

	"google.golang.org/protobuf/proto"
)

var (
	DuplicatedPartPassError = fmt.Errorf("Duplicated-Part")
)

type Part struct {
	ChainID string
	Height  int64
	Round   int
	Bytes   []byte
	Proof   *merkle.Proof
}

func (p *Part) Index() int64 { return p.Proof.Index }

func (p *Part) Verify(root []byte, total int64, chainID string) bool {
	return total == p.Proof.Total && chainID == p.ChainID && p.Proof.Verify(root, p.Bytes) == nil
}

func (p *Part) ValidateBasic() error {
	if p.Height < 0 {
		return fmt.Errorf("Part.Height cannot be negative")
	}
	if p.Round < 0 {
		return fmt.Errorf("Part.Round cannot be negative")
	}
	if len(p.Bytes) == 0 {
		return fmt.Errorf("There is no point in Part.Bytes being empty")
	}
	if len(p.ChainID) == 0 {
		return fmt.Errorf("An empty Part.ChainID doesn't make sense")
	}
	return p.Proof.ValidateBasic()
}

func (p *Part) ToProto() *pbtypes.Part {
	return &pbtypes.Part{
		Height:  p.Height,
		Round:   int32(p.Round),
		Bytes:   p.Bytes,
		Proof:   p.Proof.ToProto(),
		ChainID: p.ChainID,
	}
}
func (p *Part) ProtoBytes() []byte {
	return MustProtoBytes(p.ToProto())
}

func NewPartFromProto(pp *pbtypes.Part) *Part {
	return &Part{
		Height:  pp.Height,
		Round:   int(pp.Round),
		Bytes:   pp.Bytes,
		Proof:   merkle.NewProofFromProto(pp.Proof),
		ChainID: pp.ChainID,
	}
}

func NewPartFromBytes(bz []byte) *Part {
	pp := new(pbtypes.Part)
	if err := proto.Unmarshal(bz, pp); err != nil {
		return nil
	} else {
		return NewPartFromProto(pp)
	}
}

type PartSetHeader struct {
	Height  int64
	Round   int
	Total   int64
	Root    []byte
	ChainID string
}

func (p *PartSetHeader) ToProto() *pbtypes.PartSetHeader {
	return &pbtypes.PartSetHeader{
		Total:   p.Total,
		Root:    p.Root,
		Height:  p.Height,
		Round:   int32(p.Round),
		ChainID: p.ChainID,
	}
}
func (p *PartSetHeader) ProtoBytes() []byte {
	return MustProtoBytes(p.ToProto())
}
func NewPartSetHeaderFromProto(p *pbtypes.PartSetHeader) *PartSetHeader {
	return &PartSetHeader{
		Total:   p.Total,
		Root:    p.Root,
		Height:  p.Height,
		Round:   int(p.Round),
		ChainID: p.ChainID,
	}
}
func NewPartSetHeaderFromBytes(bz []byte) *PartSetHeader {
	p := new(pbtypes.PartSetHeader)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil
	} else {
		return NewPartSetHeaderFromProto(p)
	}
}
func (p *PartSetHeader) Equal(other *PartSetHeader) bool {
	return p.ChainID == other.ChainID &&
		p.Height == other.Height &&
		p.Round == other.Round &&
		p.Total == other.Total &&
		bytes.Equal(p.Root, other.Root)
}
func (p *PartSetHeader) ValidateBasic() error {
	if p.Height < 0 {
		return fmt.Errorf(fmt.Sprintf("PartSetHeader Height Error: %d", p.Height))
	}
	if p.Round < 0 {
		return fmt.Errorf(fmt.Sprintf("PartSetHeader Round Error: %d", p.Round))
	}
	if p.Total < 0 {
		return fmt.Errorf(fmt.Sprintf("PartSetHeader Total Error: %d", p.Round))
	}
	if len(p.ChainID) == 0 {
		return fmt.Errorf(fmt.Sprintf("PartSetHeader ChainID Error: nil ChainID"))
	}
	if len(p.Root) != hash.HashSize {
		return fmt.Errorf(fmt.Sprintf("PartSetHeader Root Error: %s is not a hash value", hex.EncodeToString(p.Root)))
	}
	return nil
}

type PartSet struct {
	Header          *PartSetHeader
	Parts           []*Part
	count           int
	BlockHeaderHash []byte
}

func NewPartSet(h *PartSetHeader, headerHash []byte) *PartSet {
	return &PartSet{
		Header:          h,
		Parts:           make([]*Part, h.Total),
		count:           0,
		BlockHeaderHash: headerHash,
	}
}

func PartSetFromBlock(b *Block, maxPartSize int, round int) *PartSet {
	bzList := utils.SplitByteArray(b.ProtoBytes(), maxPartSize)
	hs, proof := merkle.ProofsFromByteSlices(bzList)
	parts := make([]*Part, len(bzList))
	chainid, height := b.ChainID, b.Height
	for i, bz := range bzList {
		parts[i] = &Part{
			ChainID: chainid,
			Height:  height,
			Round:   round,
			Bytes:   bz,
			Proof:   proof[i],
		}
	}
	header := &PartSetHeader{
		Height:  height,
		Round:   round,
		Total:   int64(len(parts)),
		Root:    hs,
		ChainID: chainid,
	}
	return &PartSet{
		Header:          header,
		Parts:           parts,
		count:           len(parts),
		BlockHeaderHash: b.Hash(),
	}
}

func (ps *PartSet) AddPart(p *Part) error {
	if i := p.Index(); i < ps.Header.Total && ps.Parts[i] != nil {
		return DuplicatedPartPassError
	}
	if ps.Header.Height != p.Height {
		return fmt.Errorf(fmt.Sprintf("Part does not match PartSetHeader: Height(%d)! = %d"), ps.Header.Height, p.Height)
	}
	if ps.Header.Round != p.Round {
		return fmt.Errorf(fmt.Sprintf("Part does not match PartSetHeader: Round(%d)! = %d"), ps.Header.Round, p.Round)
	}
	if !p.Verify(ps.Header.Root, ps.Header.Total, ps.Header.ChainID) {
		return fmt.Errorf("The Merkle validation of Part failed")
	}
	ps.Parts[p.Index()] = p
	ps.count++
	return nil
}

func (ps *PartSet) IsComplete() bool { return ps.Header.Total == int64(ps.count) }
func (ps *PartSet) GenBlock() (*Block, error) {
	if !ps.IsComplete() {
		return nil, nil
	}
	blockBytes := make([][]byte, ps.count)
	for i, part := range ps.Parts {
		blockBytes[i] = part.Bytes
	}
	bz := bytes.Join(blockBytes, nil)
	if block := NewBlockFromBytes(bz); block == nil {
		return nil, fmt.Errorf("Block deserialization error")
	} else if block.Header.ChainID != ps.Header.ChainID {
		return nil, fmt.Errorf("Block.ChainID does not match")
	} else if block.Header.Height != ps.Header.Height {
		return nil, fmt.Errorf("Block.Height does not match")
	} else if err := block.ValidateBasic(); err != nil {
		return block, err
	} else if !bytes.Equal(block.Hash(), ps.BlockHeaderHash) {
		return block, fmt.Errorf("Block.HeaderHash does not match")
	} else {
		return block, nil
	}
}
