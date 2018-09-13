package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/rpc/core/types"
	"math/big"
	"net/http"
)

type TransactionResponse struct {
	Hash     common.HexBytes   `json:"hash"`
	RawTx    string            `json:"raw_tx"`
	Height   int64             `json:"height"`
	Index    uint32            `json:"index"`
	From     string            `json:"from"`
	Nonce    uint64            `json:"nonce"`
	GasPrice *big.Int          `json:"gas_price"`
	GasCoin  types.CoinSymbol  `json:"gas_coin"`
	GasUsed  int64             `json:"gas_used"`
	Type     byte              `json:"type"`
	Data     transaction.Data  `json:"data"`
	Payload  []byte            `json:"payload"`
	Tags     map[string]string `json:"tags"`
}

type ResultTxSearch struct {
	Txs        []*core_types.ResultTx `json:"txs"`
	TotalCount int                    `json:"total_count"`
}

func Transactions(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("query")

	rpcResult, err := client.TxSearch(query, false, 1, 100)

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

	result := make([]TransactionResponse, len(rpcResult.Txs))

	for i, tx := range rpcResult.Txs {
		decodedTx, _ := transaction.DecodeFromBytes(tx.Tx)
		sender, _ := decodedTx.Sender()

		tags := make(map[string]string)

		for _, tag := range tx.TxResult.Tags {
			tags[string(tag.Key)] = string(tag.Value)
		}

		result[i] = TransactionResponse{
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
		}
	}

	err = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result,
	})

	if err != nil {
		panic(err)
	}
}
