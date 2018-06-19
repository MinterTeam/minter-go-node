package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tmlibs/common"
	"math/big"
	"net/http"
)

type TransactionResponse struct {
	Hash     common.HexBytes   `json:"hash"`
	Height   int64             `json:"height"`
	Index    uint32            `json:"index"`
	TxResult ResponseDeliverTx `json:"tx_result"`
	From     string            `json:"from"`
	Nonce    uint64            `json:"nonce"`
	GasPrice *big.Int          `json:"gasPrice"`
	Type     byte              `json:"type"`
	Data     transaction.Data  `json:"data"`
	Payload  []byte            `json:"payload"`
}

type ResponseDeliverTx struct {
	Code      uint32          `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Data      []byte          `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Log       string          `protobuf:"bytes,3,opt,name=log,proto3" json:"log,omitempty"`
	Info      string          `protobuf:"bytes,4,opt,name=info,proto3" json:"info,omitempty"`
	GasWanted int64           `protobuf:"varint,5,opt,name=gas_wanted,json=gasWanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   int64           `protobuf:"varint,6,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`
	Tags      []common.KVPair `protobuf:"bytes,7,rep,name=tags" json:"tags,omitempty"`
	Fee       common.KI64Pair `protobuf:"bytes,8,opt,name=fee" json:"fee"`
}

type ResultTxSearch struct {
	Txs        []*core_types.ResultTx `json:"txs"`
	TotalCount int                    `json:"total_count"`
}

func Transactions(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("query")

	rpcResult := new(ResultTxSearch)
	_, err := client.Call("tx_search", map[string]interface{}{
		"query":    query,
		"page":     1,
		"per_page": 100,
	}, rpcResult)

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

	for i := range rpcResult.Txs {
		tx := rpcResult.Txs[i]
		decodedTx, _ := transaction.DecodeFromBytes(tx.Tx)
		sender, _ := decodedTx.Sender()

		result[i] = TransactionResponse{
			Hash:   common.HexBytes(tx.Tx.Hash()),
			Height: tx.Height,
			Index:  tx.Index,
			TxResult: ResponseDeliverTx{
				Code:      tx.TxResult.Code,
				Data:      tx.TxResult.Data,
				Log:       tx.TxResult.Log,
				Info:      tx.TxResult.Info,
				GasWanted: tx.TxResult.GasWanted,
				GasUsed:   tx.TxResult.GasUsed,
			},
			From:     sender.String(),
			Nonce:    decodedTx.Nonce,
			GasPrice: decodedTx.GasPrice,
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
