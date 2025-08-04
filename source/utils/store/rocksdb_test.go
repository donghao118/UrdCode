package store

import (
	dbm "emulator/libs/db"
	"fmt"
	"testing"
)

var (
	testKey   = []byte("testKey")
	testValue = []byte("testValue")
	testStore = true
)

func TestRocksDB(t *testing.T) {
	db, err := dbm.NewDB("rocksTest", dbm.RocksDBBackend, "/home/user/workspace")
	if err != nil {
		panic(err)
	}
	if testStore {
		if err := db.Set(testKey, testValue); err != nil {
			panic(err)
		}
	}
	if v, err := db.Get(testKey); err != nil {
		panic(err)
	} else {
		fmt.Println(string(v))
	}
	fmt.Println(db.Get(append(testKey, 'a')))
}
