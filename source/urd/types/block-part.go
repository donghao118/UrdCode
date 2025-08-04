package types

import (
	"bytes"
	"emulator/crypto/hash"
	"emulator/crypto/merkle"
	pbtypes "emulator/proto/urd/types"
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
	View    int64
	Round   int32
	Bytes   []byte
	Proof   *merkle.Proof
}

func (p *Part) Index() int64 { return p.Proof.Index }

func (p *Part) Verify(root []byte, total int64, chainID string) error {
	if err := p.Proof.Verify(root, p.Bytes); err != nil {
		return err
	}
	if total != p.Proof.Total || chainID != p.ChainID {
		return fmt.Errorf("%d!=%d || %s != %s", total, p.Proof.Total, chainID, p.ChainID)
	}
	return nil
}

func (p *Part) ValidateBasic() error {
	if p.View < 0 {
		return fmt.Errorf("Part.View is non-negative")
	}
	if p.Round < 0 {
		return fmt.Errorf("Part.Round is non-negative")
	}
	if len(p.Bytes) == 0 {
		return fmt.Errorf("Part.Bytes should not be NULL")
	}
	if len(p.ChainID) == 0 {
		return fmt.Errorf("Part.ChainID should not be NULL")
	}
	return p.Proof.ValidateBasic()
}

func (p *Part) ToProto() *pbtypes.Part {
	return &pbtypes.Part{
		View:    p.View,
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
		View:    pp.View,
		Round:   pp.Round,
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
	View    int64
	Round   int32
	Total   int64
	Root    []byte
	ChainID string
}

func (p *PartSetHeader) ToProto() *pbtypes.PartSetHeader {
	return &pbtypes.PartSetHeader{
		Total:   p.Total,
		Root:    p.Root,
		View:    p.View,
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
		View:    p.View,
		Round:   p.Round,
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
		p.View == other.View &&
		p.Round == other.Round &&
		p.Total == other.Total &&
		bytes.Equal(p.Root, other.Root)
}
func (p *PartSetHeader) ValidateBasic() error {
	if p.View < 0 {
		return fmt.Errorf(fmt.Sprintf("PartSetHeader Height Error: %d", p.View))
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
		return fmt.Errorf(fmt.Sprintf("PartSetHeader Root Error: %s is not a standard hash", hex.EncodeToString(p.Root)))
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

func PartSetFromBlock(b *Block, maxPartSize int, round int32) *PartSet {
	bzList := utils.SplitByteArray(b.ProtoBytes(), maxPartSize)
	hs, proof := merkle.ProofsFromByteSlices(bzList)
	parts := make([]*Part, len(bzList))
	chainid, view := b.ChainID, b.View
	for i, bz := range bzList {
		parts[i] = &Part{
			ChainID: chainid,
			View:    view,
			Round:   round,
			Bytes:   bz,
			Proof:   proof[i],
		}
	}
	header := &PartSetHeader{
		View:    view,
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
	if ps.Header.View != p.View {
		return fmt.Errorf(fmt.Sprintf("Part and PartSetHeader is not consistent：View(%d) != %d"), ps.Header.View, p.View)
	}
	if ps.Header.Round != p.Round {
		return fmt.Errorf(fmt.Sprintf("Part and PartSetHeader is not consistent：Round(%d) != %d"), ps.Header.Round, p.Round)
	}
	if err := p.Verify(ps.Header.Root, ps.Header.Total, ps.Header.ChainID); err != nil {
		return fmt.Errorf("Fail to validate Merkle for Part: ", err)
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
		return nil, fmt.Errorf("Block serizable error")
	} else if block.Header.ChainID != ps.Header.ChainID {
		return nil, fmt.Errorf("Block.ChainID is not consistent")
	} else if block.Header.View != ps.Header.View {
		return nil, fmt.Errorf("Block.Height is not consistent")
	} else if err := block.ValidateBasic(); err != nil {
		return block, err
	} else if !bytes.Equal(block.Hash(), ps.BlockHeaderHash) {
		return block, fmt.Errorf("Block.HeaderHash is not consistent: %s != %s",
			hex.EncodeToString(block.Hash()), hex.EncodeToString(ps.BlockHeaderHash))
	} else {
		return block, nil
	}
}
