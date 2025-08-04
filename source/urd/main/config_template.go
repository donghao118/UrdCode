package main

import (
	"html/template"
	"os"

	"github.com/spf13/viper"
)

const ConfigTOML = `
# ===================================================
#              Config of Ours Model
# ===================================================

node_name = "{{.NodeName}}"
dir_root  = "{{.DirRoot}}"
chain_id  = "{{.ChainID}}"

# ===================================================
#              P2P Module
# ===================================================
ip   = "{{.LocalIP}}"  
port = {{.LocalPort}}

# ===================================================
#              Consensus Module
# ===================================================
consensus_protocol       = "{{.Protocal}}"
min_block_interval       = "{{.MinBlockInterval}}" 
max_part_size            = {{.MaxPartSize}}      
max_block_tx_bytes       = {{.MaxBlockTxBytes}}
max_cross_shard_tx_bytes = {{.MaxBlockCrossShardTxBytes}}

signer_index       = {{.SignerIndex}}

# ===================================================
#              ABCI Module
# ===================================================
abci_app = "{{.ABCIApp}}"
`

func (cfg *Config) StoreConfig(path string) error {
	t, err := template.New("config").Parse(ConfigTOML)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, cfg)
}

func GetConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	return &Config{
		NodeName: viper.GetString("node_name"),
		DirRoot:  viper.GetString("dir_root"),
		ChainID:  viper.GetString("chain_id"),

		LocalIP:   viper.GetString("ip"),
		LocalPort: viper.GetInt("port"),

		Protocal:                  viper.GetString("consensus_protocol"),
		MinBlockInterval:          viper.GetString("min_block_interval"),
		MaxPartSize:               viper.GetInt("max_part_size"),
		MaxBlockTxBytes:           viper.GetInt("max_block_tx_bytes"),
		MaxBlockCrossShardTxBytes: viper.GetInt("max_cross_shard_tx_bytes"),

		SignerIndex: viper.GetInt("signer_index"),

		ABCIApp: viper.GetString("abci_app"),
	}, nil
}
