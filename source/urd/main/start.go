package main

import (
	"context"
	"emulator/urd/abci/minibank"
	"emulator/urd/consensus"
	"emulator/urd/definition"
	"emulator/urd/mempool"
	"emulator/urd/shardinfo"
	"emulator/utils"
	"emulator/utils/p2p"
	"emulator/utils/signer"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"emulator/logger/blocklogger"

	"github.com/herumi/bls-eth-go-binary/bls"
)

func InitNode(rootDir string, startTimeStr string, waitTime string) {
	defer fmt.Println("test ending")

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
	if err := shardInfo.UnmarshalJson(shardInfoBz); err != nil {
		panic(err)
	}

	abci := createABCI(cfg, shardInfo)
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

	receiver.Start()
	importor := minibank.NewImportor(mempool, cross_shard_mempool, logger, cfg.ChainID, filepath.Join(cfg.DatasetDir(), "dataset.txt"), createKeyRangeTree(cfg, shardInfo),
		cfg.SignerIndex == shardInfo.Shards[cfg.ChainID].LeaderIndex)
	if err := importor.Start(); err != nil {
		panic(err)
	}

	startTime := time_to_start(startTimeStr)
	fmt.Println(time.Until(startTime))
	time.Sleep(time.Until(startTime))

	// The SENDER is started after a certain delay to ensure that all RECEIVERS have been started completel
	if err := sender.Start(); err != nil {
		fmt.Println(err)
	}

	t, err := strconv.ParseInt(waitTime, 10, 32)
	if err == nil && t >= 0 {
		time.Sleep(time.Duration(t) * time.Second)
	} else {
		time.Sleep(10 * time.Second)
	}

	fmt.Println("Starting consensus...", shardInfo.ShardIDList)
	consensus.Start()

	select {}
}

func createKeyRangeTree(cfg *Config, si *shardinfo.ShardInfo) map[string]*utils.RangeList {
	out := make(map[string]*utils.RangeList)
	for id, shard := range si.Shards {
		t := utils.NewRangeListFromString(shard.KeyRange)
		if t == nil {
			panic("key range tree unmarshal error")
		}
		out[id] = t
	}
	return out
}

func createABCI(cfg *Config, si *shardinfo.ShardInfo) definition.ABCIConn {
	chain_id := cfg.ChainID
	rangeLists := createKeyRangeTree(cfg, si)
	switch cfg.ABCIApp {
	case "minibank":
		app := minibank.NewApplication(cfg.StoreDirRoot(), chain_id, rangeLists, si)
		return app
	default:
		panic("Undefined ABCI")
	}
}

func createMempool(cfg *Config, abci definition.ABCIConn) (definition.MempoolConn, definition.MempoolConn) {
	return mempool.NewMempool(false, abci), mempool.NewMempool(true, abci)
}
func createP2p(cfg *Config, si *shardinfo.ShardInfo) (*p2p.Sender, *p2p.Receiver) {
	sender := p2p.NewSender(fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort))
	receiver := p2p.NewReceiver(cfg.LocalIP, cfg.LocalPort, context.Background())

	for _, shard := range si.Shards {
		for _, peer := range shard.PeerList {
			sender.AddPeer(peer)
		}
	}
	return sender, receiver
}
func createConsensus(cfg *Config, si *shardinfo.ShardInfo, s *signer.Signer, mmp, cmmp definition.MempoolConn,
	abci definition.ABCIConn, sender *p2p.Sender, logger blocklogger.BlockWriter) definition.ConsensusConn {
	state, err := consensus.NewState(
		0, 0,
		s, cfg.SignerIndex,
		si, cfg.ChainID,
		mmp, cmmp, abci, sender, cfg.StoreDirRoot(), logger,
		cfg.MaxBlockTxBytes, cfg.MaxBlockCrossShardTxBytes,
	)
	if err != nil {
		panic(err)
	}
	return state

}

func parseStartTime(startTimeStr string) (int, int, error) {
	// Split the time string by ":"
	parts := strings.Split(startTimeStr, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("Incorrect time format, should be HH:MM")
	}

	// Parse hours and minutes
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("Error parsing hours: %v", err)
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("Error parsing minutes: %v", err)
	}

	// Check if hours and minutes are valid
	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return 0, 0, fmt.Errorf("Invalid hours or minutes")
	}

	return hours, minutes, nil
}

func time_to_start(startTimeStr string) time.Time {

	// Parse command - line arguments
	flag.Parse()

	// Check if the --start - time flag is provided
	if startTimeStr == "" {
		//fmt.Println("Please use the --start - time flag to specify the program startup time")
		return time.Now()
	}

	// Parse the startup time string
	hours, minutes, err := parseStartTime(startTimeStr)
	if err != nil {
		panic(fmt.Sprintln("Incorrect startup time format:", err))

	}

	// Get the current date
	now := time.Now()

	// Set the startup time
	return time.Date(now.Year(), now.Month(), now.Day(), hours, minutes, 0, 0, time.Local)
}
