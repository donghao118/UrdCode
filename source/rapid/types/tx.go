package types

import (
	"bytes"
	tmhash "emulator/crypto/hash"
	merkle "emulator/crypto/merkle"
	"errors"
)

type Tx []byte

type Txs []Tx

func (tx Tx) Hash() []byte { return tmhash.Sum(tx) }
func (tx Tx) Key() string  { return string(tx) }
func (tx Tx) Size() int {return len(tx)}

func (txs Txs) Hash() []byte {
	txBzs := make([][]byte, len(txs))
	for i := 0; i < len(txs); i++ {
		txBzs[i] = txs[i].Hash()
	}
	return merkle.HashFromByteSlices(txBzs)
}

func (txs Txs) Index(tx Tx) int {
	for i := range txs {
		if bytes.Equal(txs[i], tx) {
			return i
		}
	}
	return -1
}

func (txs Txs) IndexByHash(hash []byte) int {
	for i := range txs {
		if bytes.Equal(txs[i].Hash(), hash) {
			return i
		}
	}
	return -1
}

func (txs Txs) Proof(i int) TxProof {
	l := len(txs)
	bzs := make([][]byte, l)
	for i := 0; i < l; i++ {
		bzs[i] = txs[i].Hash()
	}
	root, proofs := merkle.ProofsFromByteSlices(bzs)

	return TxProof{
		RootHash: root,
		Data:     txs[i],
		Proof:    *proofs[i],
	}
}
func (txs Txs) Size() int {
	return len(txs)
}

func (txs Txs) ToProto() [][]byte {
	p := make([][]byte, txs.Size())
	for i, v := range txs {
		p[i] = []byte(v)
	}
	return p
}
func NewTxsFromProto(p [][]byte) Txs {
	txs := make([]Tx, len(p))
	for i, v := range p {
		txs[i] = Tx(v)
	}
	return txs
}

// TxProof represents a Merkle proof of the presence of a transaction in the Merkle tree.
type TxProof struct {
	RootHash []byte       `json:"root_hash"`
	Data     Tx           `json:"data"`
	Proof    merkle.Proof `json:"proof"`
}

func (tp TxProof) Leaf() []byte {
	return tp.Data.Hash()
}

func (tp TxProof) Validate(dataHash []byte) error {
	if !bytes.Equal(dataHash, tp.RootHash) {
		return errors.New("proof matches different data hash")
	}
	if tp.Proof.Index < 0 {
		return errors.New("proof index cannot be negative")
	}
	if tp.Proof.Total <= 0 {
		return errors.New("proof total must be positive")
	}
	valid := tp.Proof.Verify(tp.RootHash, tp.Leaf())
	if valid != nil {
		return errors.New("proof is not internally consistent")
	}
	return nil
}
