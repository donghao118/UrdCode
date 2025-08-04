package types

import (
	"bytes"
	"emulator/core/hotstuff"
	"emulator/crypto/merkle"
	"emulator/utils"
	"encoding/hex"
	"fmt"
	"time"

	prototypes "emulator/proto/urd/types"

	"google.golang.org/protobuf/proto"
)

type Block struct {
	Header `json:"header"`

	PTXS          Txs                      `json:"prepare_txs"`
	CrossShardTxs Txs                      `json:"cross_shard_txs"`
	OPT           [][]byte                 `json:"output_txs_hash"`
	AggSigVote    *hotstuff.AggregatedVote `json:"agg_vote"`
	CI            *CommitIntention         `json:"commit_intention"`
	CC            CommitCertificate        `json:"commit_certificate"`
	CTXS          []Txs                    `json:"commit_txs"`
	CTXSProof     []*OutputTxsProof        `json:"commit_txs_proof"`
}

type Header struct {
	HashPointer []byte

	ChainID string
	View    int64
	Round   int32
	Time    time.Time

	PrepareRoot           []byte // of round j
	CrossShardRoot        []byte // of round j
	OutputTxsRoot         []byte // of round j
	LastAggSigRoot        []byte // of round j-1
	CommitIntentionRoot   []byte // of round j-2
	CommitCertificateRoot []byte // of round j-4
	CommitTxsListRoot     []byte // of round j-4
	StateRoot             []byte // of round j-6

	hash    []byte
	mkproof []*merkle.Proof
}

func (block *Block) String() string {
	return fmt.Sprintf("PrepareRoot: %s\nCrossShardRoot: %s\nOutputTxsRoot: %s\nLastAggSigRoot: %s\nCommitIntentionRoot: %s\nCommitCertificateRoot: %s\nCommitTxsListRoot: %s\nStateRoot: %s\n", hex.EncodeToString(block.Header.PrepareRoot),
		hex.EncodeToString(block.Header.CrossShardRoot),
		hex.EncodeToString(block.Header.OutputTxsRoot),
		hex.EncodeToString(block.Header.LastAggSigRoot),
		hex.EncodeToString(block.Header.CommitIntentionRoot),
		hex.EncodeToString(block.Header.CommitCertificateRoot),
		hex.EncodeToString(block.Header.CommitTxsListRoot),
		hex.EncodeToString(block.Header.StateRoot),
	)
}
func (block *Block) fillInHeader() {
	if len(block.Header.PrepareRoot) == 0 && block.PTXS != nil {
		block.Header.PrepareRoot = block.PTXS.Hash()
	}
	if len(block.Header.CrossShardRoot) == 0 && block.CrossShardTxs != nil {
		block.Header.CrossShardRoot = block.CrossShardTxs.Hash()
	}
	if len(block.Header.OutputTxsRoot) == 0 {
		block.Header.OutputTxsRoot = merkle.HashFromByteSlices(block.OPT)
	}
	if len(block.Header.LastAggSigRoot) == 0 && block.AggSigVote != nil {
		block.Header.LastAggSigRoot = block.AggSigVote.Hash()
	}
	if len(block.Header.CommitIntentionRoot) == 0 && block.CI != nil {
		block.Header.CommitIntentionRoot = block.CI.Hash()
	}
	if len(block.Header.CommitCertificateRoot) == 0 {
		block.Header.CommitCertificateRoot = block.CC.Hash()
	}
	if len(block.Header.CommitTxsListRoot) == 0 {
		ctxs_hash_list := [][]byte{}
		ctxs_proof_hash_list := [][]byte{}
		for _, tx := range block.CTXS {
			ctxs_hash_list = append(ctxs_hash_list, tx.Hash())
		}
		for _, proof := range block.CTXSProof {
			ctxs_proof_hash_list = append(ctxs_proof_hash_list, proof.Hash())
		}
		block.Header.CommitTxsListRoot = merkle.HashFromByteSlices(
			[][]byte{
				merkle.HashFromByteSlices(ctxs_hash_list),
				merkle.HashFromByteSlices(ctxs_proof_hash_list),
			},
		)
	}
}

