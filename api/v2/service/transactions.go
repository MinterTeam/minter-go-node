package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/tendermint/tendermint/libs/bytes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Transactions(_ context.Context, req *pb.TransactionsRequest) (*pb.TransactionsResponse, error) {
	page := int(req.Page)
	if page == 0 {
		page = 1
	}
	perPage := int(req.PerPage)
	if perPage == 0 {
		perPage = 100
	}

	rpcResult, err := s.client.TxSearch(req.Query, false, page, perPage, "desc")
	if err != nil {
		return new(pb.TransactionsResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	result := make([]*pb.TransactionResponse, 0, len(rpcResult.Txs))
	for _, tx := range rpcResult.Txs {
		decodedTx, _ := transaction.TxDecoder.DecodeFromBytes(tx.Tx)
		sender, _ := decodedTx.Sender()

		tags := make(map[string]string)
		for _, tag := range tx.TxResult.Events[0].Attributes {
			tags[string(tag.Key)] = string(tag.Value)
		}

		dataStruct, err := s.encodeTxData(decodedTx)
		if err != nil {
			return new(pb.TransactionsResponse), status.Error(codes.FailedPrecondition, err.Error())
		}

		result = append(result, &pb.TransactionResponse{
			Hash:     bytes.HexBytes(tx.Tx.Hash()).String(),
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
		})
	}

	return &pb.TransactionsResponse{
		Transactions: result,
	}, nil
}
