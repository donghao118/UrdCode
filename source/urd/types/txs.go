package types

import (
	"emulator/crypto/hash"
	"emulator/crypto/merkle"
	prototypes "emulator/proto/urd/types"
	"encoding/hex"
	"fmt"
)

func TxHash(tx []byte) []byte {
	return hash.Sum(tx)
}

func TxKey(tx []byte) string {
	return hex.EncodeToString(TxHash(tx))
}

type Txs [][]byte

func (ptxs Txs) Hash() []byte {
	return merkle.HashFromByteSlices(ptxs)
}
func (ptxs Txs) Index(i int) []byte {
	return ptxs[i]
}
func (ptxs Txs) Size() int {
	return len(ptxs)
}
func (ptxs Txs) ToProto() *prototypes.Txs {
	return &prototypes.Txs{
		Txs: ptxs,
	}
}
func NewTxsFromProto(pro *prototypes.Txs) Txs {
	if pro == nil {
		return Txs{}
	}
	return Txs(pro.Txs)
}

type OutputTxsProof struct {
	BlockProof *merkle.Proof
	OPTProof   *merkle.Proof
}

func (optp *OutputTxsProof) Verify(root []byte, output_txs Txs) error {
	if optp == nil {
		return nil
	}
	opt_root := optp.OPTProof.ComputeRootHash()
	opthash := output_txs.Hash()
	if err := optp.OPTProof.Verify(opt_root, opthash); err != nil {
		return fmt.Errorf("OPTProof.Verify failed: %w", err)
	}
	if err := optp.BlockProof.Verify(root, opt_root); err != nil {
		return fmt.Errorf("BlockProof.Verify failed: %w", err)
	}
	return nil
}

func (optp *OutputTxsProof) ToProto() *prototypes.OutputTxsProof {
	if optp == nil {
		return nil
	}
	return &prototypes.OutputTxsProof{
		BlockProof: optp.BlockProof.ToProto(),
		OPTProof:   optp.OPTProof.ToProto(),
	}
}

func NewOutputTxsProofFromProto(pro *prototypes.OutputTxsProof) *OutputTxsProof {
	if pro == nil {
		return nil
	}
	return &OutputTxsProof{
		BlockProof: merkle.NewProofFromProto(pro.BlockProof),
		OPTProof:   merkle.NewProofFromProto(pro.OPTProof),
	}
}

func (optp *OutputTxsProof) Hash() []byte {
	return merkle.HashFromByteSlices(
		[][]byte{
			optp.BlockProof.ContentHash(),
			optp.OPTProof.ContentHash(),
		},
	)
}

func (block *Block) GetOutputTxsProofOf(index int) *OutputTxsProof {
	otp := new(OutputTxsProof)
	otp.BlockProof = block.GetOutputTxsProof()
	_, proofs := merkle.ProofsFromByteSlices(block.OPT)
	otp.OPTProof = proofs[index]
	return otp
}
