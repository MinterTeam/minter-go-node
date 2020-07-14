package api

import (
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/transaction/encoder"
)

func Transaction(hash []byte) (*encoder.TransactionResponse, error) {
	tx, err := client.Tx(hash, false)
	if err != nil {
		return nil, err
	}

	decodedTx, _ := transaction.TxDecoder.DecodeFromBytes(tx.Tx)

	cState, err := GetStateForHeight(int(tx.Height))
	if err != nil {
		return nil, err
	}

	txJsonEncoder := encoder.NewTxEncoderJSON(cState)

	return txJsonEncoder.Encode(decodedTx, tx)
}
