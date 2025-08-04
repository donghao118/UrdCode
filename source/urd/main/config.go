package main

import (
	"emulator/urd/abci/minibank"
	"emulator/urd/shardinfo"
	"emulator/utils"
	"emulator/utils/p2p"
	"emulator/utils/signer"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	privateKeyPath = "config/private_key.txt"
	shardInfoPath  = "config/shard_info.json"
	configPath     = "config/config.toml"
	configDir      = "config"
	storeDir       = "database"
	datasetDir     = "dataset"
)

const (
	defaultMinBlockInterval          = "10ms"
	defaultMaxBlockPartSize          = 200 * 1024 // 20 KB
	defaultMaxBlockTxBytes           = 160 * 1024
	defaultMaxBlockCrossShardTxBytes = 640 * 1024
	defaultProtocal                  = "tendermint"
	defaultABCI                      = "minibank"
)

type Config struct {
	NodeName string
	DirRoot  string
	ChainID  string

	// P2P
	LocalIP   string
	LocalPort int

	// consensus
	Protocal                                   string
	MinBlockInterval                           string
	MaxPartSize                                int
	MaxBlockTxBytes, MaxBlockCrossShardTxBytes int

	SignerIndex int
	IsLeader    bool

	// abci
	ABCIApp string
}

func (c *Config) PrivateKeyPath() string { return filepath.Join(c.DirRoot, privateKeyPath) }
func (c *Config) ShardInfoPath() string  { return filepath.Join(c.DirRoot, shardInfoPath) }
func (c *Config) StoreDirRoot() string   { return filepath.Join(c.DirRoot, storeDir) }
func (c *Config) ConfigPath() string     { return filepath.Join(c.DirRoot, configPath) }
func (c *Config) ConfigDir() string      { return filepath.Join(c.DirRoot, configDir) }
func (c *Config) DatasetDir() string     { return filepath.Join(c.DirRoot, datasetDir) }

func GenerateConfigFiles(shard_config_path string, store_dir string) {
	shardConfig := new(ShardConfig)
	if err := shardConfig.ReadJSONFromFile(shard_config_path); err != nil {
		panic(err)
	}

	// Assign IP addresses and port numbers through roulette wheel
	IPPortToUse := map[string]int{}
	IPs := []string{}
	IPGun := []string{}
	IPGunPointer := 0
	maxCounter := -1
	for key, value := range shardConfig.IPInUse {
		IPs = append(IPs, key)
		IPPortToUse[key] = 26601
		if int(value) > maxCounter {
			maxCounter = int(value)
		}
	}
	sort.Strings(IPs)
	for i := 0; i < maxCounter; i++ {
		for _, ip := range IPs {
			if shardConfig.IPInUse[ip] > uint32(i) {
				IPGun = append(IPGun, ip)
			}
		}
	}
	GetIP := func() (string, int) {
		if IPGunPointer == len(IPGun) {
			IPGunPointer = 0
		}
		theip := IPGun[IPGunPointer]
		theport := IPPortToUse[theip]
		IPPortToUse[theip] = theport + 1
		IPGunPointer++
		return theip, theport
	}

	// calculate node nums
	totalNodes := 0
	for _, si := range shardConfig.Shards {
		totalNodes += int(si.PeerNum)
	}

	// generate publicKeys, privKeys and corresponding IP
	publicKeys := make([]string, totalNodes)
	privateKeys := make([]string, totalNodes)
	ipList := make([]string, totalNodes)
	portList := make([]int, totalNodes)
	for i := 0; i < totalNodes; i++ {
		priveKey, pubkey, err := signer.NewBLSKeyPair(signer.BaseCurve)
		if err != nil {
			panic(err)
		}
		publicKeys[i] = pubkey
		privateKeys[i] = priveKey
		ipList[i], portList[i] = GetIP()
	}

	// generate ShardInfo
	PeerList := make(map[string][]*p2p.Peer)
	keyRangeMap := make(map[string]string)
	count := 0
	for _, si := range shardConfig.Shards {
		for i := 0; i < int(si.PeerNum); i++ {
			peer, err := p2p.NewPeer(fmt.Sprintf("%s:%d", ipList[count], portList[count]),
				map[string]bool{si.ChainID: true},
				publicKeys[count], 1)
			if err != nil {
				panic(err)
			}
			PeerList[si.ChainID] = append(PeerList[si.ChainID], peer)
			count++
		}
		keyRangeMap[si.ChainID] = si.KeyRange
	}
	var shard_info *shardinfo.ShardInfo = shardinfo.NewShardInfo(PeerList, 0, keyRangeMap)

	// generate config
	configList := make([]*Config, totalNodes)
	count = 0
	for _, si := range shardConfig.Shards {
		for i := 0; i < int(si.PeerNum); i++ {
			nodeName := fmt.Sprintf("node%d", count+1)
			dirRoot := filepath.Join(store_dir, ipList[count], nodeName)
			configList[count] = &Config{
				DirRoot:  dirRoot,
				NodeName: nodeName,
				ChainID:  si.ChainID,

				LocalIP:   ipList[count],
				LocalPort: portList[count],

				Protocal:                  defaultProtocal,
				MinBlockInterval:          defaultMinBlockInterval,
				MaxPartSize:               defaultMaxBlockPartSize,
				MaxBlockTxBytes:           defaultMaxBlockTxBytes,
				MaxBlockCrossShardTxBytes: defaultMaxBlockCrossShardTxBytes,

				SignerIndex: i,
				IsLeader:    i == 0,

				ABCIApp: defaultABCI,
			}
			count++
		}
	}

	rls := map[string]*utils.RangeList{}
	for s, v := range keyRangeMap {
		rls[s] = utils.NewRangeListFromString(v)
	}
	generator := minibank.NewImportorForGenerator(rls)
	txs := generator.GenerateTxs()

	for i, cfg := range configList {
		if err := os.MkdirAll(cfg.StoreDirRoot(), os.ModePerm); err != nil {
			panic(err)
		}
		if err := os.MkdirAll(cfg.ConfigDir(), os.ModePerm); err != nil {
			panic(err)
		}
		privKey := privateKeys[i]
		if str, err := json.MarshalIndent(shard_info, "", "    "); err != nil {
			panic(err)
		} else if err2 := os.WriteFile(cfg.ShardInfoPath(), []byte(str), 0666); err2 != nil {
			panic(err)
		}
		if err := os.WriteFile(cfg.PrivateKeyPath(), []byte(privKey), 0666); err != nil {
			panic(err)
		}
		if err := cfg.StoreConfig(cfg.ConfigPath()); err != nil {
			panic(err)
		}

		if cfg.IsLeader {
			if err := os.MkdirAll(cfg.DatasetDir(), os.ModePerm); err != nil {
				panic(err)
			}

			if err := writeStringsToFile(txs, filepath.Join(cfg.DatasetDir(), "dataset.txt")); err != nil {
				panic(err)
			}
		}
	}
}

// =========================================================================
func writeStringsToFile(strs []string, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, str := range strs {
		_, err := file.WriteString(str + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}
