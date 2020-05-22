package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Halts(_ context.Context, _ *pb.HaltsRequest) (*pb.HaltsResponse, error) {
	return &pb.HaltsResponse{}, status.Error(codes.Unimplemented, fmt.Sprintf("method Halts not implemented in version %s", s.version))
}
