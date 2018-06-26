package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/gorilla/mux"
	"github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/common"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

type BlockResponse struct {
	Hash         common.HexBytes            `json:"hash"`
	Height       int64                      `json:"height"`
	Time         time.Time                  `json:"time"`
	NumTxs       int64                      `json:"num_txs"`
	TotalTxs     int64                      `json:"total_txs"`
	Transactions []BlockTransactionResponse `json:"transactions"`
	Precommits   []*types.Vote              `json:"precommits"`
}

type BlockTransactionResponse struct {
	Hash        string           `json:"hash"`
	From        string           `json:"from"`
	Nonce       uint64           `json:"nonce"`
	GasPrice    *big.Int         `json:"gasPrice"`
	Type        byte             `json:"type"`
	Data        transaction.Data `json:"data"`
	Payload     []byte           `json:"payload"`
	ServiceData []byte           `json:"serviceData"`
}

func Block(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	height, _ := strconv.ParseInt(vars["height"], 10, 64)

	result := new(core_types.ResultBlock)
	_, err := client.Call("block", map[string]interface{}{
		"height": height,
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

	txs := make([]BlockTransactionResponse, len(result.Block.Data.Txs))

	for i, rawTx := range result.Block.Data.Txs {
		tx, _ := transaction.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		txs[i] = BlockTransactionResponse{
			Hash:        fmt.Sprintf("Mt%x", types.Tx(rawTx).Hash()),
			From:        sender.String(),
			Nonce:       tx.Nonce,
			GasPrice:    tx.GasPrice,
			Type:        tx.Type,
			Data:        tx.GetDecodedData(),
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
		}
	}

	response := BlockResponse{
		Hash:         result.Block.Hash(),
		Height:       result.Block.Height,
		Time:         result.Block.Time,
		NumTxs:       result.Block.NumTxs,
		TotalTxs:     result.Block.TotalTxs,
		Precommits:   result.Block.LastCommit.Precommits,
		Transactions: txs,
	}

	err = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: response,
	})

	if err != nil {
		panic(err)
	}
}
