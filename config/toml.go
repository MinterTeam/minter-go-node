package config

import (
	"bytes"
	"github.com/tendermint/tendermint/libs/os"
	"path/filepath"
	"text/template"
)

var configTemplate *template.Template

func init() {
	var err error
	if configTemplate, err = template.New("configFileTemplate").Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

/****** these are for production settings ***********/

// EnsureRoot creates the root, config, and data directories if they don't exist,
// and panics if it fails.
func EnsureRoot(rootDir string) {
	if err := os.EnsureDir(rootDir, 0700); err != nil {
		panic(err.Error())
	}
	if err := os.EnsureDir(filepath.Join(rootDir, DefaultConfigDir), 0700); err != nil {
		panic(err.Error())
	}
	if err := os.EnsureDir(filepath.Join(rootDir, DefaultDataDir), 0700); err != nil {
		panic(err.Error())
	}

	configFilePath := filepath.Join(rootDir, defaultConfigFilePath)

	// Write default config file if missing.
	if !os.FileExists(configFilePath) {
		writeDefaultConfigFile(configFilePath)
	}
}

// XXX: this func should probably be called by cmd/tendermint/commands/init.go
// alongside the writing of the genesis.json and priv_validator.json
func writeDefaultConfigFile(configFilePath string) {
	WriteConfigFile(configFilePath, DefaultConfig())
}

// WriteConfigFile renders config using the template and writes it to configFilePath.
func WriteConfigFile(configFilePath string, config *Config) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	os.MustWriteFile(configFilePath, buffer.Bytes(), 0644)
}

// Note: any changes to the comments/variables/mapstructure
// must be reflected in the appropriate struct in config/config.go
const defaultConfigTemplate string = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

##### main base config options #####

# A custom human readable name for this node
moniker = "{{ .BaseConfig.Moniker }}"

# Address to listen for gRPC connections
grpc_listen_addr = "{{ .BaseConfig.GRPCListenAddress }}"

# Address to listen for API V2 connections
api_v2_listen_addr = "{{ .BaseConfig.APIv2ListenAddress }}"

# API v2 Timeout
api_v2_timeout_duration = "{{ .BaseConfig.APIv2TimeoutDuration }}"

# Need add "rpc:info" to log_level
api_v2_logger = "{{ .BaseConfig.APIv2Logger }}"

api_v2_prometheus = "{{ .BaseConfig.APIv2Prometheus }}"

# WebSocket connection duration
ws_connection_duration = "{{ .BaseConfig.WSConnectionDuration }}"

# Sets node to be in validator mode. Disables API, events, history of blocks, indexes, etc. 
validator_mode = {{ .BaseConfig.ValidatorMode }}

# Sets number of last stated to be saved on disk.
keep_last_states = {{ .BaseConfig.KeepLastStates }}

# State cache size 
state_cache_size = {{ .BaseConfig.StateCacheSize }}

# State memory in MB
state_mem_available = {{ .BaseConfig.StateMemAvailable }}

# Limit for simultaneous requests to API
api_simultaneous_requests = {{ .BaseConfig.APISimultaneousRequests }}

# If this node is many blocks behind the tip of the chain, FastSync
# allows them to catchup quickly by downloading blocks in parallel
# and verifying their commits
fast_sync = {{ .BaseConfig.FastSync }}

# State sync snapshot interval
snapshot_interval = {{ .BaseConfig.SnapshotInterval }}

# State sync snapshot to keep
snapshot_keep_recent = {{ .BaseConfig.SnapshotKeepRecent }}

# Database backend: leveldb | memdb
db_backend = "{{ .BaseConfig.DBBackend }}"

# Database directory
db_path = "{{ js .BaseConfig.DBPath }}"

# Output level for logging, including package level options
log_level = "{{ .BaseConfig.LogLevel }}"

# Output format: 'plain' (colored text) or 'json'
log_format = "{{ .BaseConfig.LogFormat }}"

# Path to file for logs, "stdout" by default
log_path = "{{ .BaseConfig.LogPath }}"

##### additional base config options #####

# Path to the JSON file containing the private key to use as a validator in the consensus protocol
priv_validator_key_file = "{{ js .BaseConfig.PrivValidatorKey }}"
priv_validator_state_file = "{{ js .BaseConfig.PrivValidatorState }}"

# Path to the JSON file containing the private key to use for node authentication in the p2p protocol
node_key_file = "{{ js .BaseConfig.NodeKey}}"

# TCP or UNIX socket address for the profiling server to listen on
prof_laddr = "{{ .BaseConfig.ProfListenAddress }}"

##### advanced configuration options #####

[statesync]

enable = {{ .StateSync.Enable }}

