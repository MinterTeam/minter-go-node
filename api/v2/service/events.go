package service

import (
	"context"
	"encoding/json"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Events returns events at given height.
func (s *Service) Events(ctx context.Context, req *pb.EventsRequest) (*pb.EventsResponse, error) {
	height := uint32(req.Height)
	events := s.blockchain.GetEventsDB().LoadEvents(height)
	resultEvents := make([]*pb.EventsResponse_Event, 0, len(events))
	for _, event := range events {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.EventsResponse), timeoutStatus.Err()
		}

		var find = true
		for _, s := range req.Search {
			if event.AddressString() == s || event.ValidatorPubKeyString() == s {
				find = true
				break
			}
			find = false
		}
		if !find {
			continue
		}

		b, err := json.Marshal(event)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		data, err := encodeToStruct(b)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		var t string
		switch event.(type) {
		case *eventsdb.RewardEvent:
			t = "minter/RewardEvent"
		case *eventsdb.SlashEvent:
			t = "minter/SlashEvent"
		case *eventsdb.UnbondEvent:
			t = "minter/UnbondEvent"
		case *eventsdb.StakeKickEvent:
			t = "minter/StakeKickEvent"
		default:
			t = "Undefined Type"
		}

		resultEvents = append(resultEvents, &pb.EventsResponse_Event{Type: t, Value: data})
	}
	return &pb.EventsResponse{
		Events: resultEvents,
	}, nil
}
