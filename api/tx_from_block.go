package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"time"
)

type TransactionsResponse struct {
	Height      int64           `json:"height"`
	Time        time.Time       `json:"time"`
	Hash        string          `json:"hash"`
	Type        uint8           `json:"type"`
	From        string          `json:"from"`
	Nonce       uint64          `json:"nonce"`
	Data        json.RawMessage `json:"data"`
	Payload     []byte          `json:"payload"`
	ServiceData []byte          `json:"service_data"`
	Gas         int64           `json:"gas"`
	GasCoin     string          `json:"gas_coin"`
	Code        uint32          `json:"code,omitempty"`
	Log         string          `json:"log,omitempty"`
}

func TxsFromBlock(height int64) ([]*TransactionsResponse, error) {
	block, _ := client.Block(&height)
	blockResults, _ := client.BlockResults(&height)

	countTx := len(block.Block.Data.Txs)
	trs := make([]*TransactionsResponse, 0, countTx)

	if len(blockResults.TxsResults) != countTx {
		return trs, nil
	}

	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.TxDecoder.DecodeFromBytes(rawTx)

		sender, _ := tx.Sender()
		data, _ := encodeTxData(tx)

		tr := &TransactionsResponse{
			Height:      block.Block.Height,
			Time:        block.Block.Time,
			Hash:        fmt.Sprintf("Mt%x", rawTx.Hash()),
			Type:        uint8(tx.Type),
			From:        sender.String(),
			Nonce:       tx.Nonce,
			Data:        data,
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
			Gas:         tx.Gas(),
			GasCoin:     tx.GasCoin.String(),
			Code:        blockResults.TxsResults[i].Code,
			Log:         blockResults.TxsResults[i].Log,
		}

		trs = append(trs, tr)
	}

	return trs, nil
}
