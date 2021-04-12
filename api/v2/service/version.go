package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
)

// Status returns current min gas price.
func (s *Service) VersionNetwork(ctx context.Context, _ *pb.VersionNetworkRequest) (*pb.VersionNetworkResponse, error) {
	versions := s.blockchain.UpdateVersions()

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var cur string
	var list []*pb.VersionNetworkResponse_Version
	for _, version := range versions {
		list = append(list, &pb.VersionNetworkResponse_Version{
			Name:   version.Name,
			Height: version.Height,
		})
		cur = version.Name
	}
	return &pb.VersionNetworkResponse{
		Current:  cur,
		Versions: list,
	}, nil
}
