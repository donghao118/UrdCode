package signer

import (
	"emulator/utils"
	"fmt"
	"testing"

	"github.com/herumi/bls-eth-go-binary/bls"
)

func TestBLS(t *testing.T) {

	// Generate three random private keys
	privateKey1, publicKey1, err := NewBLSKeyPair(bls.BLS12_381)
	if err != nil {
		panic(err)
	}
	privateKey2, publicKey2, err := NewBLSKeyPair(bls.BLS12_381)
	if err != nil {
		panic(err)
	}
	privateKey3, publicKey3, err := NewBLSKeyPair(bls.BLS12_381)
	if err != nil {
		panic(err)
	}

	signer1, err := NewSigner(privateKey1)
	if err != nil {
		panic(err)
	}
	signer2, err := NewSigner(privateKey2)
	if err != nil {
		panic(err)
	}
	signer3, err := NewSigner(privateKey3)
	if err != nil {
		panic(err)
	}
	msg := []byte("testxxx")

	sig1, err := signer1.Sign(msg)
	if err != nil {
		panic(err)
	}
	sig2, err := signer2.Sign(msg)
	if err != nil {
		panic(err)
	}
	sig3, err := signer3.Sign(msg)
	if err != nil {
		panic(err)
	}

	verifier, err := NewVerifier([]string{publicKey1, publicKey2, publicKey3})
	if err != nil {
		panic(err)
	}

	aggSig, err := AggregateSignatures([]string{sig2, sig3, sig1})
	if err != nil {
		panic(err)
	}

	bv := utils.NewBitVector(3)
	bv.SetIndex(0, true)
	bv.SetIndex(1, true)
	bv.SetIndex(2, true)

	fmt.Println(verifier.VerifyAggregateSignature(aggSig, msg, bv.Byte()))
}
