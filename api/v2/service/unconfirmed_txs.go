package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnconfirmedTxs returns unconfirmed transactions.
func (s *Service) UnconfirmedTxs(ctx context.Context, req *pb.UnconfirmedTxsRequest) (*pb.UnconfirmedTxsResponse, error) {
	txs, err := s.client.UnconfirmedTxs(int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	transactions := make([]string, 0, len(txs.Txs))
	for _, tx := range txs.Txs {
		transactions = append(transactions, tx.String())
	}
	return &pb.UnconfirmedTxsResponse{
		TransactionCount:  uint64(txs.Count),
		TotalTransactions: uint64(txs.Total),
		TotalBytes:        uint64(txs.TotalBytes),
		Transactions:      transactions,
	}, nil
}
