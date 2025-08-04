package minibank

import (
	bank "emulator/proto/pyramid/abci/minibank"
	"emulator/rapid/definition"
	"emulator/rapid/types"
	"emulator/utils"
	"emulator/utils/p2p"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type AddTxInterface interface {
	AddTx(*types.Tx) error
}

type Importor struct {
	mempool             AddTxInterface
	cross_shard_mempool AddTxInterface

	transferSize       int
	mustLen            int
	accountNumPerShard int

	txPool             []*bank.TransferTx
	cross_shard_txPool []*bank.TransferTx

	commit_rate      float64
	cross_shard_rate float64

	isI                 bool
	MyRangeTree         *utils.RangeTree
	AllRelatedRangeTree []*utils.RangeTree

	p2pConn *p2p.Sender
	myChain string
}

var batch = 1

func NewImportor(
	mmp, cmmp AddTxInterface,
	transferSize, mustLen, accountNumPerShard int,
	commit_rate float64,
	isI bool,
	myRangeTree *utils.RangeTree,
	KeyRangeTree []*utils.RangeTree,
	p2pConn *p2p.Sender, chain string,
	cross_shard_rate float64,
) *Importor {
	return &Importor{
		mempool:             mmp,
		cross_shard_mempool: cmmp,

		transferSize:       transferSize,
		mustLen:            mustLen,
		accountNumPerShard: accountNumPerShard,

		txPool:             nil,
		cross_shard_txPool: nil,

		commit_rate:      commit_rate,
		cross_shard_rate: cross_shard_rate,

		isI:                 isI,
		MyRangeTree:         myRangeTree,
		AllRelatedRangeTree: KeyRangeTree,

		p2pConn: p2pConn,
		myChain: chain,
	}
}

func (im *Importor) StartMempool() {
	if im.isI {
		if im.mempool == nil {
			return
		}
		go func() {
			rate := int(math.Ceil(im.commit_rate))
			if rate > 0 {
				dur := time.Second / time.Duration(rate) * time.Duration(batch)
				log.Printf("I shard start mempool, broadcast time interval:%d/%v", rate, dur)
				for len(im.txPool) > 0 {
					startTime := time.Now()
					dst := batch
					if len(im.txPool) < batch {
						dst = len(im.txPool)
					}
					for _, tx := range im.txPool[:dst] {
						tx.Time = utils.ThirdPartyProtoTime(time.Now())
						btx := types.Tx(TransferBytes(tx))
						im.mempool.AddTx(&btx)
						if im.p2pConn != nil {
							im.p2pConn.SendToShard(im.myChain, p2p.ChannelIDMempool, []byte(btx), definition.TxTransfer)
						}
					}
					im.txPool = im.txPool[dst:]
					time.Sleep(time.Until(startTime.Add(dur)))
				}
				log.Println("=======================================")
				log.Println("        transaction broadcast end                  ")
				log.Println("=======================================")
			}
		}()
	} else {
		if im.mempool == nil || im.cross_shard_mempool == nil {
			return
		}
		rate1 := int(math.Ceil(im.commit_rate * (1.0 - im.cross_shard_rate)))
		if rate1 > 0 {
			dur1 := time.Second / time.Duration(rate1) * time.Duration(batch)
			log.Printf("B shard start mempool, broadcast time interval:%d/%v", rate1, dur1)
			go func() {
				for len(im.txPool) > 0 {
					startTime := time.Now()
					dst := batch
					if len(im.txPool) < batch {
						dst = len(im.txPool)
					}
					for _, tx := range im.txPool[:dst] {
						tx.Time = utils.ThirdPartyProtoTime(time.Now())
						btx := types.Tx(TransferBytes(tx))
						im.mempool.AddTx(&btx)
						if im.p2pConn != nil {
							im.p2pConn.SendToShard(im.myChain, p2p.ChannelIDMempool, []byte(btx), definition.TxTransfer)
						}
					}
					im.txPool = im.txPool[dst:]
					time.Sleep(time.Until(startTime.Add(dur1)))
				}
				log.Println("=======================================")
				log.Println("        transaction broadcast end                    ")
				log.Println("=======================================")
			}()
		}

		rate2 := int(math.Ceil(im.commit_rate * im.cross_shard_rate))
		if rate2 > 0 {
			dur2 := time.Second / time.Duration(rate2) * time.Duration(batch)
			log.Printf("B shard start mempool, broadcast time interval:%d/%v", rate2, dur2)
			go func() {
				for len(im.cross_shard_txPool) > 0 {
					startTime := time.Now()
					dst := batch
					if len(im.cross_shard_txPool) < batch {
						dst = len(im.cross_shard_txPool)
					}
					for _, tx := range im.cross_shard_txPool[:dst] {
						tx.Time = utils.ThirdPartyProtoTime(time.Now())
						btx := types.Tx(TransferBytes(tx))
						im.cross_shard_mempool.AddTx(&btx)
						if im.p2pConn != nil {
							im.p2pConn.SendToShard(im.myChain, p2p.ChannelIDCrossShardMempool, []byte(btx), definition.TxTransfer)
						}
					}
					im.cross_shard_txPool = im.cross_shard_txPool[dst:]
					time.Sleep(time.Until(startTime.Add(dur2)))
				}
				log.Println("=======================================")
				log.Println("   cross shrad transaction broadcast end              ")
				log.Println("=======================================")
			}()
		}
	}
}

func (im *Importor) RandomGenerateTx(n int) {
	fmt.Println("Generate intra-shard transactions", n)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	inputSize := im.transferSize / 2
	myPrefixRange := im.MyRangeTree.Range()
	myPrefix := make([]string, len(myPrefixRange))
	for i, preRange := range myPrefixRange {
		myPrefix[i] = preRange[0]
	}

	im.txPool = make([]*bank.TransferTx, n)

	transferMoney := make([]uint32, 0, im.transferSize)
	for i := 0; i < im.transferSize; i++ {
		transferMoney = append(transferMoney, 1)
	}

	for i := 0; i < n; i++ {
		accounts := generateAccounts(myPrefix, im.accountNumPerShard, im.transferSize, r)
		im.txPool[i] = NewTransferTxMustLen(accounts[:inputSize], transferMoney[:inputSize], accounts[inputSize:], transferMoney[inputSize:], im.mustLen)
	}
}
func (im *Importor) RandomGenerateCrossShardTx(n int) {
	fmt.Println("Generate cross-shard transactions", n)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	inputSize := im.transferSize / 2
	myPrefixRange := im.AllRelatedRangeTree
	myPrefix := make([]string, 0)
	for _, prerange := range myPrefixRange {
		myPrefix = append(myPrefix, prerange.Range()[0][0])
		fmt.Println(prerange.Range()[0][0])
	}

	im.cross_shard_txPool = make([]*bank.TransferTx, n)

	transferMoney := make([]uint32, 0, im.transferSize)
	for i := 0; i < im.transferSize; i++ {
		transferMoney = append(transferMoney, 1)
	}

	for i := 0; i < n; i++ {
		accounts := generateAccounts(myPrefix, im.accountNumPerShard, im.transferSize, r)
		im.cross_shard_txPool[i] = NewTransferTxMustLen(accounts[:inputSize], transferMoney[:inputSize], accounts[inputSize:], transferMoney[inputSize:], im.mustLen)
	}
}

func generateAccounts(prefixes []string, n int, k int, r *rand.Rand) []string {
	results := make([]string, k)
	generated := make(map[string]bool)

	for i := 0; i < k; i++ {
		prefix := prefixes[r.Intn(len(prefixes))]
		num := r.Intn(n) + 1
		numStr := strconv.Itoa(num)
		numStr = strings.Repeat("0", 32-len(prefix)-len(numStr)) + numStr
		result := prefix + numStr

		for generated[result] {
			prefix = prefixes[r.Intn(len(prefixes))]
			num = r.Intn(n) + 1
			numStr = strconv.Itoa(num)
			numStr = strings.Repeat("0", 32-len(prefix)-len(numStr)) + numStr
			result = prefix + numStr
		}

		results[i] = result
		generated[result] = true
	}

	return results
}