func (block *Block) Hash() []byte {
	if block == nil {
		return nil
	}

	if len(block.Header.hash) == 0 {
		block.calBlockHashAndProof()
	}
	return block.Header.hash
}

func (block *Block) ProofOfIndex(i int) *merkle.Proof {
	if len(block.mkproof) == 0 {
		block.calBlockHashAndProof()
	}
	if i >= 0 && i < len(block.mkproof) {
		return block.mkproof[i]
	}
	return nil
}

func (block *Block) calBlockHashAndProof() {
	block.fillInHeader()
	packageHash := merkle.HashFromByteSlices(
		[][]byte{
			mustBytes(block.Header.ChainID),
			mustBytes(block.Header.View),
			mustBytes(block.Header.Round),
			mustBytes(block.Header.Time),

			block.Header.LastAggSigRoot,
			block.Header.CommitCertificateRoot,
		},
	)

	block.Header.hash, block.Header.mkproof = merkle.ProofsFromByteSlices(
		[][]byte{
			packageHash,
			block.Header.HashPointer, // 1
			block.Header.PrepareRoot,

			block.Header.OutputTxsRoot,       // 3
			block.Header.CommitIntentionRoot, // 4

			block.Header.CommitTxsListRoot,
			block.Header.StateRoot,
		},
	)
}
func (block *Block) GetHeaderProof() *merkle.Proof {
	return block.ProofOfIndex(1)
}
func (block *Block) GetOutputTxsProof() *merkle.Proof {
	return block.ProofOfIndex(3)
}
func (block *Block) GetCommitIntentionProof() *merkle.Proof {
	return block.ProofOfIndex(4)
}
func (header *Header) ToProto() *prototypes.Header {
	return &prototypes.Header{
		HashPointer:           header.HashPointer,
		ChainID:               header.ChainID,
		View:                  header.View,
		Round:                 header.Round,
		Time:                  utils.ThirdPartyProtoTime(header.Time),
		PrepareRoot:           header.PrepareRoot,
		OutputTxsRoot:         header.OutputTxsRoot,
		LastAggSigRoot:        header.LastAggSigRoot,
		CommitIntentionRoot:   header.CommitIntentionRoot,
		CommitCertificateRoot: header.CommitCertificateRoot,
		CommitTxsRoot:         header.CommitTxsListRoot,
		StateRoot:             header.StateRoot,
	}
}
func (b *Block) ProtoBytes() []byte { return MustProtoBytes(b.ToProto()) }
func NewHeaderFromProto(header *prototypes.Header) *Header {
	return &Header{
		HashPointer:           header.HashPointer,
		ChainID:               header.ChainID,
		View:                  header.View,
		Round:                 header.Round,
		Time:                  utils.ThirdPartyUnmarshalTime(header.Time),
		PrepareRoot:           header.PrepareRoot,
		OutputTxsRoot:         header.OutputTxsRoot,
		LastAggSigRoot:        header.LastAggSigRoot,
		CommitIntentionRoot:   header.CommitIntentionRoot,
		CommitCertificateRoot: header.CommitCertificateRoot,
		CommitTxsListRoot:     header.CommitTxsRoot,
		StateRoot:             header.StateRoot,
	}
}

