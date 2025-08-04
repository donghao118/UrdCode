package main

import (
	"emulator/rapid/shardinfo"
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
)

const (
	ProtocolRapid = "rapid"
)

const (
	defaultMinBlockInterval = "10ms"
	defaultMaxBlockPartSize = 1024 * 200 // 200 KB
	defaultMaxBlockTxNum    = 4096
	defaultProtocal         = ProtocolRapid
	defaultABCI             = "minibank"
)

type Config struct {
	NodeName string
	DirRoot  string
	ChainID  string
	IsI      bool

	BShardNum int
	IShardNum int

	// P2P
	LocalIP   string
	LocalPort int

	// consensus
	Protocal         string
	MinBlockInterval string
	MaxPartSize      int
	MaxBlockTxNum    int

	SignerIndex int

	// abci
	ABCIApp string
}

func (c *Config) PrivateKeyPath() string { return filepath.Join(c.DirRoot, privateKeyPath) }
func (c *Config) ShardInfoPath() string  { return filepath.Join(c.DirRoot, shardInfoPath) }
func (c *Config) StoreDirRoot() string   { return filepath.Join(c.DirRoot, storeDir) }
func (c *Config) ConfigPath() string     { return filepath.Join(c.DirRoot, configPath) }
func (c *Config) ConfigDir() string      { return filepath.Join(c.DirRoot, configDir) }

func GenerateConfigFiles(shard_config_path string, store_dir string) {
	shardConfig := new(ShardConfig)
	if err := shardConfig.ReadJSONFromFile(shard_config_path); err != nil {
		panic(err)
	}
	bnum, inum := 0, 0

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

	totalNodes := 0
	for _, si := range shardConfig.Shards {
		totalNodes += int(si.PeerNum)
		if si.IsI {
			inum++
		} else {
			bnum++
		}
	}

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

	PeerRelatedMap := make(map[string][]string)
	PeerList := make(map[string][]*p2p.Peer)
	keyRangeMap := make(map[string]string)
	count := 0
	for _, si := range shardConfig.Shards {
		PeerRelatedMap[si.ChainID] = si.RelatedShards
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
	var shardInfoList = make([]*shardinfo.ShardInfo, 0, totalNodes)
	for _, si := range shardConfig.Shards {
		var shardInfo = new(shardinfo.ShardInfo)
		if si.IsI {
			shardInfo.SetIShard()
		} else {
			shardInfo.SetBShard()
		}
		shardInfo.PeerRelatedMap = PeerRelatedMap
		shardInfo.PeerList = PeerList
		shardInfo.KeyRangeMap = keyRangeMap
		shardInfo.RelatedShards = make(map[string]bool)
		for _, shard := range si.RelatedShards {
			shardInfo.RelatedShards[shard] = true
		}
		for i := 0; i < int(si.PeerNum); i++ {
			shardInfoList = append(shardInfoList, shardInfo)
		}
	}

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
				IsI:      si.IsI,

				LocalIP:   ipList[count],
				LocalPort: portList[count],

				Protocal:         defaultProtocal,
				MinBlockInterval: defaultMinBlockInterval,
				MaxPartSize:      defaultMaxBlockPartSize,
				MaxBlockTxNum:    defaultMaxBlockTxNum,

				SignerIndex: i,

				ABCIApp: defaultABCI,

				IShardNum: inum,
				BShardNum: bnum,
			}
			count++
		}
	}

	for i, cfg := range configList {
		if err := os.MkdirAll(cfg.StoreDirRoot(), os.ModePerm); err != nil {
			panic(err)
		}
		if err := os.MkdirAll(cfg.ConfigDir(), os.ModePerm); err != nil {
			panic(err)
		}
		privKey := privateKeys[i]
		shardInfo := shardInfoList[i]
		if str, err := json.MarshalIndent(shardInfo, "", "    "); err != nil {
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
	}
}

// =========================================================================
