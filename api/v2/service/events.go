package service

import (
	"bytes"
	"context"
	"encoding/json"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/jsonpb"
	_struct "github.com/golang/protobuf/ptypes/struct"
)

func (s *Service) Events(_ context.Context, req *pb.EventsRequest) (*pb.EventsResponse, error) {
	height := uint32(req.Height)
	events := s.blockchain.GetEventsDB().LoadEvents(height)
	resultEvents := make([]*pb.EventsResponse_Event, 0, len(events))
	for _, event := range events {
		byteData, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}

		var bb bytes.Buffer
		bb.Write(byteData)
		data := &_struct.Struct{Fields: make(map[string]*_struct.Value)}
		if err := (&jsonpb.Unmarshaler{}).Unmarshal(&bb, data); err != nil {
			return nil, err
		}

		var t string
		switch event.(type) {
		case *eventsdb.RewardEvent:
			t = "minter/RewardEvent"
		case *eventsdb.SlashEvent:
			t = "minter/SlashEvent"
		case *eventsdb.UnbondEvent:
			t = "minter/UnbondEvent"
		default:
			t = "Undefined Type"
		}

		resultEvents = append(resultEvents, &pb.EventsResponse_Event{Type: t, Value: data})
	}
	return &pb.EventsResponse{
		Events: resultEvents,
	}, nil
}
