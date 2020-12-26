package api

import (
	"encoding/json"

	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/transaction/encoder"
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

func Transactions(query string, page, perPage int) (*[]json.RawMessage, error) {
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

	cState, err := GetStateForHeight(0)
	if err != nil {
		return nil, err
	}

	result := make([]json.RawMessage, 0, len(rpcResult.Txs))
	for _, tx := range rpcResult.Txs {
		decodedTx, _ := transaction.DecodeFromBytes(tx.Tx)
		txJsonEncoder := encoder.NewTxEncoderJSON(cState)
		response, err := txJsonEncoder.Encode(decodedTx, tx)
		if err != nil {
			return nil, err
		}

		result = append(result, response)
	}

	return &result, nil
}
