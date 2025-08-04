package main

import (
	"fmt"
	"testing"
)

func TestUnmarshalTOML(t *testing.T) {
	cfg, err := GetConfig("./mytestnet/192.168.200.49/node1/config/config.toml")
	if err != nil {
		panic(err)
	}
	fmt.Println(cfg)
}
