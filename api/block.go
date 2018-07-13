package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/gorilla/mux"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/types"
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
	Hash        string            `json:"hash"`
	RawTx       string            `json:"raw_tx"`
	From        string            `json:"from"`
	Nonce       uint64            `json:"nonce"`
	GasPrice    *big.Int          `json:"gasPrice"`
	Type        byte              `json:"type"`
	Data        transaction.Data  `json:"data"`
	Payload     []byte            `json:"payload"`
	ServiceData []byte            `json:"serviceData"`
	Gas         int64             `json:"gas"`
	TxResult    ResponseDeliverTx `json:"tx_result"`
}

func Block(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	height, _ := strconv.ParseInt(vars["height"], 10, 64)

	block, err := client.Block(&height)
	blockResults, err := client.BlockResults(&height)

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

	txs := make([]BlockTransactionResponse, len(block.Block.Data.Txs))

	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		txs[i] = BlockTransactionResponse{
			Hash:        fmt.Sprintf("Mt%x", types.Tx(rawTx).Hash()),
			RawTx:       fmt.Sprintf("%x", rawTx),
			From:        sender.String(),
			Nonce:       tx.Nonce,
			GasPrice:    tx.GasPrice,
			Type:        tx.Type,
			Data:        tx.GetDecodedData(),
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
			Gas:         tx.Gas(),
			TxResult: ResponseDeliverTx{
				Code:      blockResults.Results.DeliverTx[i].Code,
				Data:      blockResults.Results.DeliverTx[i].Data,
				Log:       blockResults.Results.DeliverTx[i].Log,
				Info:      blockResults.Results.DeliverTx[i].Info,
				GasWanted: blockResults.Results.DeliverTx[i].GasWanted,
				GasUsed:   blockResults.Results.DeliverTx[i].GasUsed,
				Tags:      blockResults.Results.DeliverTx[i].Tags,
			},
		}
	}

	response := BlockResponse{
		Hash:         block.Block.Hash(),
		Height:       block.Block.Height,
		Time:         block.Block.Time,
		NumTxs:       block.Block.NumTxs,
		TotalTxs:     block.Block.TotalTxs,
		Precommits:   block.Block.LastCommit.Precommits,
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
