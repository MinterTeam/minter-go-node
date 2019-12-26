package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) UnconfirmedTxs(_ context.Context, req *pb.UnconfirmedTxsRequest) (*pb.UnconfirmedTxsResponse, error) {
	txs, err := s.client.UnconfirmedTxs(int(req.Limit))
	if err != nil {
		return new(pb.UnconfirmedTxsResponse), status.Error(codes.Internal, err.Error())
	}
	return &pb.UnconfirmedTxsResponse{
		NTxs:       fmt.Sprintf("%d", txs.Count),
		Total:      fmt.Sprintf("%d", txs.Total),
		TotalBytes: fmt.Sprintf("%d", txs.TotalBytes),
	}, nil
}
