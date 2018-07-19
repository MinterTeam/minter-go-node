package config

import (
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	tmConfig "github.com/tendermint/tendermint/config"
)

var MinterDir = utils.GetMinterHome()
var config = tmConfig.DefaultConfig()

func GetConfig() *tmConfig.Config {

	config.P2P.PersistentPeers = "249c62818bf4601605a65b5adc35278236bd5312@95.216.148.138:26656"

	config.Moniker = "MinterNode"

	config.TxIndex = &tmConfig.TxIndexConfig{
		Indexer:      "kv",
		IndexTags:    "",
		IndexAllTags: true,
	}

	config.DBPath = MinterDir + "/tmdata"

	config.Mempool.CacheSize = 100000
	config.Mempool.WalPath = MinterDir + "/tmdata/mempool.wal"
	config.Mempool.Recheck = true
	config.Mempool.RecheckEmpty = true

	config.Consensus.WalPath = MinterDir + "/tmdata/cs.wal/wal"
	config.Consensus.TimeoutPropose = 3000
	config.Consensus.TimeoutProposeDelta = 500
	config.Consensus.TimeoutPrevote = 1000
	config.Consensus.TimeoutPrevoteDelta = 500
	config.Consensus.TimeoutPrecommit = 1000
	config.Consensus.TimeoutPrecommitDelta = 500
	config.Consensus.TimeoutCommit = 5000

	config.PrivValidator = MinterDir + "/config/priv_validator.json"

	config.NodeKey = MinterDir + "/config/node_key.json"

	config.P2P.AddrBook = MinterDir + "/config/addrbook.json"
	config.P2P.ListenAddress = "tcp://0.0.0.0:26656"
	return config
}
