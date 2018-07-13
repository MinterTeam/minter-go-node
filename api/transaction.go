package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/gorilla/mux"
	"github.com/tendermint/tendermint/libs/common"
	"net/http"
	"strings"
)

func Transaction(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	hash := strings.TrimLeft(vars["hash"], "Mt")
	decoded, err := hex.DecodeString(hash)

	tx, err := client.Tx(decoded, false)

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

	decodedTx, _ := transaction.DecodeFromBytes(tx.Tx)
	sender, _ := decodedTx.Sender()

	err = json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: TransactionResponse{
			Hash:   common.HexBytes(tx.Tx.Hash()),
			RawTx:  fmt.Sprintf("%x", tx.Tx),
			Height: tx.Height,
			Index:  tx.Index,
			TxResult: ResponseDeliverTx{
				Code:      tx.TxResult.Code,
				Data:      tx.TxResult.Data,
				Log:       tx.TxResult.Log,
				Info:      tx.TxResult.Info,
				GasWanted: tx.TxResult.GasWanted,
				GasUsed:   tx.TxResult.GasUsed,
				Tags:      tx.TxResult.Tags,
			},
			From:     sender.String(),
			Nonce:    decodedTx.Nonce,
			GasPrice: decodedTx.GasPrice,
			Type:     decodedTx.Type,
			Data:     decodedTx.GetDecodedData(),
			Payload:  decodedTx.Payload,
		},
	})

	if err != nil {
		panic(err)
	}
}
