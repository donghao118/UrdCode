package main

import (
	"context"
	"emulator/pyramid/abci/minibank"
	pyramid "emulator/pyramid/consensus"
	"emulator/pyramid/definition"
	"emulator/pyramid/mempool"
	"emulator/pyramid/shardinfo"
	"emulator/utils"
	"emulator/utils/p2p"
	"emulator/utils/signer"
	"emulator/utils/store"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"emulator/logger/blocklogger"

	"github.com/herumi/bls-eth-go-binary/bls"
)

var (
	transferSize    = 6
	mustLen         = 512
	accountNumTotal = 1000000

	commit_rates      float64 = 33000
	cross_shard_rates float64 = 0.8

	txTotals = 1000000

	bHasData = true
)

func InitNode(rootDir string, startTimeStr string, waitTime string) {
	defer fmt.Println("test end")

	var cfg = new(Config)
	cfg.DirRoot = rootDir
	cfg, err := GetConfig(cfg.ConfigPath())
	if err != nil {
		panic(err)
	}
	cfg.DirRoot = rootDir

	priveKeyBz, err := os.ReadFile(cfg.PrivateKeyPath())
	if err != nil {
		panic(err)
	}

	if err := bls.Init(signer.BaseCurve); err != nil {
		panic(err)
	}
	privateKey := string(priveKeyBz)
	Signer, err := signer.NewSigner(privateKey)
	if err != nil {
		panic(err)
	}

	shardInfoBz, err := os.ReadFile(cfg.ShardInfoPath())
	if err != nil {
		panic(err)
	}
	var shardInfo = new(shardinfo.ShardInfo)
	if err := json.Unmarshal(shardInfoBz, shardInfo); err != nil {
		panic(err)
	}

	shardNum := len(shardInfo.PeerList)
	bNum := cfg.BShardNum
	iNum := cfg.IShardNum
	nodeNum := len(shardInfo.PeerList[cfg.ChainID])
	fmt.Println(shardNum, bNum, iNum, nodeNum)

	cross_rate := commit_rates * cross_shard_rates
	inner_rate := commit_rates - cross_rate

	crossTxNum := float64(txTotals) * cross_shard_rates
	innerTxNum := float64(txTotals) - crossTxNum

	cross_rate /= float64(bNum * nodeNum)
	crossTxNum /= float64(bNum * nodeNum)
	if bHasData {
		innerTxNum /= float64(shardNum * nodeNum)
		inner_rate /= float64(shardNum * nodeNum)
	} else {
		innerTxNum /= float64(iNum * nodeNum)
		inner_rate /= float64(iNum * nodeNum)
		if !cfg.IsI {
			innerTxNum = 0.0
			inner_rate = 0.0
		}
	}
	if cfg.IsI {
		cross_rate = 0.0
		crossTxNum = 0.0
	}

	fmt.Println(innerTxNum, crossTxNum, inner_rate, cross_rate)
	abci := createABCI(cfg, shardInfo, nil)
	defer abci.Stop()
	mempool, cross_shard_mempool := createMempool(cfg, abci)
	sender, receiver := createP2p(cfg, shardInfo)
	logger := blocklogger.NewBlockWriter(cfg.DirRoot, cfg.NodeName, cfg.ChainID)
	if err := logger.OnStart(); err != nil {
		panic(err)
	}
	defer logger.OnStop()
	consensus := createConsensus(
		cfg, shardInfo,
		Signer,
		mempool, cross_shard_mempool,
		abci, sender,
		logger,
	)

	receiver.AddChennel(consensus, p2p.ChannelIDConsensusState)
	receiver.AddChennel(mempool, p2p.ChannelIDMempool)
	receiver.AddChennel(cross_shard_mempool, p2p.ChannelIDCrossShardMempool)
	defer consensus.Stop()

	keyRangeTree, rf := createKeyRangeTree(cfg, shardInfo)
	var myKeyRangeTree *utils.RangeTree
	var relatedKeyRangeTrees []*utils.RangeTree
	if cfg.IsI {
		myKeyRangeTree = keyRangeTree
		relatedKeyRangeTrees = nil
	} else {
		myKeyRangeTree = rf[cfg.ChainID]
		for _, v := range rf {
			relatedKeyRangeTrees = append(relatedKeyRangeTrees, v)
		}
	}

	minibankAdder := minibank.NewImportor(
		mempool, cross_shard_mempool,
		transferSize, mustLen, accountNumTotal/shardNum,
		cross_rate+inner_rate, cfg.IsI,
		myKeyRangeTree, relatedKeyRangeTrees,
		sender, cfg.ChainID,
		cross_rate/(cross_rate+inner_rate),
	)
	if cfg.IsI {
		minibankAdder.RandomGenerateTx(int(math.Ceil(innerTxNum)))
	} else {
		minibankAdder.RandomGenerateCrossShardTx(int(math.Ceil(crossTxNum)))
		minibankAdder.RandomGenerateTx(int(math.Ceil(innerTxNum)))
	}

	receiver.Start()

	startTime := time_to_start(startTimeStr)
	fmt.Println(time.Until(startTime))
	time.Sleep(time.Until(startTime))

	if err := sender.Start(); err != nil {
		fmt.Println(err)
	}

	t, err := strconv.ParseInt(waitTime, 10, 32)
	if err == nil && t >= 0 {
		time.Sleep(time.Duration(t) * time.Second)
	} else {
		time.Sleep(10 * time.Second)
	}

	minibankAdder.StartMempool()
	consensus.Start()
	select {}
}

