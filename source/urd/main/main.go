package main

import (
	"flag"
	"fmt"
	"path/filepath"
)

// ./ours --method=example  --root=.
// ./ours --method=generate --config=./example-shard-config.json --root=./mytestnet
// ./ours --method=start --root=./mytestnet --start-time=0:53  --wait-time="20s"

func main() {
	rootDir := flag.String("root", ".", "Root directory")
	jsonDir := flag.String("config", "./example-shard-config.json", "The JSON file of sharding topology structure")
	enable_pipeline := flag.Bool("enable-pipeline", true, "Choose false to start a CoCSV, and true for Urd")

	var startTimeStr, method, waitTime string
	flag.StringVar(&startTimeStr, "start-time", "", "Program startup time, in the format of HH:MM")
	flag.StringVar(&waitTime, "wait-time", "10", "System sleep time delay, the unit is seconds, please enter a positive integer")
	flag.StringVar(&method, "method", "start", "The command to be used")
	flag.Parse()

	fmt.Println(*rootDir)

	if method == "generate" {
		GenerateConfigFiles(*jsonDir, *rootDir)
	} else if method == "start" {
		InitNode(*rootDir, startTimeStr, waitTime, *enable_pipeline)
	} else if method == "example" {
		config := ExampleShardConfig()
		config.WriteJSONToFile(filepath.Join(*rootDir, "example-shard-config.json"))
	} else {
		panic("Undefined command")
	}
}
