package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) UnconfirmedTxs(_ context.Context, req *pb.UnconfirmedTxsRequest) (*pb.UnconfirmedTxsResponse, error) {
	txs, err := s.client.UnconfirmedTxs(int(req.Limit))
	if err != nil {
		return new(pb.UnconfirmedTxsResponse), status.Error(codes.Internal, err.Error())
	}
	return &pb.UnconfirmedTxsResponse{
		TransactionsCount: fmt.Sprintf("%d", txs.Count),
		TotalTransactions: fmt.Sprintf("%d", txs.Total),
		TotalBytes:        fmt.Sprintf("%d", txs.TotalBytes),
	}, nil
}
