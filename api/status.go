package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/tendermint/tendermint/rpc/core/types"
	"net/http"
	"time"
)

type StatusResponse struct {
	MinterVersion     string                   `json:"version"`
	LatestBlockHash   string                   `json:"latest_block_hash"`
	LatestAppHash     string                   `json:"latest_app_hash"`
	LatestBlockHeight int64                    `json:"latest_block_height"`
	LatestBlockTime   time.Time                `json:"latest_block_time"`
	TmStatus          *core_types.ResultStatus `json:"tm_status"`
}

func Status(w http.ResponseWriter, r *http.Request) {

	result, err := client.Status()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Response{
			Code: 500,
			Log:  err.Error(),
		})
		return
	}

	tmStatus, err := cdc.MarshalJSON(StatusResponse{
		MinterVersion:     version.Version,
		LatestBlockHash:   fmt.Sprintf("%X", result.SyncInfo.LatestBlockHash),
		LatestAppHash:     fmt.Sprintf("%X", result.SyncInfo.LatestAppHash),
		LatestBlockHeight: result.SyncInfo.LatestBlockHeight,
		LatestBlockTime:   result.SyncInfo.LatestBlockTime,
		TmStatus:          result,
	})

	_ = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: json.RawMessage(tmStatus),
	})
}
