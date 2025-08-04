package main

import (
	"encoding/json"
	"os"
)

type ShardConfig struct {
	IPInUse map[string]uint32 `json:"ip_in_use"`
	Shards  []ShardInfo       `json:"shards"`
}

type ShardInfo struct {
	ChainID       string   `json:"chain_id"`
	PeerNum       uint32   `json:"peer_num"`
	RelatedShards []string `json:"related_shards"`
	IsI           bool     `json:"is_ishard"`
	KeyRange      string   `json:"key_range"`
}

func (cfg *ShardConfig) SerializeToJSON() ([]byte, error) {
	return json.MarshalIndent(cfg, "", "    ")
}

func (cfg *ShardConfig) DeserializeFromJSON(data []byte) error {
	err := json.Unmarshal(data, cfg)
	return err
}

func (cfg *ShardConfig) WriteJSONToFile(filePath string) error {
	data, err := cfg.SerializeToJSON()
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, data, 0666)
	return err
}

func (cfg *ShardConfig) ReadJSONFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	err = cfg.DeserializeFromJSON(data)
	return err
}

func ExampleShardConfig() *ShardConfig {
	return &ShardConfig{
		IPInUse: map[string]uint32{
			"192.168.200.51": 4,
			"192.168.200.52": 5,
			"192.168.200.53": 6,
			"192.168.200.49": 5,
		},
		Shards: []ShardInfo{
			ShardInfo{
				ChainID:       "i1",
				PeerNum:       4,
				RelatedShards: []string{"b1"},
				IsI:           true,
				KeyRange:      "[10,11]",
			},
			ShardInfo{
				ChainID:       "i2",
				PeerNum:       4,
				RelatedShards: []string{"b1", "b2"},
				IsI:           true,
				KeyRange:      "[11,12]",
			},
			ShardInfo{
				ChainID:       "i3",
				PeerNum:       4,
				RelatedShards: []string{"b2"},
				IsI:           true,
				KeyRange:      "[12,13]",
			},
			ShardInfo{
				ChainID:       "b1",
				PeerNum:       4,
				RelatedShards: []string{"i1", "i2"},
				IsI:           false,
				KeyRange:      "[20,21]",
			},
			ShardInfo{
				ChainID:       "b2",
				PeerNum:       4,
				RelatedShards: []string{"i2", "i3"},
				IsI:           false,
				KeyRange:      "[31,32]",
			},
		},
	}
}
