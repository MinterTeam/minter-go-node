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

	tags := make(map[string]string)

	for _, tag := range tx.TxResult.Tags {
		tags[string(tag.Key)] = string(tag.Value)
	}

	err = json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: TransactionResponse{
			Hash:     common.HexBytes(tx.Tx.Hash()),
			RawTx:    fmt.Sprintf("%x", []byte(tx.Tx)),
			Height:   tx.Height,
			Index:    tx.Index,
			From:     sender.String(),
			Nonce:    decodedTx.Nonce,
			GasPrice: decodedTx.GasPrice,
			GasCoin:  decodedTx.GasCoin,
			GasUsed:  tx.TxResult.GasUsed,
			Type:     decodedTx.Type,
			Data:     decodedTx.GetDecodedData(),
			Payload:  decodedTx.Payload,
			Tags:     tags,
			Code:     tx.TxResult.Code,
			Log:      tx.TxResult.Log,
		},
	})

	if err != nil {
		panic(err)
	}
}
