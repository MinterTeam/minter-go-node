package api

import (
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/mux"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/common"
	"net/http"
	"strings"
)

type ResTx struct {
	Hash     common.HexBytes        `json:"hash"`
	Height   int64                  `json:"height"`
	Index    uint32                 `json:"index"`
	TxResult abci.ResponseDeliverTx `json:"tx_result"`
	Tx       types.Tx               `json:"tx"`
	Proof    types.TxProof          `json:"proof,omitempty"`
}

func Transaction(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	hash := strings.TrimRight(vars["hash"], "Mt")
	decoded, err := hex.DecodeString(hash)

	result := new(ResTx)
	_, err = client.Call("tx", map[string]interface{}{
		"hash": decoded,
	}, result)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Code:   0,
			Result: err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result,
	})

	if err != nil {
		panic(err)
	}
}
