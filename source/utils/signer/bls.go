package signer

import (
	"emulator/utils"
	"fmt"

	"github.com/herumi/bls-eth-go-binary/bls"
)

const (
	BaseCurve = bls.BLS12_381
)

type Signer struct {
	privateKey *bls.SecretKey
}

type Verifier struct {
	publicKeys []*bls.PublicKey
}

func NewSigner(privateKeyStr string) (*Signer, error) {
	privateKey := new(bls.SecretKey)

	if err := privateKey.DeserializeHexStr(privateKeyStr); err != nil {
		return nil, fmt.Errorf("Wrong private key: " + err.Error())
	}

	return &Signer{
		privateKey: privateKey,
	}, nil
}

func NewVerifier(publicKeysStr []string) (*Verifier, error) {
	publicKeys := make([]*bls.PublicKey, len(publicKeysStr))
	for i, publicKeyStr := range publicKeysStr {
		key := new(bls.PublicKey)
		if err := key.DeserializeHexStr(publicKeyStr); err != nil {
			return nil, err
		} else {
			publicKeys[i] = key
		}
	}
	return &Verifier{
		publicKeys: publicKeys,
	}, nil
}

// This function is used to sign a given byte slice
func (s *Signer) Sign(data []byte) (string, error) {
	sig := s.privateKey.SignByte(data)
	return sig.SerializeToHexStr(), nil
}

func (s *Signer) SignType(data SignableType) (string, error) {
	return s.Sign(data.SignBytes())
}

// VerifyAggregateSignature is used to verify an aggregated signature by BLS
func (s *Verifier) VerifyAggregateSignature(aggSig string, msg []byte, bitMapBytes []byte) bool {
	bitMap := utils.NewBitArrayFromByte(bitMapBytes)
	if len(s.publicKeys) != bitMap.Size() {
		fmt.Println("public keys size not match bitmap size")
		return false
	}
	sig := new(bls.Sign)
	if err := sig.DeserializeHexStr(aggSig); err != nil {
		fmt.Println("signature error: " + err.Error())
		return false
	}
	publicKey := new(bls.PublicKey)
	for i := 0; i < bitMap.Size(); i++ {
		if bitMap.GetIndex(i) {
			if s.publicKeys[i] == nil {
				fmt.Println("public key error")
				return false
			}
			publicKey.Add(s.publicKeys[i])
		}
	}
	return sig.VerifyByte(publicKey, msg)
}

func (s *Verifier) Verify(aggSig string, msg []byte, index int) bool {
	if index >= len(s.publicKeys) || s.publicKeys[index] == nil {
		return false
	}
	sig := new(bls.Sign)
	if err := sig.DeserializeHexStr(aggSig); err != nil {
		return false
	}
	return sig.VerifyByte(s.publicKeys[index], msg)
}
func (s *Verifier) Size() int {
	return len(s.publicKeys)
}

func AggregateSignatures(sigs []string) (string, error) {
	aggSig := bls.Sign{}
	for _, sig := range sigs {
		subSign := new(bls.Sign)
		err := subSign.DeserializeHexStr(sig)
		if err != nil {
			return "", err
		}
		aggSig.Add(subSign)
	}
	return aggSig.SerializeToHexStr(), nil
}

func NewBLSKeyPair(curve int) (string, string, error) {
	if err := bls.Init(curve); err != nil {
		return "", "", err
	}

	p := bls.SecretKey{}
	p.SetByCSPRNG()

	return p.SerializeToHexStr(), p.GetPublicKey().SerializeToHexStr(), nil
}

func BLSPubkey(privateKey string) (string, error) {
	p := bls.SecretKey{}
	if err := p.DeserializeHexStr(privateKey); err != nil {
		return "", err
	}
	return p.GetPop().SerializeToHexStr(), nil
}
