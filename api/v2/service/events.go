package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Events returns events at given height.
func (s *Service) Events(ctx context.Context, req *pb.EventsRequest) (*pb.EventsResponse, error) {
	currentHeight := s.blockchain.Height()
	if req.Height > currentHeight {
		return nil, status.Errorf(codes.NotFound, "wanted to load target %d but only found up to %d", req.Height, currentHeight)
	}

	height := uint32(req.Height)
	events := s.blockchain.GetEventsDB().LoadEvents(height)
	resultEvents := make([]*_struct.Struct, 0, len(events))
	for _, event := range events {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
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

		marshalJSON, err := s.cdc.MarshalJSON(event)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}

		data, err := encodeToStruct(marshalJSON)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		resultEvents = append(resultEvents, data)
	}
	return &pb.EventsResponse{
		Events: resultEvents,
	}, nil
}
