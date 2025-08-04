package main

import (
	"flag"
	"fmt"
	"path/filepath"
)

// ./pyramid --method=example  --root=.
// ./pyramid --method=generate --config=./example-shard-config.json --root=./mytestnet
// ./pyramid --method=start --root=./mytestnet --start-time=0:53  --wait-time="20s"

func main() {
	rootDir := flag.String("root", ".", "Root directory")
	jsonDir := flag.String("config", "./example-shard-config.json", "Shard topology json file")
	var startTimeStr, method, waitTime string
	flag.StringVar(&startTimeStr, "start-time", "", "Program start time in HH:MM")
	flag.StringVar(&waitTime, "wait-time", "0", "System sleep time delay in seconds, please enter a positive integer")
	flag.StringVar(&method, "method", "start", "Command to use")
	flag.Parse()

	fmt.Println(*rootDir)

	if method == "generate" {
		GenerateConfigFiles(*jsonDir, *rootDir)
	} else if method == "start" {
		InitNode(*rootDir, startTimeStr, waitTime)
	} else if method == "example-shard-config" {
		config := ExampleShardConfig()
		config.WriteJSONToFile(filepath.Join(*rootDir, "example-shard-config.json"))
	} else {
		panic("Undefined command")
	}
}
