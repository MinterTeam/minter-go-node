package service

import (
	"context"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
)

func (s *Service) Events(_ context.Context, req *pb.EventsRequest) (*pb.EventsResponse, error) {
	events := s.blockchain.GetEventsDB().LoadEvents(uint32(req.Height))
	resultEvents := make([]*pb.EventsResponse_Event, 0, len(events))
	return &pb.EventsResponse{
		Events: resultEvents,
	}, nil
}
