package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/tendermint/tendermint/libs/bytes"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

type TransactionResponse struct {
	Hash     string            `json:"hash"`
	RawTx    string            `json:"raw_tx"`
	Height   int64             `json:"height"`
	Index    uint32            `json:"index"`
	From     string            `json:"from"`
	Nonce    uint64            `json:"nonce"`
	Gas      int64             `json:"gas"`
	GasPrice uint32            `json:"gas_price"`
	GasCoin  Coin              `json:"gas_coin"`
	Type     uint8             `json:"type"`
	Data     json.RawMessage   `json:"data"`
	Payload  []byte            `json:"payload"`
	Tags     map[string]string `json:"tags"`
	Code     uint32            `json:"code,omitempty"`
	Log      string            `json:"log,omitempty"`
}

type ResultTxSearch struct {
	Txs        []*core_types.ResultTx `json:"txs"`
	TotalCount int                    `json:"total_count"`
}

func Transactions(query string, page, perPage int) (*[]TransactionResponse, error) {
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 100
	}

	rpcResult, err := client.TxSearch(query, false, page, perPage, "desc")
	if err != nil {
		return nil, err
	}

	result := make([]TransactionResponse, len(rpcResult.Txs))
	for i, tx := range rpcResult.Txs {
		decodedTx, _ := transaction.TxDecoder.DecodeFromBytes(tx.Tx)
		sender, _ := decodedTx.Sender()

		tags := make(map[string]string)
		for _, tag := range tx.TxResult.Events[0].Attributes {
			tags[string(tag.Key)] = string(tag.Value)
		}

		data, err := encodeTxData(decodedTx)
		if err != nil {
			return nil, err
		}

		result[i] = TransactionResponse{
			Hash:     bytes.HexBytes(tx.Tx.Hash()).String(),
			RawTx:    fmt.Sprintf("%x", []byte(tx.Tx)),
			Height:   tx.Height,
			Index:    tx.Index,
			From:     sender.String(),
			Nonce:    decodedTx.Nonce,
			Gas:      decodedTx.Gas(),
			GasPrice: decodedTx.GasPrice,
			GasCoin:  Coin{
				ID:     decodedTx.GasCoin.Uint32(),
				Symbol: "",
			},
			Type:     uint8(decodedTx.Type),
			Data:     data,
			Payload:  decodedTx.Payload,
			Tags:     tags,
			Code:     tx.TxResult.Code,
			Log:      tx.TxResult.Log,
		}
	}

	return &result, nil
}
