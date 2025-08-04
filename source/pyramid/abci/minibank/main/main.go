package main

import (
	"emulator/logger/blocklogger"
	"emulator/utils"
	"emulator/utils/store"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"emulator/pyramid/abci/minibank"
	"emulator/pyramid/consensus/constypes"
	"emulator/pyramid/types"
)

func getLastFolderName(path string) string {
	return filepath.Base(path)
}
func main() {
	// ./pyramid-latency ./mytestnet/127.0.0.1/node1  b1
	rootPath := os.Args[1]
	storePath := path.Join(rootPath, "database")
	nodeName := getLastFolderName(rootPath)
	blockloggerDir := path.Join(rootPath, fmt.Sprintf("%s-blocklogger-brief.txt", nodeName))

	reader := blocklogger.NewReader(blockloggerDir)
	blockRangeA, blockRangeB, err := reader.NoneZeroPeriods()
	if err != nil {
		panic(err)
	}

	db := store.NewPrefixStore("consensus", storePath)
	defer db.Close()

	chain_id := os.Args[2]

	innerShardTxCount := 0
	crossShardTxCount := 0
	innerShardLatencyCount := time.Duration(0)
	crossShardLatencyCount := time.Duration(0)

	for i := 0; i < len(blockRangeA); i++ {
		start, end := blockRangeA[i], blockRangeB[i]
		for j := start; j <= end; j++ {
			var block, blockNext *types.Block

			if bz, err := db.GetBlockByHeight(int64(j), chain_id); err != nil {
				continue
			} else if block = types.NewBlockFromBytes(bz); block == nil {
				continue
			}
			if bz, err := db.GetBlockByHeight(int64(j+1), chain_id); err != nil || bz == nil {
				continue
			} else if blockNext = types.NewBlockFromBytes(bz); blockNext == nil {
				continue
			}
			commitTime := blockNext.Time

			switch block.BlockType {
			case types.BLOCKTYPE_InnerShard:
				for _, txBytes := range block.BodyTxs {
					tx, err := minibank.NewTransferTxFromBytes(txBytes)
					if err != nil {
						continue
					}
					innerShardLatencyCount += commitTime.Sub(utils.ThirdPartyUnmarshalTime(tx.Time))
					innerShardTxCount++
				}
			case types.BLOCKTYPE_BCommitBlock:
				cmt, err := constypes.NewMessageAcceptSetFromBytes(block.BodyTxs[0])
				if err != nil {
					panic(err)
				}
				isok := true
				for _, ma := range cmt.Accepts {
					if !ma.CollectiveSignatures.IsOK() {
						isok = false
					}
				}
				if isok {
					ublockBz, err := db.GetBlockByHash(cmt.BlockHash)
					if err != nil {
						panic(err)
					}
					ublock := types.NewBlockFromBytes(ublockBz)
					for _, txBytes := range ublock.CrossShardTxs {
						tx, err := minibank.NewTransferTxFromBytes(txBytes)
						if err != nil {
							continue
						}
						crossShardLatencyCount += commitTime.Sub(utils.ThirdPartyUnmarshalTime(tx.Time)) * time.Duration(4) / time.Duration(3)
						crossShardTxCount++
					}
				}
			}

		}
	}

	if innerShardTxCount > 0 {
		fmt.Printf("intra-shard transactions: %d Average latency: %.3f seconds \n", innerShardTxCount, innerShardLatencyCount.Seconds()/float64(innerShardTxCount))
	}
	if crossShardTxCount > 0 {
		fmt.Printf("Cross-shard transactions: %d Average latency: %.3f seconds\n", crossShardTxCount, crossShardLatencyCount.Seconds()/float64(crossShardTxCount))
	}
	if innerShardTxCount+crossShardTxCount > 0 {
		fmt.Printf("Total transactions: %d Average latency: %.3f seconds\n",
			crossShardTxCount+innerShardTxCount,
			(crossShardLatencyCount.Seconds()+innerShardLatencyCount.Seconds())/float64(innerShardTxCount+crossShardTxCount),
		)
	}
}
