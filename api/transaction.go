package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"github.com/tendermint/tendermint/libs/common"
)

func Transaction(hash []byte) (*TransactionResponse, error) {
	tx, err := client.Tx(hash, false)
	if err != nil {
		return nil, err
	}

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

	return &TransactionResponse{
		Hash:     common.HexBytes(tx.Tx.Hash()).String(),
		RawTx:    fmt.Sprintf("%x", []byte(tx.Tx)),
		Height:   tx.Height,
		Index:    tx.Index,
		From:     sender.String(),
		Nonce:    decodedTx.Nonce,
		GasPrice: decodedTx.GasPrice,
		GasCoin:  decodedTx.GasCoin.String(),
		Gas:      decodedTx.Gas(),
		Type:     uint8(decodedTx.Type),
		Data:     data,
		Payload:  decodedTx.Payload,
		Tags:     tags,
		Code:     tx.TxResult.Code,
		Log:      tx.TxResult.Log,
	}, nil
}

func encodeTxData(decodedTx *transaction.Transaction) ([]byte, error) {
	switch decodedTx.Type {
	case transaction.TypeSend:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SendData))
	case transaction.TypeRedeemCheck:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.RedeemCheckData))
	case transaction.TypeSellCoin:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SellCoinData))
	case transaction.TypeSellAllCoin:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SellAllCoinData))
	case transaction.TypeBuyCoin:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.BuyCoinData))
	case transaction.TypeCreateCoin:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.CreateCoinData))
	case transaction.TypeDeclareCandidacy:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.DeclareCandidacyData))
	case transaction.TypeDelegate:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.DelegateData))
	case transaction.TypeSetCandidateOnline:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SetCandidateOnData))
	case transaction.TypeSetCandidateOffline:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SetCandidateOffData))
	case transaction.TypeUnbond:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.UnbondData))
	case transaction.TypeMultisend:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.MultisendData))
	case transaction.TypeCreateMultisig:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.CreateMultisigData))
	case transaction.TypeEditCandidate:
		return cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.EditCandidateData))
	}

	return nil, rpctypes.RPCError{Code: 500, Message: "unknown tx type"}
}
