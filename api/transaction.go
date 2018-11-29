package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/common"
)

func Transaction(hash []byte) (*TransactionResponse, error) {
	tx, err := client.Tx(hash, false)
	if err != nil {
		return nil, err
	}

	if tx.Height > blockchain.LastCommittedHeight() {
		return nil, errors.New("Tx not found")
	}

	decodedTx, _ := transaction.DecodeFromBytes(tx.Tx)
	sender, _ := decodedTx.Sender()

	tags := make(map[string]string)

	for _, tag := range tx.TxResult.Tags {
		switch string(tag.Key) {
		case "tx.type":
			tags[string(tag.Key)] = fmt.Sprintf("%X", tag.Value)
		default:
			tags[string(tag.Key)] = string(tag.Value)
		}
	}

	return &TransactionResponse{
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
		Code:     tx.TxResult.Code,
		Log:      tx.TxResult.Log,
	}, nil
}
