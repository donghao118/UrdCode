package types

import (
	tmhash "emulator/crypto/hash"
	"emulator/crypto/merkle"
	"time"

	"emulator/utils"

	gogotypes "github.com/gogo/protobuf/types"
	"google.golang.org/protobuf/proto"

	pbtypes "emulator/proto/pyramid/types"
)

const (
	BLOCKTYPE_NONE = iota
	BLOCKTYPE_I
	BLOCKTYPE_B

	BLOCKTYPE_InnerShard
	BLOCKTYPE_CrossShard
	BLOCKTYPE_BCommitBlock
	BLOCKTYPE_ICommitBlock
	BLOCKTYPE_IAcceptBlock
)

type Header struct {
	BlockType int8

	HashPointer []byte

	ChainID string
	Height  int64
	Time    time.Time

	BodyRoot           []byte
	CrossShardBodyRoot []byte
	StateRoot          []byte
	ReceiptRoot        []byte
	RelationRoot       []byte

	//ProposerAddr []byte
}

func (h *Header) Hash() []byte {
	pbt, err := gogotypes.StdTimeMarshal(h.Time)
	if err != nil {
		return nil
	}

	byteList := [][]byte{
		utils.IntToBytes(h.BlockType),

		h.HashPointer,

		[]byte(h.ChainID),
		utils.IntToBytes(h.Height),
		pbt,

		h.BodyRoot,
		h.CrossShardBodyRoot,
		h.StateRoot,
		h.ReceiptRoot,
		h.RelationRoot,

		//h.ProposerAddr,
	}
	return merkle.HashFromByteSlices(byteList)
}

func (h *Header) ToProto() *pbtypes.Header {
	return &pbtypes.Header{
		BlockType: int32(h.BlockType),

		HashPointer: h.HashPointer,

		ChainID: h.ChainID,
		Height:  h.Height,
		Time:    utils.ThirdPartyProtoTime(h.Time),

		BodyRoot:           h.BodyRoot,
		CrossShardBodyRoot: h.CrossShardBodyRoot,
		StateRoot:          h.StateRoot,
		ReceiptRoot:        h.ReceiptRoot,
		RelationRoot:       h.RelationRoot,

		//ProposerAddr: h.ProposerAddr,
	}
}
func (h *Header) ProtoBytes() []byte {
	return MustProtoBytes(h.ToProto())
}

func NewHeaderFromProto(headerProto *pbtypes.Header) *Header {
	return &Header{
		BlockType:          int8(headerProto.BlockType),
		HashPointer:        headerProto.HashPointer,
		ChainID:            headerProto.ChainID,
		Height:             headerProto.Height,
		Time:               utils.ThirdPartyUnmarshalTime(headerProto.Time),
		BodyRoot:           headerProto.BodyRoot,
		CrossShardBodyRoot: headerProto.CrossShardBodyRoot,
		StateRoot:          headerProto.StateRoot,
		ReceiptRoot:        headerProto.ReceiptRoot,
		RelationRoot:       headerProto.RelationRoot,
		//ProposerAddr:       headerProto.ProposerAddr,
	}
}

func NewHeaderFromBytes(bz []byte) *Header {
	var h = new(pbtypes.Header)
	if err := proto.Unmarshal(bz, h); err != nil {
		return nil
	} else {
		return NewHeaderFromProto(h)
	}
}

type Relation struct {
	RelatedCommitStatus [][]byte
	RelatedHash         [][]byte
	MyCommitStatus      []byte

	//LastCommit Commit
}

func (r *Relation) Hash() []byte {
	byteList := [][]byte{
		merkle.HashFromByteSlices(r.RelatedCommitStatus),
		merkle.HashFromByteSlices(r.RelatedHash),
		r.MyCommitStatus,
		//r.LastCommit.Hash(),
	}
	return merkle.HashFromByteSlices(byteList)
}

func (r *Relation) ToProto() *pbtypes.Relation {
	return &pbtypes.Relation{
		RelatedCommitStatus: r.RelatedCommitStatus,
		RelatedHash:         r.RelatedHash,
		MyCommitStatus:      r.MyCommitStatus,
		//LastCommit:          r.LastCommit.ToProto(),
	}
}

func (r *Relation) ProtoBytes() []byte { return MustProtoBytes(r.ToProto()) }

func NewRelationFromProto(relationProto *pbtypes.Relation) *Relation {
	return &Relation{
		RelatedCommitStatus: relationProto.RelatedCommitStatus,
		RelatedHash:         relationProto.RelatedHash,
		MyCommitStatus:      relationProto.MyCommitStatus,
		//LastCommit:          *NewCommitFromProto(relationProto.LastCommit),
	}
}

