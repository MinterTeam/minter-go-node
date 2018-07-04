package api

import (
	"encoding/json"
	"github.com/tendermint/tendermint/libs/common"
	"net/http"
	"time"
)

type StatusResponse struct {
	LatestBlockHash   common.HexBytes `json:"latest_block_hash"`
	LatestAppHash     common.HexBytes `json:"latest_app_hash"`
	LatestBlockHeight int64           `json:"latest_block_height"`
	LatestBlockTime   time.Time       `json:"latest_block_time"`
}

func Status(w http.ResponseWriter, r *http.Request) {

	result, err := client.Status()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Code:   500,
			Result: nil,
			Log:    err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: StatusResponse{
			LatestBlockHash:   common.HexBytes(result.SyncInfo.LatestBlockHash),
			LatestAppHash:     common.HexBytes(result.SyncInfo.LatestAppHash),
			LatestBlockHeight: result.SyncInfo.LatestBlockHeight,
			LatestBlockTime:   result.SyncInfo.LatestBlockTime,
		},
	})
}
