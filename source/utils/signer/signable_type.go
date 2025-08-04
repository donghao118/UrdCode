package signer

type SignableType interface {
	SignBytes() []byte
}
type HashAbleType interface {
	Hash() []byte
	ProtoBytes() []byte
}

type VerifiableType interface {
	ValidateBasic() error
	SignableType
	HashAbleType

	GetSign() string
}
