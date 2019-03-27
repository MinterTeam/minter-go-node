package config

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/spf13/viper"
	tmConfig "github.com/tendermint/tendermint/config"
	"os"
	"path/filepath"
	"time"
)

var (
	defaultConfigDir = "config"
	defaultDataDir   = "data"

	defaultConfigFileName  = "config.toml"
	defaultGenesisJSONName = "genesis.json"

	defaultPrivValName      = "priv_validator.json"
	defaultPrivValStateName = "priv_validator_state.json"
	defaultNodeKeyName      = "node_key.json"

	defaultConfigFilePath   = filepath.Join(defaultConfigDir, defaultConfigFileName)
	defaultGenesisJSONPath  = filepath.Join(defaultConfigDir, defaultGenesisJSONName)
	defaultPrivValKeyPath   = filepath.Join(defaultConfigDir, defaultPrivValName)
	defaultPrivValStatePath = filepath.Join(defaultConfigDir, defaultPrivValStateName)
	defaultNodeKeyPath      = filepath.Join(defaultConfigDir, defaultNodeKeyName)
)

const (
	// LogFormatPlain is a format for colored text
	LogFormatPlain = "plain"
	// LogFormatJSON is a format for json output
	LogFormatJSON = "json"
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

func DefaultConfig() *Config {
	cfg := defaultConfig()

	cfg.P2P.Seeds = "647e32df3b9c54809b5aca2877d9ba60900bc2d9@minter-node-1.testnet.minter.network:26656,d20522aa7ba4af8139749c5e724063c4ba18c58b@minter-node-2.testnet.minter.network,249c62818bf4601605a65b5adc35278236bd5312@minter-node-3.testnet.minter.network:26656,b698b07f13f2210dfc82967bfa2a127d1cdfdc54@minter-node-4.testnet.minter.network:26656"

	cfg.TxIndex = &tmConfig.TxIndexConfig{
		Indexer:      "kv",
		IndexTags:    "",
		IndexAllTags: true,
	}

	cfg.DBPath = "tmdata"

	cfg.Mempool.CacheSize = 100000
	cfg.Mempool.Recheck = false
	cfg.Mempool.Size = 10000

	cfg.Consensus.WalPath = "tmdata/cs.wal/wal"
	cfg.Consensus.TimeoutPropose = 2 * time.Second
	cfg.Consensus.TimeoutProposeDelta = 500 * time.Millisecond
	cfg.Consensus.TimeoutPrevote = 1 * time.Second
	cfg.Consensus.TimeoutPrevoteDelta = 500 * time.Millisecond
	cfg.Consensus.TimeoutPrecommit = 1 * time.Second
	cfg.Consensus.TimeoutPrecommitDelta = 500 * time.Millisecond
	cfg.Consensus.TimeoutCommit = 4500 * time.Millisecond

	cfg.P2P.RecvRate = 15360000 // 15 mB/s
	cfg.P2P.SendRate = 15360000 // 15 mB/s
	cfg.P2P.FlushThrottleTimeout = 10 * time.Millisecond

	cfg.PrivValidatorKey = "config/priv_validator.json"
	cfg.PrivValidatorState = "config/priv_validator_state.json"
	cfg.NodeKey = "config/node_key.json"
	cfg.P2P.AddrBook = "config/addrbook.json"

	return cfg
}

func GetConfig() *Config {
	cfg := DefaultConfig()

	err := viper.Unmarshal(cfg)
	if err != nil {
		panic(err)
	}

	if cfg.ValidatorMode {
		cfg.TxIndex.IndexAllTags = false
		cfg.TxIndex.IndexTags = ""

		cfg.RPC.ListenAddress = ""
		cfg.RPC.GRPCListenAddress = ""
	}

	cfg.Mempool.Recheck = false

	cfg.SetRoot(utils.GetMinterHome())
	EnsureRoot(utils.GetMinterHome())

	return cfg
}

// Config defines the top level configuration for a Tendermint node
type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`

	// Options for services
	RPC             *tmConfig.RPCConfig             `mapstructure:"rpc"`
	P2P             *tmConfig.P2PConfig             `mapstructure:"p2p"`
	Mempool         *tmConfig.MempoolConfig         `mapstructure:"mempool"`
	Consensus       *tmConfig.ConsensusConfig       `mapstructure:"consensus"`
	TxIndex         *tmConfig.TxIndexConfig         `mapstructure:"tx_index"`
	Instrumentation *tmConfig.InstrumentationConfig `mapstructure:"instrumentation"`
}

// DefaultConfig returns a default configuration for a Tendermint node
func defaultConfig() *Config {
	return &Config{
		BaseConfig:      DefaultBaseConfig(),
		RPC:             tmConfig.DefaultRPCConfig(),
		P2P:             tmConfig.DefaultP2PConfig(),
		Mempool:         tmConfig.DefaultMempoolConfig(),
		Consensus:       tmConfig.DefaultConsensusConfig(),
		TxIndex:         tmConfig.DefaultTxIndexConfig(),
		Instrumentation: tmConfig.DefaultInstrumentationConfig(),
	}
}

// SetRoot sets the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	cfg.RPC.RootDir = root
	cfg.P2P.RootDir = root
	cfg.Mempool.RootDir = root
	cfg.Consensus.RootDir = root
	return cfg
}

func GetTmConfig() *tmConfig.Config {
	cfg := GetConfig()

	return &tmConfig.Config{
		BaseConfig: tmConfig.BaseConfig{
			RootDir:                 cfg.RootDir,
			Genesis:                 cfg.Genesis,
			PrivValidatorKey:        cfg.PrivValidatorKey,
			PrivValidatorState:      cfg.PrivValidatorState,
			NodeKey:                 cfg.NodeKey,
			Moniker:                 cfg.Moniker,
			PrivValidatorListenAddr: cfg.PrivValidatorListenAddr,
			ProxyApp:                cfg.ProxyApp,
			ABCI:                    cfg.ABCI,
			LogLevel:                cfg.LogLevel,
			LogFormat:               cfg.LogFormat,
			ProfListenAddress:       cfg.ProfListenAddress,
			FastSync:                cfg.FastSync,
			FilterPeers:             cfg.FilterPeers,
			DBBackend:               cfg.DBBackend,
			DBPath:                  cfg.DBPath,
		},
		RPC:             cfg.RPC,
		P2P:             cfg.P2P,
		Mempool:         cfg.Mempool,
		Consensus:       cfg.Consensus,
		TxIndex:         cfg.TxIndex,
		Instrumentation: cfg.Instrumentation,
	}
}

//-----------------------------------------------------------------------------
// BaseConfig

// BaseConfig defines the base configuration for a Tendermint node
type BaseConfig struct {
	// chainID is unexposed and immutable but here for convenience
	chainID string

	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	// Path to the JSON file containing the initial validator set and other meta data
	Genesis string `mapstructure:"genesis_file"`

	// Path to the JSON file containing the private key to use as a validator in the consensus protocol
	PrivValidatorKey string `mapstructure:"priv_validator_key_file"`

	// Path to the JSON file containing the last sign state of a validator
	PrivValidatorState string `mapstructure:"priv_validator_state_file"`

	// TCP or UNIX socket address for Tendermint to listen on for
	// connections from an external PrivValidator process
	PrivValidatorListenAddr string `mapstructure:"priv_validator_laddr"`

	// A JSON file containing the private key to use for p2p authenticated encryption
	NodeKey string `mapstructure:"node_key_file"`

	// A custom human readable name for this node
	Moniker string `mapstructure:"moniker"`

	// TCP or UNIX socket address of the ABCI application,
	// or the name of an ABCI application compiled in with the Tendermint binary
	ProxyApp string `mapstructure:"proxy_app"`

	// Mechanism to connect to the ABCI application: socket | grpc
	ABCI string `mapstructure:"abci"`

	// Output level for logging
	LogLevel string `mapstructure:"log_level"`

	// Output format: 'plain' (colored text) or 'json'
	LogFormat string `mapstructure:"log_format"`

	// TCP or UNIX socket address for the profiling server to listen on
	ProfListenAddress string `mapstructure:"prof_laddr"`

	// If this node is many blocks behind the tip of the chain, FastSync
	// allows them to catchup quickly by downloading blocks in parallel
	// and verifying their commits
	FastSync bool `mapstructure:"fast_sync"`

	// If true, query the ABCI app on connecting to a new peer
	// so the app can decide if we should keep the connection or not
	FilterPeers bool `mapstructure:"filter_peers"` // false

	// Database backend: leveldb | memdb
	DBBackend string `mapstructure:"db_backend"`

	// Database directory
	DBPath string `mapstructure:"db_dir"`

	// Address to listen for GUI connections
	GUIListenAddress string `mapstructure:"gui_listen_addr"`

	// Address to listen for API connections
	APIListenAddress string `mapstructure:"api_listen_addr"`

	ValidatorMode bool `mapstructure:"validator_mode"`

	KeepStateHistory bool `mapstructure:"keep_state_history"`

	APISimultaneousRequests int `mapstructure:"api_simultaneous_requests"`

	LogPath string `mapstructure:"log_path"`
}

// DefaultBaseConfig returns a default base configuration for a Tendermint node
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Genesis:                 defaultGenesisJSONPath,
		PrivValidatorKey:        defaultPrivValKeyPath,
		PrivValidatorState:      defaultPrivValStatePath,
		NodeKey:                 defaultNodeKeyPath,
		Moniker:                 defaultMoniker,
		LogLevel:                DefaultPackageLogLevels(),
		ProfListenAddress:       "",
		FastSync:                true,
		FilterPeers:             false,
		DBBackend:               "leveldb",
		DBPath:                  "data",
		GUIListenAddress:        ":3000",
		APIListenAddress:        "tcp://0.0.0.0:8841",
		ValidatorMode:           false,
		KeepStateHistory:        false,
		APISimultaneousRequests: 100,
		LogPath:                 "stdout",
		LogFormat:               LogFormatPlain,
	}
}

func (cfg BaseConfig) ChainID() string {
	return cfg.chainID
}

// GenesisFile returns the full path to the genesis.json file
func (cfg BaseConfig) GenesisFile() string {
	return rootify(cfg.Genesis, cfg.RootDir)
}

// PrivValidatorFile returns the full path to the priv_validator.json file
func (cfg BaseConfig) PrivValidatorStateFile() string {
	return rootify(cfg.PrivValidatorState, cfg.RootDir)
}

// NodeKeyFile returns the full path to the node_key.json file
func (cfg BaseConfig) NodeKeyFile() string {
	return rootify(cfg.NodeKey, cfg.RootDir)
}

// DBDir returns the full path to the database directory
func (cfg BaseConfig) DBDir() string {
	return rootify(cfg.DBPath, cfg.RootDir)
}

// DefaultLogLevel returns a default log level of "error"
func DefaultLogLevel() string {
	return "error"
}

// DefaultPackageLogLevels returns a default log level setting so all packages
// log at "error", while the `state` and `main` packages log at "info"
func DefaultPackageLogLevels() string {
	return fmt.Sprintf("consensus:info,main:info,blockchain:info,state:info,*:%s", DefaultLogLevel())
}

//-----------------------------------------------------------------------------
// Utils

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

var defaultMoniker = getDefaultMoniker()

// getDefaultMoniker returns a default moniker, which is the host name. If runtime
// fails to get the host name, "anonymous" will be returned.
func getDefaultMoniker() string {
	moniker, err := os.Hostname()
	if err != nil {
		moniker = "anonymous"
	}
	return moniker
}
