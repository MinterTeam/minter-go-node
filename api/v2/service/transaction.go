package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/tendermint/tendermint/libs/common"
)

func (s *Service) Transaction(_ context.Context, req *pb.TransactionRequest) (*pb.TransactionResponse, error) {
	tx, err := s.client.Tx([]byte(req.Hash), false)
	if err != nil {
		return &pb.TransactionResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	decodedTx, _ := transaction.TxDecoder.DecodeFromBytes(tx.Tx)
	sender, _ := decodedTx.Sender()

	tags := make(map[string]string)
	for _, tag := range tx.TxResult.Events[0].Attributes {
		tags[string(tag.Key)] = string(tag.Value)
	}

	data, err := s.encodeTxData(decodedTx)
	if err != nil {
		return &pb.TransactionResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	dataStruct := &_struct.Struct{Fields: make(map[string]*_struct.Value)}
	err = json.Unmarshal(data, dataStruct.Fields)
	if err != nil {
		return &pb.TransactionResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	return &pb.TransactionResponse{
		Result: &pb.TransactionResult{
			Hash:     common.HexBytes(tx.Tx.Hash()).String(),
			RawTx:    fmt.Sprintf("%x", []byte(tx.Tx)),
			Height:   fmt.Sprintf("%d", tx.Height),
			Index:    fmt.Sprintf("%d", tx.Index),
			From:     sender.String(),
			Nonce:    fmt.Sprintf("%d", decodedTx.Nonce),
			GasPrice: fmt.Sprintf("%d", decodedTx.GasPrice),
			GasCoin:  decodedTx.GasCoin.String(),
			Gas:      fmt.Sprintf("%d", decodedTx.Gas()),
			Type:     fmt.Sprintf("%d", uint8(decodedTx.Type)),
			Data:     dataStruct,
			Payload:  decodedTx.Payload,
			Tags:     tags,
			Code:     fmt.Sprintf("%d", tx.TxResult.Code),
			Log:      tx.TxResult.Log,
		},
	}, nil
}

func (s *Service) encodeTxData(decodedTx *transaction.Transaction) ([]byte, error) {
	switch decodedTx.Type {
	case transaction.TypeSend:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SendData))
	case transaction.TypeRedeemCheck:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.RedeemCheckData))
	case transaction.TypeSellCoin:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SellCoinData))
	case transaction.TypeSellAllCoin:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SellAllCoinData))
	case transaction.TypeBuyCoin:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.BuyCoinData))
	case transaction.TypeCreateCoin:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.CreateCoinData))
	case transaction.TypeDeclareCandidacy:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.DeclareCandidacyData))
	case transaction.TypeDelegate:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.DelegateData))
	case transaction.TypeSetCandidateOnline:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SetCandidateOnData))
	case transaction.TypeSetCandidateOffline:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SetCandidateOffData))
	case transaction.TypeUnbond:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.UnbondData))
	case transaction.TypeMultisend:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.MultisendData))
	case transaction.TypeCreateMultisig:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.CreateMultisigData))
	case transaction.TypeEditCandidate:
		return s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.EditCandidateData))
	}

	return nil, errors.New("unknown tx type")
}
