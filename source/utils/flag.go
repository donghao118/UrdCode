package utils

import "fmt"

const (
	CodeTypeNil = iota
	CodeTypeOK
	CodeTypeReject
)

var (
	ErrInvalidVoteCode       = fmt.Errorf("invalid vote code")
	ErrInvalidValidatorIndex = fmt.Errorf("invalid validator index")
	ErrInvalidHeight         = fmt.Errorf("invalid negative height")
	ErrInvalidView           = fmt.Errorf("invalid negative view")
	ErrInvalidRound          = fmt.Errorf("invalid negative round")

	ErrInvalidSign    = fmt.Errorf("invalid SIGN")
	ErrDuplicatedVote = fmt.Errorf("duplicated vote")
)

func StrIn(s string, strs []string) bool {
	for _, str := range strs {
		if s == str {
			return true
		}
	}
	return false
}

func StrEqual(str1, str2 []string) bool {
	if len(str1) != len(str2) {
		return false
	}
	for i := range str1 {
		if str1[i] != str2[i] {
			return false
		}
	}
	return true
}
