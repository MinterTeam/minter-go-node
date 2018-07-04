package api

import (
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/tendermint/tendermint/types"
	"net/http"
	"strings"
)

func Transaction(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	hash := strings.TrimLeft(vars["hash"], "Mt")
	decoded, err := hex.DecodeString(hash)

	result := new(types.TxResult)
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
