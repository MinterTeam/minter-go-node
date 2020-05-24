package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"time"
)

type TransactionsResponse struct{
	Height       int64            `json:"height"`
	Time         time.Time        `json:"time"`
	Hash         string           `json:"hash"`
	Type        uint8             `json:"type"`
	From         string           `json:"from"`
	Nonce       uint64            `json:"nonce"`
	Data        json.RawMessage   `json:"data"`
	Payload     []byte            `json:"payload"`
	ServiceData []byte            `json:"service_data"`
	Gas         int64             `json:"gas"`
	GasCoin     string            `json:"gas_coin"`
	Code        uint32            `json:"code,omitempty"`
	Log         string            `json:"log,omitempty"`
}


func TxsFromBlock(height int64) ([]TransactionsResponse, error) {
	block, _ := client.Block(&height)
	blockResults, _ := client.BlockResults(&height)

	var tr 		TransactionsResponse 
	var trs 	[]TransactionsResponse 

	if len(block.Block.Data.Txs) > 0 {
		tr.Height = block.Block.Height
		tr.Time = block.Block.Time
	}

	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.TxDecoder.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		if len(blockResults.TxsResults) == 0 {
			break
		}

		data, _ := encodeTxData(tx)

		tr.Hash         = fmt.Sprintf("Mt%x", rawTx.Hash())
		tr.Type         = uint8(tx.Type)
		tr.From         = sender.String()
		tr.Nonce	= tx.Nonce
		tr.Data	 	= data
		tr.Payload      = tx.Payload
		tr.ServiceData	 = tx.ServiceData
		tr.Gas		 = tx.Gas()
		tr.GasCoin      = tx.GasCoin.String()
		tr.Code         = blockResults.TxsResults[i].Code
		tr.Log		 = blockResults.TxsResults[i].Log
		trs = append(trs,tr)
	}
	
	return trs , nil
}