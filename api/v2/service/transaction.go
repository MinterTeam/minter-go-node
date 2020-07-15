package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Transaction(ctx context.Context, req *pb.TransactionRequest) (*pb.TransactionResponse, error) {
	if len(req.Hash) < 3 {
		return new(pb.TransactionResponse), status.Error(codes.InvalidArgument, "invalid hash")
	}
	decodeString, err := hex.DecodeString(req.Hash[2:])
	if err != nil {
		return new(pb.TransactionResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	tx, err := s.client.Tx(decodeString, false)
	if err != nil {
		return new(pb.TransactionResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	decodedTx, _ := transaction.TxDecoder.DecodeFromBytes(tx.Tx)
	sender, _ := decodedTx.Sender()

	tags := make(map[string]string)
	for _, tag := range tx.TxResult.Events[0].Attributes {
		tags[string(tag.Key)] = string(tag.Value)
	}

	dataStruct, err := s.encodeTxData(decodedTx)
	if err != nil {
		return new(pb.TransactionResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	return &pb.TransactionResponse{
		Hash:     tmbytes.HexBytes(tx.Tx.Hash()).String(),
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
	}, nil
}

func (s *Service) encodeTxData(decodedTx *transaction.Transaction) (*_struct.Struct, error) {
	var (
		err error
		b   []byte
	)
	switch decodedTx.Type {
	case transaction.TypeSend:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SendData))
	case transaction.TypeRedeemCheck:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.RedeemCheckData))
	case transaction.TypeSellCoin:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SellCoinData))
	case transaction.TypeSellAllCoin:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SellAllCoinData))
	case transaction.TypeBuyCoin:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.BuyCoinData))
	case transaction.TypeCreateCoin:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.CreateCoinData))
	case transaction.TypeDeclareCandidacy:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.DeclareCandidacyData))
	case transaction.TypeDelegate:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.DelegateData))
	case transaction.TypeSetCandidateOnline:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SetCandidateOnData))
	case transaction.TypeSetCandidateOffline:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SetCandidateOffData))
	case transaction.TypeUnbond:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.UnbondData))
	case transaction.TypeMultisend:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.MultisendData))
	case transaction.TypeCreateMultisig:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.CreateMultisigData))
	case transaction.TypeEditCandidate:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.EditCandidateData))
	case transaction.TypeSetHaltBlock:
		b, err = s.cdc.MarshalJSON(decodedTx.GetDecodedData().(*transaction.SetHaltBlockData))
	default:
		return nil, errors.New("unknown tx type")
	}

	if err != nil {
		return nil, err
	}

	dataStruct := &_struct.Struct{}
	if err := dataStruct.UnmarshalJSON(b); err != nil {
		return nil, err
	}

	return dataStruct, nil
}
