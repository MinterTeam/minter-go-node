package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/tendermint/tendermint/rpc/core/types"
	"time"
)

type StatusResponse struct {
	MinterVersion     string                   `json:"version"`
	LatestBlockHash   string                   `json:"latest_block_hash"`
	LatestAppHash     string                   `json:"latest_app_hash"`
	LatestBlockHeight int64                    `json:"latest_block_height"`
	LatestBlockTime   time.Time                `json:"latest_block_time"`
	KeepLastStates    int64                    `json:"keep_last_states"`
	TmStatus          *core_types.ResultStatus `json:"tm_status"`
}

func Status() (*StatusResponse, error) {
	result, err := client.Status()
	if err != nil {
		return nil, err
	}

	return &StatusResponse{
		MinterVersion:     version.Version,
		LatestBlockHash:   fmt.Sprintf("%X", result.SyncInfo.LatestBlockHash),
		LatestAppHash:     fmt.Sprintf("%X", result.SyncInfo.LatestAppHash),
		LatestBlockHeight: result.SyncInfo.LatestBlockHeight,
		LatestBlockTime:   result.SyncInfo.LatestBlockTime,
		KeepLastStates:    minterCfg.BaseConfig.KeepLastStates,
		TmStatus:          result,
	}, nil
}