func NewRelationFromBytes(bz []byte) *Relation {
	var r = new(pbtypes.Relation)
	if err := proto.Unmarshal(bz, r); err != nil {
		return nil
	} else {
		return NewRelationFromProto(r)
	}
}

type CrossShardBody struct {
	CrossShardTxs   Txs
	CrossShardDatas [][]byte
}

func (b *CrossShardBody) Hash() []byte {
	byteList := [][]byte{
		b.CrossShardTxs.Hash(),
		merkle.HashFromByteSlices(b.CrossShardDatas),
	}
	return merkle.HashFromByteSlices(byteList)
}
func (b *CrossShardBody) ToProto() *pbtypes.CrossShardBody {
	return &pbtypes.CrossShardBody{
		Txs:             b.CrossShardTxs.ToProto(),
		CrossShardDatas: b.CrossShardDatas,
	}
}
func NewCrossShardBodyFromProto(p *pbtypes.CrossShardBody) *CrossShardBody {
	return &CrossShardBody{
		CrossShardTxs:   NewTxsFromProto(p.Txs),
		CrossShardDatas: p.CrossShardDatas,
	}
}

type Body struct {
	BodyTxs Txs
}

func (b *Body) Hash() []byte {
	return b.BodyTxs.Hash()
}
func (b *Body) ToProto() *pbtypes.Body {
	return &pbtypes.Body{
		Txs: b.BodyTxs.ToProto(),
	}
}
func NewBodyFromProto(p *pbtypes.Body) *Body {
	return &Body{
		BodyTxs: NewTxsFromProto(p.Txs),
	}
}

type Commit struct {
	Height    int64
	Round     int
	BlockHash []byte
	Sigs      [][]byte

	hash []byte
}

func (c *Commit) Hash() []byte {
	if c.hash != nil {
		return c.hash
	}
	byteList := [][]byte{
		utils.IntToBytes(c.Height),
		utils.IntToBytes(c.Round),
		c.BlockHash,
		merkle.HashFromByteSlices(c.Sigs),
	}
	return merkle.HashFromByteSlices(byteList)
}

func (c *Commit) ToProto() *pbtypes.Commit {
	return &pbtypes.Commit{
		Height:    c.Height,
		Round:     int32(c.Round),
		BlockHash: c.BlockHash,
		Sigs:      c.Sigs,
	}
}

func (c *Commit) ProtoBytes() []byte { return MustProtoBytes(c.ToProto()) }
func NewCommitFromProto(commitProto *pbtypes.Commit) *Commit {
	return &Commit{
		Height:    commitProto.Height,
		Round:     int(commitProto.Round),
		BlockHash: commitProto.BlockHash,
		Sigs:      commitProto.Sigs,
	}
}
func NewCommitFromBytes(bz []byte) *Commit {
	var c = new(pbtypes.Commit)
	if err := proto.Unmarshal(bz, c); err != nil {
		return nil
	} else {
		return NewCommitFromProto(c)
	}
}

type Block struct {
	Header
	Body
	CrossShardBody
	Relation

	blockhash []byte
}

func (b *Block) fillHeader() {
	if len(b.Header.BodyRoot) != tmhash.HashSize {
		b.Header.BodyRoot = b.Body.Hash()
	}
	if len(b.Header.RelationRoot) != tmhash.HashSize {
		b.Header.ReceiptRoot = b.Relation.Hash()
	}
	if len(b.Header.CrossShardBodyRoot) != tmhash.HashSize {
		b.Header.CrossShardBodyRoot = b.CrossShardBody.Hash()
	}
}

func (b *Block) Hash() []byte {
	b.fillHeader()
	if len(b.blockhash) != tmhash.HashSize {
		b.blockhash = b.Header.Hash()
	}
	return b.blockhash
}

func (b *Block) ToProto() *pbtypes.Block {
	return &pbtypes.Block{
		Header:         b.Header.ToProto(),
		Body:           b.Body.ToProto(),
		CrossShardBody: b.CrossShardBody.ToProto(),
		Relation:       b.Relation.ToProto(),
	}
}

func (b *Block) ProtoBytes() []byte { return MustProtoBytes(b.ToProto()) }

func NewBlockFromProto(p *pbtypes.Block) *Block {
	return &Block{
		Header:         *NewHeaderFromProto(p.Header),
		Body:           *NewBodyFromProto(p.Body),
		CrossShardBody: *NewCrossShardBodyFromProto(p.CrossShardBody),
		Relation:       *NewRelationFromProto(p.Relation),
	}
}
func NewBlockFromBytes(bz []byte) *Block {
	var block = new(pbtypes.Block)
	if err := proto.Unmarshal(bz, block); err != nil {
		return nil
	} else {
		return NewBlockFromProto(block)
	}
}

func (b *Block) ValidateBasic() error { return nil }
