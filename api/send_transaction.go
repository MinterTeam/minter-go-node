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

type SendTransactionRequest struct {
	Transaction string `json:"transaction"`
}

type SendTransactionResponse struct {
	Hash string `json:"hash"`
}

func SendTransaction(w http.ResponseWriter, r *http.Request) {

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

	result, err := client.BroadcastTxCommit(types.Hex2Bytes(req.Transaction))

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if result.CheckTx.Code != code.OK {
		w.WriteHeader(http.StatusInternalServerError)

		_ = json.NewEncoder(w).Encode(Response{
			Code: result.CheckTx.Code,
			Log:  "Check tx error: " + result.CheckTx.Log,
		})
		return
	}

	if result.DeliverTx.Code != code.OK {
		w.WriteHeader(http.StatusInternalServerError)

		_ = json.NewEncoder(w).Encode(Response{
			Code: result.CheckTx.Code,
			Log:  "Deliver tx error: " + result.DeliverTx.Log,
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
