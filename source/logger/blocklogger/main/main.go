package main

import (
	"fmt"
	"os"

	reader "emulator/logger/blocklogger"
)

func main() {
	path := os.Args[1]
	rd := reader.NewReader(path)
	if rd == nil {
		panic("There exists some problems")
	}
	is, js, err := rd.NoneZeroPeriods()
	if err != nil {
		panic(err)
	}
	if len(is) == 0 || len(js) == 0 {
		fmt.Println("No new transactions")
	}
	n := len(is)
	if n > len(js) {
		n = len(js)
	}
	for i := 0; i < n; i++ {
		if js[i]-is[i] < 5 {
			continue
		}
		tps, commit_rate, err := rd.CalculateTPS(is[i], js[i])
		if err != nil {
			panic(err)
		}
		fmt.Printf("[%d-%d] : %.3f tps, %.3f %% commit\n", is[i], js[i], tps, commit_rate)
	}
}
