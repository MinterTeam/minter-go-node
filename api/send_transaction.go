package api

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/rpc/core/types"
	"net/http"
	"strings"
)

type SendTransactionRequest struct {
	Transaction string `json:"transaction"`
}

func SendTransaction(w http.ResponseWriter, r *http.Request) {

	var req SendTransactionRequest
	body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	json.Unmarshal(body, &req)

	result := new(core_types.ResultBroadcastTxCommit)
	_, err := client.Call("broadcast_tx_commit", map[string]interface{}{
		"tx": types.Hex2Bytes(req.Transaction),
	}, result)

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if result.CheckTx.Code != code.OK {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Response{
			Code: result.CheckTx.Code,
			Log:  "Check tx error: " + result.CheckTx.Log,
		})
		return
	}

	if result.DeliverTx.Code != code.OK {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Response{
			Code: result.CheckTx.Code,
			Log:  "Deliver tx error: " + result.DeliverTx.Log,
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   code.OK,
		Result: "Mt" + strings.ToLower(result.Hash.String()),
	})
}