func (block *Block) ToProto() *prototypes.Block {
	block.fillInHeader()
	ctxs := make([]*prototypes.Txs, len(block.CTXS))
	for i, txs := range block.CTXS {
		ctxs[i] = txs.ToProto()
	}
	commitTxsProof := make([]*prototypes.OutputTxsProof, len(block.CTXSProof))
	for i, proof := range block.CTXSProof {
		commitTxsProof[i] = proof.ToProto()
	}
	return &prototypes.Block{
		Header:        block.Header.ToProto(),
		PrepareTxs:    block.PTXS.ToProto(),
		CrossShardTxs: block.CrossShardTxs.ToProto(),
		OPT:           block.OPT,
		AggSigVote:    block.AggSigVote.ToProto(),
		CI:            block.CI.ToProto(),
		CC:            block.CC.ToProto(),
		CTXS:          ctxs,
		CTXSProof:     commitTxsProof,
	}
}

func NewBlockFromProto(block *prototypes.Block) *Block {
	ctxs := make([]Txs, len(block.CTXS))
	for i, txs := range block.CTXS {
		ctxs[i] = NewTxsFromProto(txs)
	}
	commitTxsProof := make([]*OutputTxsProof, len(block.CTXSProof))
	for i, proof := range block.CTXSProof {
		commitTxsProof[i] = NewOutputTxsProofFromProto(proof)
	}
	return &Block{
		Header:        *NewHeaderFromProto(block.Header),
		PTXS:          NewTxsFromProto(block.PrepareTxs),
		CrossShardTxs: NewTxsFromProto(block.CrossShardTxs),
		OPT:           block.OPT,
		AggSigVote: hotstuff.NewAggregatedVoteFromProto(
			block.AggSigVote,
		),
		CI:        NewCommitIntentionFromProto(block.CI),
		CC:        NewCommitCertificateFromProto(block.CC),
		CTXS:      ctxs,
		CTXSProof: commitTxsProof,
	}
}

func NewBlockFromBytes(bz []byte) *Block {
	var block = new(prototypes.Block)
	if err := proto.Unmarshal(bz, block); err != nil {
		return nil
	} else {
		return NewBlockFromProto(block)
	}
}

func (b *Block) ValidateBasic() error {
	// validate commitment intention
	if b.View > 5 {
		if len(b.CI.IntentionHash) != len(b.CI.AggregatedSignatures) {
			return fmt.Errorf("block contains %d Intention Hash but %d AggregatedSignatures",
				len(b.CI.IntentionHash), len(b.CI.AggregatedSignatures))
		}
		for i, vote := range b.CI.AggregatedSignatures {
			h1 := b.CI.IntentionHash[i]
			if h2 := GetLastHashOfAggVote(vote); !bytes.Equal(h1, h2) {
				return fmt.Errorf("commit intention of block(view=%d,round=%d) is inconsistent with its vote",
					b.View, b.Round)
			}
		}

		// validate commitment certificate
		if len(b.CC) != len(b.CI.IntentionHash) {
			return fmt.Errorf("block contains %d Commitment Certificate but %d Intention Hash",
				len(b.CC), len(b.CI.IntentionHash))
		}
		for i, core := range b.CC {
			vote := b.CI.AggregatedSignatures[i]
			if h1, h2 := core.Hash, GetLastHashOfAggVote(vote); !bytes.Equal(h1, h2) {
				return fmt.Errorf("commit certificate of block(view=%d,round=%d) is inconsistent with its vote",
					b.View, b.Round)
			}
		}
		if err := b.CC.Verify(); err != nil {
			return err
		}
	}

	// validate CTXs
	if len(b.CTXS) != len(b.CTXSProof) || len(b.CTXS) != len(b.CC) {
		return fmt.Errorf("block contains not enough CTXsProof")
	}
	intentions, errs := b.CC.Result()
	for i, intention := range intentions {
		if errs[i] {
			fmt.Println("Skipping CTXSProof validation for intention with error:", hex.EncodeToString(intention), errs[i])
			continue
		}
		fmt.Println("Validating CTXSProof for intention:", hex.EncodeToString(intention))
		if err := b.CTXSProof[i].Verify(GetLastHashOfAggVote(b.CI.AggregatedSignatures[i]), b.CTXS[i]); err != nil {
			return err
		}
	}
	return nil
}
