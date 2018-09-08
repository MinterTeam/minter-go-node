package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/gorilla/mux"
	"github.com/tendermint/tendermint/libs/common"
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
	Precommits   json.RawMessage            `json:"precommits"`
	BlockReward  string                     `json:"block_reward"`
	Size         int                        `json:"size"`
}

type BlockTransactionResponse struct {
	Hash        string            `json:"hash"`
	RawTx       string            `json:"raw_tx"`
	From        string            `json:"from"`
	Nonce       uint64            `json:"nonce"`
	GasPrice    *big.Int          `json:"gas_price"`
	Type        byte              `json:"type"`
	Data        transaction.Data  `json:"data"`
	Payload     []byte            `json:"payload"`
	ServiceData []byte            `json:"service_data"`
	Gas         int64             `json:"gas"`
	GasCoin     types.CoinSymbol  `json:"gas_coin"`
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
			Code: 0,
			Log:  err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	txs := make([]BlockTransactionResponse, len(block.Block.Data.Txs))

	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		txs[i] = BlockTransactionResponse{
			Hash:        fmt.Sprintf("Mt%x", rawTx.Hash()),
			RawTx:       fmt.Sprintf("%x", []byte(rawTx)),
			From:        sender.String(),
			Nonce:       tx.Nonce,
			GasPrice:    tx.GasPrice,
			Type:        tx.Type,
			Data:        tx.GetDecodedData(),
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
			Gas:         tx.Gas(),
			GasCoin:     tx.GasCoin,
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

	precommits, _ := cdc.MarshalJSON(block.Block.LastCommit.Precommits)

	encodedBlock, _ := cdc.MarshalBinary(block)

	size := len(encodedBlock)

	response := BlockResponse{
		Hash:         block.Block.Hash(),
		Height:       block.Block.Height,
		Time:         block.Block.Time,
		NumTxs:       block.Block.NumTxs,
		TotalTxs:     block.Block.TotalTxs,
		Transactions: txs,
		Precommits:   json.RawMessage(precommits),
		BlockReward:  rewards.GetRewardForBlock(uint64(height)).String(),
		Size:         size,
	}

	err = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: response,
	})

	if err != nil {
		panic(err)
	}
}
