package minibank

import (
	"bufio"
	"emulator/logger/blocklogger"
	bank "emulator/proto/urd/abci/minibank"
	"emulator/utils"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	transferSize     = 4
	mustLen          = 512
	accountsPerShard = 100000
	txsPerShard      = 500000

	cross_shard_ratio float64 = 0.8

	amountMoney = []uint32{}
)

type AddTxInterface interface {
	AddTx([]byte) error
}

func init() {
	for i := 0; i < transferSize; i++ {
		amountMoney = append(amountMoney, 1)
	}
}

type Importor struct {
	mempool             AddTxInterface
	cross_shard_mempool AddTxInterface
	logger              blocklogger.BlockWriter

	rangeLists map[string]*utils.RangeList
	myChain    string
	rootDir    string

	enabled bool

	all_prefixes []string
	rander       *rand.Rand
}

var batch = 1

func NewImportorForGenerator(rangeList map[string]*utils.RangeList) *Importor {
	return NewImportor(nil, nil, nil, "", "", rangeList, true)
}

func NewImportor(
	mmp, cmmp AddTxInterface, logger blocklogger.BlockWriter,
	chain_id string, rootDir string,
	rangeLists map[string]*utils.RangeList,
	enabled bool,
) *Importor {
	im := &Importor{
		mempool:             mmp,
		cross_shard_mempool: cmmp,
		logger:              logger,

		rangeLists: rangeLists,
		myChain:    chain_id,
		rootDir:    rootDir,
		enabled:    enabled,

		all_prefixes: make([]string, 0),
		rander:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for _, rl := range im.rangeLists {
		im.all_prefixes = append(im.all_prefixes, rl.StartKey())
	}
	return im
}

func (importor *Importor) Start() error {
	if !importor.enabled {
		return nil
	}
	useless := 0

	if lines, err := readLinesFromFile(importor.rootDir); err != nil {
		return err
	} else {
		log_intervel := len(lines) / 10
		fmt.Println("[%v] init transactons\n", time.Now())
		for i, line := range lines {
			var tx *bank.TransferTx
			var txBz []byte
			var err error
			if txBz, err = hex.DecodeString(line); err != nil {
				return err
			} else if tx, err = NewTransferTxFromBytes(txBz); err != nil {
				return err
			}
			if i < 5 {
				fmt.Println(tx)
			}
			if utils.StrIn(importor.myChain, tx.Shards) {
				if len(tx.Shards) == 1 {
					if err := importor.mempool.AddTx(txBz); err != nil {
						fmt.Printf("Error adding transaction: %v\n", err)
						return err
					}
				}
				if len(tx.Shards) > 1 {
					if err := importor.cross_shard_mempool.AddTx(txBz); err != nil {
						fmt.Printf("Error adding cross-shard transaction: %v\n", err)
						return err
					}
				}
			} else {
				useless++
			}

			if (i+1)%log_intervel == 0 {
				fmt.Println("[%v] finished: %d", i+1)
			}
		}
		fmt.Println("[%v] finished: %d, useless: %d", time.Now(), len(lines)-useless, useless)
	}
	return nil
}
func generateSingleShardAccounts(prefixes []string, n int, k int, r *rand.Rand) []string {
	prefix := prefixes[r.Intn(len(prefixes))]
	return generateAccounts([]string{prefix}, n, k, r)
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

func (i *Importor) generateRandomTx(prefixes []string) string {
	gun := rand.Float64()
	var accounts []string
	if gun < cross_shard_ratio {
		// generate a cross-shard-tx
		accounts = generateAccounts(prefixes, accountsPerShard, transferSize, i.rander)
	} else {
		accounts = generateSingleShardAccounts(prefixes, accountsPerShard, transferSize, i.rander)
	}
	mid := transferSize / 2

	shardsMap := make(map[string]bool)
	for _, acc := range accounts {
		for k, v := range i.rangeLists {
			if v.Search(acc) {
				shardsMap[k] = true
			}
		}
	}
	keys := []string{}
	for k := range shardsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tx := NewTransferTxMustLen(accounts[:mid], amountMoney[:mid], accounts[mid:], amountMoney[mid:], keys, mustLen)
	bz := TransferBytes(tx)
	return hex.EncodeToString(bz)
}

func (im *Importor) GenerateTxs() []string {
	fmt.Println("all prefixes: ", im.all_prefixes)
	num := len(im.rangeLists) * txsPerShard
	out := make([]string, num)
	for i := 0; i < num; i++ {
		out[i] = im.generateRandomTx(im.all_prefixes)
	}
	return out
}

func readLinesFromFile(filePath string) ([]string, error) {
	var out []string
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		out = append(out, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
