package api

import (
	"encoding/json"
	"github.com/tendermint/tmlibs/common"
	"minter/rpc/lib/client"
	"minter/tmtypes"
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

	client := rpcclient.NewJSONRPCClient(tendermintRpcAddr)
	tmtypes.RegisterAmino(client.Codec())
	result := new(tmtypes.ResultStatus)
	_, err := client.Call("status", map[string]interface{}{}, result)

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: StatusResponse{
			LatestBlockHash:   common.HexBytes(result.SyncInfo.LatestBlockHash),
			LatestAppHash:     common.HexBytes(result.SyncInfo.LatestAppHash),
			LatestBlockHeight: result.SyncInfo.LatestBlockHeight,
			LatestBlockTime:   result.SyncInfo.LatestBlockTime,
		},
	})

	if err != nil {
		panic(err)
	}
}
