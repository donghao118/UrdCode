package store

import (
	"emulator/utils"
	"fmt"
	"path"
	"testing"
)

func TestXXX(t *testing.T) {
	name := "abci.minibank"
	dir := "../../scripts/mytestnet/127.0.0.1/"
	node1 := "node9"
	node2 := "node12"

	CompareRange(name, path.Join(dir, node1, "database"), name, path.Join(dir, node2, "database"), "0", "9")
}

func CompareRange(name1, path1, name2, path2, prefix1, prefix2 string) {
	s1 := NewPrefixStore(name1, path1)
	s2 := NewPrefixStore(name2, path2)

	iter1, err := s1.Iterator([]byte(prefix1), []byte(prefix2))
	if err != nil {
		fmt.Println("error", err)
		return
	}
	iter2, err := s2.Iterator([]byte(prefix1), []byte(prefix2))
	if err != nil {
		fmt.Println("error", err)
		return
	}
	fmt.Println(iter1.Valid(), iter2.Valid())
	for iter1.Valid() {
		key1 := string(iter1.Key())
		v1 := utils.BytesToUint32(iter1.Value())
		key2 := string(iter2.Key())
		v2 := utils.BytesToUint32(iter2.Value())
		if key1 != key2 || v1 != v2 {
			fmt.Println("Error:", key1, v1, key2, v2)
		}
		iter1.Next()
		iter2.Next()
	}
}
