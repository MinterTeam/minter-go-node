package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
)

func (s *Service) Events(_ context.Context, req *pb.EventsRequest) (*pb.EventsResponse, error) {
	events := s.blockchain.GetEventsDB().LoadEvents(uint32(req.Height))
	resultEvents := make([]*pb.EventsResponse_Result_Event, 0, len(events))
	return &pb.EventsResponse{
		Result: &pb.EventsResponse_Result{
			Events: resultEvents,
		},
	}, nil
}