# At least 2 available RPC servers.
rpc_servers = [{{range $element := .StateSync.RPCServers}} "{{$element}}", {{end}}]

## Use for update [curl -s http://{{index .StateSync.RPCServers 1}}/block | jq -r '.result.block.header.height + "\n" + .result.block_id.hash']
# A trusted height
trust_height = {{ .StateSync.TrustHeight }}
# The block ID hash of the trusted height
trust_hash = "{{ .StateSync.TrustHash }}"

trust_period = "{{ .StateSync.TrustPeriod }}"

##### rpc server configuration options #####
[rpc]

# TCP or UNIX socket address for the RPC server to listen on
laddr = "{{ .RPC.ListenAddress }}"

# TCP or UNIX socket address for the gRPC server to listen on
# NOTE: This server only supports /broadcast_tx_commit
grpc_laddr = "{{ .RPC.GRPCListenAddress }}"

# Maximum number of simultaneous connections.
# Does not include RPC (HTTP&WebSocket) connections. See max_open_connections
# If you want to accept more significant number than the default, make sure
# you increase your OS limits.
# 0 - unlimited.
grpc_max_open_connections = {{ .RPC.GRPCMaxOpenConnections }}

# Activate unsafe RPC commands like /dial_seeds and /unsafe_flush_mempool
unsafe = {{ .RPC.Unsafe }}

# Maximum number of simultaneous connections (including WebSocket).
# Does not include gRPC connections. See grpc_max_open_connections
# If you want to accept more significant number than the default, make sure
# you increase your OS limits.
# 0 - unlimited.
max_open_connections = {{ .RPC.MaxOpenConnections }}

##### peer to peer configuration options #####
[p2p]

# Address to listen for incoming connections
laddr = "{{ .P2P.ListenAddress }}"

# Address to advertise to peers for them to dial
# If empty, will use the same port as the laddr,
# and will introspect on the listener or use UPnP
# to figure out the address.
external_address = "{{ .P2P.ExternalAddress }}"

# Comma separated list of seed nodes to connect to
seeds = "{{ .P2P.Seeds }}"

# Comma separated list of nodes to keep persistent connections to
persistent_peers = "{{ .P2P.PersistentPeers }}"

# UPNP port forwarding
upnp = {{ .P2P.UPNP }}

# Set true for strict address routability rules
addr_book_strict = {{ .P2P.AddrBookStrict }}

# Time to wait before flushing messages out on the connection, in ms
flush_throttle_timeout = "{{ .P2P.FlushThrottleTimeout }}"

# Maximum number of inbound peers
max_num_inbound_peers = {{ .P2P.MaxNumInboundPeers }}

# Maximum number of outbound peers to connect to, excluding persistent peers
max_num_outbound_peers = {{ .P2P.MaxNumOutboundPeers }}

# Maximum size of a message packet payload, in bytes
max_packet_msg_payload_size = {{ .P2P.MaxPacketMsgPayloadSize }}

# Rate at which packets can be sent, in bytes/second
send_rate = {{ .P2P.SendRate }}

# Rate at which packets can be received, in bytes/second
recv_rate = {{ .P2P.RecvRate }}

# Set true to enable the peer-exchange reactor
pex = {{ .P2P.PexReactor }}

# Seed mode, in which node constantly crawls the network and looks for
# peers. If another node asks it for addresses, it responds and disconnects.
#
# Does not work if the peer-exchange reactor is disabled.
seed_mode = {{ .P2P.SeedMode }}

# Comma separated list of peer IDs to keep private (will not be gossiped to other peers)
private_peer_ids = "{{ .P2P.PrivatePeerIDs }}"

##### mempool configuration options #####
[mempool]

recheck = false
broadcast = {{ .Mempool.Broadcast }}
wal_dir = "{{ js .Mempool.WalPath }}"

# size of the mempool
size = {{ .Mempool.Size }}

# size of the cache (used to filter transactions we saw earlier)
cache_size = {{ .Mempool.CacheSize }}

##### instrumentation configuration options #####
[instrumentation]

# When true, Prometheus metrics are served under /metrics on
# PrometheusListenAddr.
# Check out the documentation for the list of available metrics.
prometheus = {{ .Instrumentation.Prometheus }}

# Address to listen for Prometheus collector(s) connections
prometheus_listen_addr = "{{ .Instrumentation.PrometheusListenAddr }}"

# Maximum number of simultaneous connections.
# If you want to accept more significant number than the default, make sure
# you increase your OS limits.
# 0 - unlimited.
max_open_connections = {{ .Instrumentation.MaxOpenConnections }}

# Instrumentation namespace
namespace = "minter"
`
