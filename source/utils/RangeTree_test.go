package utils

import (
	"fmt"
	"testing"
)

func TestRangeTreeAdd_Delete_Range_Search1(t *testing.T) {
	tree := NewRangeTree()

	tree.AddRange("b1", "b2")
	tree.AddRange("i1", "i2")
	tree.AddRange("i3", "i4")
	tree.AddRange("i2", "i3")

	ranges := tree.Range()
	fmt.Println(ranges)
	fmt.Println(tree.Search("i1900000255a4"))
}

func sliceEqual(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if !sliceValueEqual(v, b[i]) {
			return false
		}
	}
	return true
}

func sliceValueEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
