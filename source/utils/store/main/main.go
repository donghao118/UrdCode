package main

import (
	"emulator/utils/store"
	"fmt"
	"os"
)

func main() {
	storePath := os.Args[1]
	name := os.Args[2]
	db := store.NewPrefixStore(name, storePath)
	defer db.Close()

	bz := 0
	iter, err := db.Database.Iterator([]byte("0"), []byte("z"))
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for iter.Valid() {
		bz += len(iter.Key()) + len(iter.Value())
		iter.Next()
	}

	fmt.Println(bz / 1024 / 1024)
}
