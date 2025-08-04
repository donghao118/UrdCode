package types

import (
	"bytes"
	"emulator/crypto/merkle"
	prototypes "emulator/proto/urd/types"
	"errors"
	"fmt"
)

type CommitCertificateCore struct {
	Hash           []byte
	MKProof        ProofOfIntention
	IntentionBrief IntentionBrief
}

func (core *CommitCertificateCore) ToProto() *prototypes.CommitCertificateCore {
	if core == nil {
		return nil
	}
	return &prototypes.CommitCertificateCore{
		Hash:           core.Hash,
		MKProof:        core.MKProof.ToProto(),
		IntentionBrief: core.IntentionBrief,
	}
}

func NewCommitCertificateCoreFromProto(pcore *prototypes.CommitCertificateCore) *CommitCertificateCore {
	if pcore == nil {
		return nil
	}
	return &CommitCertificateCore{
		Hash:           pcore.Hash,
		MKProof:        *NewProofOfIntentionFromProto(pcore.MKProof),
		IntentionBrief: pcore.IntentionBrief,
	}
}

func (core *CommitCertificateCore) Verify() error {
	leftHash := core.IntentionBrief.Hash()
	return core.MKProof.Verify(core.Hash, leftHash)
}

type CommitCertificate []*CommitCertificateCore

func (cc CommitCertificate) ToProto() *prototypes.CommitCertificate {
	pcc := new(prototypes.CommitCertificate)
	pcc.Cores = make([]*prototypes.CommitCertificateCore, len(cc))
	for i, core := range cc {
		pcc.Cores[i] = core.ToProto()
	}
	return pcc
}
func NewCommitCertificateFromProto(pcc *prototypes.CommitCertificate) CommitCertificate {
	var cc = make(CommitCertificate, len(pcc.Cores))
	for i, core := range pcc.Cores {
		cc[i] = NewCommitCertificateCoreFromProto(core)
	}
	return cc
}
func (cc CommitCertificate) Verify() error {
	for i, core := range cc {
		if err := core.Verify(); err != nil {
			return errors.New(fmt.Sprintf("error when verify CC(index=%d): %v", i, err))
		}
	}
	return nil
}
func (cc CommitCertificate) Hash() []byte {
	coreHashList := [][]byte{}
	for _, core := range cc {
		hash := merkle.HashFromByteSlices(
			[][]byte{
				core.Hash,
				core.IntentionBrief.Hash(),
				core.MKProof.Hash(),
			},
		)
		coreHashList = append(coreHashList, hash)
	}
	return merkle.HashFromByteSlices(coreHashList)
}

func (cc CommitCertificate) Result() (results [][]byte, Errors []bool) {
	if len(cc) == 0 {
		return
	}
	results = make([][]byte, len(cc))
	Errors = make([]bool, len(cc))
	for _, core := range cc {
		for i, bz := range core.IntentionBrief {
			if bytes.Equal(results[i], bz) || len(bz) == 0 {
				continue
			} else if len(results[i]) > 0 {
				Errors[i] = true
			} else {
				results[i] = bz
			}
		}
	}
	return
}
