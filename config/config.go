package config

import (
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/spf13/viper"
	tmConfig "github.com/tendermint/tendermint/config"
	"path/filepath"
)

var (
	defaultConfigDir      = "config"
	defaultDataDir        = "data"
	defaultConfigFileName = "config.toml"
	defaultConfigFilePath = filepath.Join(defaultConfigDir, defaultConfigFileName)
)

func init() {
	homeDir := utils.GetMinterHome()
	viper.SetConfigName("config")
	viper.AddConfigPath(filepath.Join(homeDir, "config"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// stderr, so if we redirect output to json file, this doesn't appear
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// ignore not found error, return other errors
		panic(err)
	}
}

func DefaultConfig() *tmConfig.Config {
	cfg := tmConfig.DefaultConfig()

	cfg.P2P.Seeds = "647e32df3b9c54809b5aca2877d9ba60900bc2d9@minter-node-1.testnet.minter.network:26656,d20522aa7ba4af8139749c5e724063c4ba18c58b@minter-node-2.testnet.minter.network:26656,249c62818bf4601605a65b5adc35278236bd5312@minter-node-3.testnet.minter.network:26656,b698b07f13f2210dfc82967bfa2a127d1cdfdc54@minter-node-4.testnet.minter.network:26656"
	cfg.P2P.PersistentPeers = "647e32df3b9c54809b5aca2877d9ba60900bc2d9@minter-node-1.testnet.minter.network:26656"

	cfg.TxIndex = &tmConfig.TxIndexConfig{
		Indexer:      "kv",
		IndexTags:    "",
		IndexAllTags: true,
	}

	cfg.DBPath = "tmdata"

	cfg.Mempool.CacheSize = 100000
	cfg.Mempool.WalPath = "tmdata/mempool.wal"
	cfg.Mempool.Recheck = true
	cfg.Mempool.RecheckEmpty = true

	cfg.Consensus.WalPath = "tmdata/cs.wal/wal"
	cfg.Consensus.TimeoutPropose = 3000
	cfg.Consensus.TimeoutProposeDelta = 500
	cfg.Consensus.TimeoutPrevote = 1000
	cfg.Consensus.TimeoutPrevoteDelta = 500
	cfg.Consensus.TimeoutPrecommit = 1000
	cfg.Consensus.TimeoutPrecommitDelta = 500
	cfg.Consensus.TimeoutCommit = 5000

	cfg.PrivValidator = "config/priv_validator.json"

	cfg.NodeKey = "config/node_key.json"

	cfg.P2P.AddrBook = "config/addrbook.json"
	cfg.P2P.ListenAddress = "tcp://0.0.0.0:26656"
	cfg.P2P.SendRate = 5120000 // 5mb/s
	cfg.P2P.RecvRate = 5120000 // 5mb/s

	return cfg
}

func GetConfig() *tmConfig.Config {
	cfg := DefaultConfig()

	err := viper.Unmarshal(cfg)
	if err != nil {
		panic(err)
	}

	cfg.SetRoot(utils.GetMinterHome())
	EnsureRoot(utils.GetMinterHome())

	return cfg
}
