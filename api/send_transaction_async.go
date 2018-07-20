package api

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"net/http"
	"strings"
)

func SendTransactionAsync(w http.ResponseWriter, r *http.Request) {

	var req SendTransactionRequest
	body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	json.Unmarshal(body, &req)

	result, err := client.BroadcastTxAsync(types.Hex2Bytes(req.Transaction))

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code: code.OK,
		Result: SendTransactionResponse{
			Hash: "Mt" + strings.ToLower(result.Hash.String()),
		},
	})
}
