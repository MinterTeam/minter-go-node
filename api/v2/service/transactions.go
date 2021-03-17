package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"strings"
)

// Transactions return transactions by query.
func (s *Service) Transactions(ctx context.Context, req *pb.TransactionsRequest) (*pb.TransactionsResponse, error) {
	page := int(req.Page)
	perPage := int(req.PerPage)
	rpcResult, err := s.client.TxSearch(ctx, req.Query, false, &page, &perPage, "desc")
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	lenTx := len(rpcResult.Txs)
	result := make([]*pb.TransactionResponse, 0, lenTx)
	if lenTx != 0 {

		cState := s.blockchain.CurrentState()

		for _, tx := range rpcResult.Txs {

			if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
				return nil, timeoutStatus.Err()
			}

			decodedTx, _ := transaction.DecodeFromBytes(tx.Tx)
			sender, _ := decodedTx.Sender()

			tags := make(map[string]string)
			var gas int
			for _, tag := range tx.TxResult.Events[0].Attributes {
				key := string(tag.Key)
				value := string(tag.Value)
				tags[key] = value
				if key == "tx.gas" {
					gas, _ = strconv.Atoi(value)
				}
			}

			data, err := encode(decodedTx.GetDecodedData(), cState.Coins())
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			result = append(result, &pb.TransactionResponse{
				Hash:     "Mt" + strings.ToLower(hex.EncodeToString(tx.Tx.Hash())),
				RawTx:    fmt.Sprintf("%x", []byte(tx.Tx)),
				Height:   uint64(tx.Height),
				Index:    uint64(tx.Index),
				From:     sender.String(),
				Nonce:    decodedTx.Nonce,
				GasPrice: uint64(decodedTx.GasPrice),
				GasCoin: &pb.Coin{
					Id:     uint64(decodedTx.GasCoin),
					Symbol: cState.Coins().GetCoin(decodedTx.GasCoin).GetFullSymbol(),
				},
				Gas:     uint64(gas),
				HexType: decodedTx.Type.String(),
				Type:    decodedTx.Type.UInt64(),
				Data:    data,
				Payload: decodedTx.Payload,
				Tags:    tags,
				Code:    uint64(tx.TxResult.Code),
				Log:     tx.TxResult.Log,
			})
		}
	}
	return &pb.TransactionsResponse{
		Transactions: result,
	}, nil
}
