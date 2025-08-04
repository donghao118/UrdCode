package main

import (
	"fmt"
	"testing"
)

func TestPath(t *testing.T) {
	path := "/home/user/documents"
	folderName := getLastFolderName(path)
	fmt.Println(folderName)
}
