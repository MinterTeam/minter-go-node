package tmtypes

import (
	"github.com/tendermint/go-amino"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tmlibs/common"
	"time"
)

func RegisterAmino(cdc *amino.Codec) {
	RegisterEventDatas(cdc)
	RegisterEvidences(cdc)
	crypto.RegisterAmino(cdc)
}

// Info about the node's syncing state
type SyncInfo struct {
	LatestBlockHash   common.HexBytes `json:"latest_block_hash"`
	LatestAppHash     common.HexBytes `json:"latest_app_hash"`
	LatestBlockHeight int64           `json:"latest_block_height"`
	LatestBlockTime   time.Time       `json:"latest_block_time"`
	Syncing           bool            `json:"syncing"`
}

// Info about the node's validator
type ValidatorInfo struct {
	Address     common.HexBytes `json:"address"`
	PubKey      crypto.PubKey   `json:"pub_key"`
	VotingPower int64           `json:"voting_power"`
}

// Node Status
type ResultStatus struct {
	NodeInfo      p2p.NodeInfo  `json:"node_info"`
	SyncInfo      SyncInfo      `json:"sync_info"`
	ValidatorInfo ValidatorInfo `json:"validator_info"`
}