func createKeyRangeTree(cfg *Config, si *shardinfo.ShardInfo) (*utils.RangeTree, map[string]*utils.RangeTree) {
	var chain_id = cfg.ChainID
	var keyRangeTree *utils.RangeTree
	var relatedKeyRangeForest = map[string]*utils.RangeTree{}
	if cfg.IsI {
		keyRangeTree = utils.NewRangeTreeFromString(si.KeyRangeMap[chain_id])
		relatedKeyRangeForest[chain_id] = utils.NewRangeTreeFromString(si.KeyRangeMap[chain_id])
		return keyRangeTree, relatedKeyRangeForest
	} else {
		keyRangeTree = utils.NewRangeTreeFromString(si.KeyRangeMap[chain_id])
		relatedKeyRangeForest[chain_id] = utils.NewRangeTreeFromString(si.KeyRangeMap[chain_id])
		for shard := range si.RelatedShards {
			t := utils.NewRangeTreeFromString(si.KeyRangeMap[shard])
			keyRangeTree.Add(t)
			relatedKeyRangeForest[shard] = t
		}
		return keyRangeTree, relatedKeyRangeForest
	}
}

func createABCI(cfg *Config, si *shardinfo.ShardInfo, block_store *store.PrefixStore) definition.ABCIConn {
	chain_id := cfg.ChainID
	keyRangeTree, relatedKeyRangeForest := createKeyRangeTree(cfg, si)
	switch cfg.ABCIApp {
	case "minibank":
		app := minibank.NewApplication(cfg.StoreDirRoot(), chain_id,
			cfg.IsI, keyRangeTree, relatedKeyRangeForest, block_store)
		return app
	default:
		panic("Undefined ABCI interface")
	}
}

func createMempool(cfg *Config, abci definition.ABCIConn) (definition.MempoolConn, definition.MempoolConn) {
	if cfg.IsI {
		return mempool.NewMempool(false, abci), nil
	} else {
		return mempool.NewMempool(false, abci), mempool.NewMempool(true, abci)
	}
}
func createP2p(cfg *Config, si *shardinfo.ShardInfo) (*p2p.Sender, *p2p.Receiver) {
	sender := p2p.NewSender(fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort))
	receiver := p2p.NewReceiver(cfg.LocalIP, cfg.LocalPort, context.Background())

	for shard := range si.RelatedShards {
		for _, peer := range si.PeerList[shard] {
			sender.AddPeer(peer)
		}
	}
	for _, peer := range si.PeerList[cfg.ChainID] {
		sender.AddPeer(peer)
	}
	return sender, receiver
}
func createConsensus(cfg *Config, si *shardinfo.ShardInfo, s *signer.Signer, mmp, cmmp definition.MempoolConn,
	abci definition.ABCIConn, sender *p2p.Sender, logger blocklogger.BlockWriter) definition.ConsensusConn {
	interval, err := time.ParseDuration(cfg.MinBlockInterval)
	if err != nil {
		panic(err)

	}
	switch cfg.Protocal {
	case ProtocolPyramid:
		return pyramid.NewConsensusState(
			cfg.ChainID, si,
			s, cfg.SignerIndex,
			mmp, cmmp,
			abci, sender,
			cfg.StoreDirRoot(),
			interval,
			cfg.MaxPartSize,
			cfg.MaxBlockTxNum,
			logger,
		)
	default:
		panic("Undefined Consensus interface")
	}
}

func parseStartTime(startTimeStr string) (int, int, error) {
	parts := strings.Split(startTimeStr, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("Time format error, should be HH:MM")
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("Hour parsing error: %v", err)
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("Minute parse error: %v", err)
	}
	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return 0, 0, fmt.Errorf("Hours or minutes are not valid")
	}

	return hours, minutes, nil
}

func time_to_start(startTimeStr string) time.Time {

	flag.Parse()

	if startTimeStr == "" {
		return time.Now()
	}

	hours, minutes, err := parseStartTime(startTimeStr)
	if err != nil {
		panic(fmt.Sprintln("Startup time is formatted incorrectly:", err))

	}

	now := time.Now()

	return time.Date(now.Year(), now.Month(), now.Day(), hours, minutes, 0, 0, time.Local)
}
