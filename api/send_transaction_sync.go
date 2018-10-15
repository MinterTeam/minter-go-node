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

func SendTransactionSync(w http.ResponseWriter, r *http.Request) {

	var req SendTransactionRequest
	body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	err := json.Unmarshal(body, &req)

	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{
			Code: 1,
			Log:  "Request decode error",
		})
		return
	}

	result, err := client.BroadcastTxSync(types.Hex2Bytes(req.Transaction))

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if result.Code != code.OK {
		w.WriteHeader(http.StatusInternalServerError)

		_ = json.NewEncoder(w).Encode(Response{
			Code: result.Code,
			Log:  "Check tx error: " + result.Log,
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(Response{
		Code: code.OK,
		Result: SendTransactionResponse{
			Hash: "Mt" + strings.ToLower(result.Hash.String()),
		},
	})
}
