package service

import (
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/google/uuid"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

const (
	SubscribeTimeout = 5 * time.Second
)

// Returns a subscription for events by query.
func (s *Service) Subscribe(request *pb.SubscribeRequest, stream pb.ApiService_SubscribeServer) error {

	if s.client.NumClients() >= s.minterCfg.RPC.MaxSubscriptionClients {
		return status.Error(codes.Internal, fmt.Sprintf("max_subscription_clients %d reached", s.minterCfg.RPC.MaxSubscriptionClients))
	}

	s.client.Logger.Info("Subscribe to query", "query", request.Query)

	subCtx, cancel := context.WithTimeout(stream.Context(), SubscribeTimeout)
	defer cancel()
	subscriber := uuid.New().String()
	sub, err := s.client.Subscribe(subCtx, subscriber, request.Query)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer func() {
		if err := s.client.UnsubscribeAll(context.Background(), subscriber); err != nil {
			s.client.Logger.Error(err.Error())
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case msg, ok := <-sub:
			if !ok {
				return nil
			}
			res, err := subscribeResponse(msg)
			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}
			if err := stream.Send(res); err != nil {
				return err
			}
		}
	}
}

func subscribeResponse(msg core_types.ResultEvent) (*pb.SubscribeResponse, error) {
	events := make([]*pb.SubscribeResponse_Event, 0, len(msg.Events))
	for key, eventSlice := range msg.Events {
		events = append(events, &pb.SubscribeResponse_Event{
			Key:    key,
			Events: eventSlice,
		})
	}

	byteData, err := json.Marshal(msg.Data)
	if err != nil {
		return nil, err
	}

	data, err := encodeToStruct(byteData)
	if err != nil {
		return nil, err
	}

	return &pb.SubscribeResponse{Query: msg.Query, Data: data, Events: events}, nil
}
