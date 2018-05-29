package api

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"minter/rpc/lib/client"
	"encoding/hex"
	types "minter/tmtypes"
	"github.com/tendermint/tmlibs/common"
	abci "github.com/tendermint/abci/types"
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
	hash := vars["hash"]
	decoded, err := hex.DecodeString(hash)

	client := rpcclient.NewJSONRPCClient(tendermintSocket)
	types.RegisterAmino(client.Codec())
	result := new(ResTx)
	_, err = client.Call("tx", map[string]interface{}{
		"hash": decoded,
	}, result)

	if err != nil {
		panic(err)
	}

	// TODO: check error

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result,
	})

	if err != nil {
		panic(err)
	}
}
