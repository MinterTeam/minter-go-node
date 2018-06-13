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

func SendTransactionAsync(w http.ResponseWriter, r *http.Request) {

	var req SendTransactionRequest
	body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	json.Unmarshal(body, &req)

	result := new(core_types.ResultBroadcastTx)
	_, err := client.Call("broadcast_tx_async", map[string]interface{}{
		"tx": types.Hex2Bytes(strings.TrimLeft(req.Transaction, "Mx")),
	}, result)

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   code.OK,
		Result: "Mx" + strings.ToLower(result.Hash.String()),
	})
}
