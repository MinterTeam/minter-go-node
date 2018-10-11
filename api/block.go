package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/eventsdb"
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
	Events       json.RawMessage            `json:"events,omitempty"`
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
	GasUsed     int64             `json:"gas_used"`
	Tags        map[string]string `json:"tags"`
	Code        uint32            `json:"code,omitempty"`
	Log         string            `json:"log,omitempty"`
}

func Block(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	height, _ := strconv.ParseInt(vars["height"], 10, 64)

	block, err := client.Block(&height)
	blockResults, err := client.BlockResults(&height)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		err = json.NewEncoder(w).Encode(Response{
			Code: 0,
			Log:  err.Error(),
		})

		if err != nil {
			panic(err)
		}
		return
	}

	txs := make([]BlockTransactionResponse, len(block.Block.Data.Txs))

	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		tags := make(map[string]string)

		for _, tag := range blockResults.Results.DeliverTx[i].Tags {
			switch string(tag.Key) {
			case "tx.type":
				tags[string(tag.Key)] = fmt.Sprintf("%X", tag.Value)
			default:
				tags[string(tag.Key)] = string(tag.Value)
			}
		}

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
			GasUsed:     blockResults.Results.DeliverTx[i].GasUsed,
			Tags:        tags,
			Code:        blockResults.Results.DeliverTx[i].Code,
			Log:         blockResults.Results.DeliverTx[i].Log,
		}
	}

	precommits, _ := cdc.MarshalJSON(block.Block.LastCommit.Precommits)

	encodedBlock, _ := cdc.MarshalBinary(block)

	size := len(encodedBlock)

	var eventsRaw []byte

	events := eventsdb.GetCurrent().LoadEvents(height)

	if len(events) > 0 {
		eventsRaw, err = cdc.MarshalJSON(events)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			err = json.NewEncoder(w).Encode(Response{
				Code: 0,
				Log:  err.Error(),
			})

			if err != nil {
				panic(err)
			}
			return
		}
	}

	response := BlockResponse{
		Hash:         block.Block.Hash(),
		Height:       block.Block.Height,
		Time:         block.Block.Time,
		NumTxs:       block.Block.NumTxs,
		TotalTxs:     block.Block.TotalTxs,
		Transactions: txs,
		Precommits:   precommits,
		BlockReward:  rewards.GetRewardForBlock(uint64(height)).String(),
		Size:         size,
		Events:       eventsRaw,
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: response,
	})

	if err != nil {
		panic(err)
	}
}
