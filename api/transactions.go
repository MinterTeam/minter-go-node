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
	TxResult ResponseDeliverTx `json:"tx_result"`
	From     string            `json:"from"`
	Nonce    uint64            `json:"nonce"`
	GasPrice *big.Int          `json:"gas_price"`
	GasCoin  types.CoinSymbol  `json:"gas_coin"`
	Type     byte              `json:"type"`
	Data     transaction.Data  `json:"data"`
	Payload  []byte            `json:"payload"`
}

type ResponseDeliverTx struct {
	Code      uint32          `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Data      []byte          `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Log       string          `protobuf:"bytes,3,opt,name=log,proto3" json:"log,omitempty"`
	Info      string          `protobuf:"bytes,4,opt,name=info,proto3" json:"info,omitempty"`
	GasWanted int64           `protobuf:"varint,5,opt,name=gas_wanted,json=gas_wanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   int64           `protobuf:"varint,6,opt,name=gas_used,json=gas_used,proto3" json:"gas_used,omitempty"`
	Tags      []common.KVPair `protobuf:"bytes,7,rep,name=tags" json:"tags,omitempty"`
	Fee       common.KI64Pair `protobuf:"bytes,8,opt,name=fee" json:"fee"`
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

		result[i] = TransactionResponse{
			Hash:   common.HexBytes(tx.Tx.Hash()),
			RawTx:  fmt.Sprintf("%x", []byte(tx.Tx)),
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
			GasCoin:  decodedTx.GasCoin,
			Type:     decodedTx.Type,
			Data:     decodedTx.GetDecodedData(),
			Payload:  decodedTx.Payload,
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
