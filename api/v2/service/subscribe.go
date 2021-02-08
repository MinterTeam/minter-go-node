package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"time"
)

const (
	subscribeTimeout = 5 * time.Second
)

// Subscribe returns a subscription for events by query.
func (s *Service) Subscribe(request *pb.SubscribeRequest, stream pb.ApiService_SubscribeServer) error {

	if s.client.NumClients() >= s.minterCfg.RPC.MaxSubscriptionClients {
		return status.Error(codes.ResourceExhausted, fmt.Sprintf("max_subscription_clients %d reached", s.minterCfg.RPC.MaxSubscriptionClients))
	}

	s.client.Logger.Info("Subscribe to query", "query", request.Query)

	ctx, cancel := context.WithTimeout(stream.Context(), subscribeTimeout)
	defer cancel()

	subscriber, _ := peer.FromContext(ctx)
	remote := subscriber.Addr.String()
	sub, err := s.client.Subscribe(ctx, remote, request.Query)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	defer func() {
		if err := s.client.Unsubscribe(context.Background(), remote, request.Query); err != nil {
			s.client.Logger.Error(err.Error())
		}
	}()

	ctxSubscribeConnection, _ := context.WithTimeout(stream.Context(), s.minterCfg.WSConnectionDuration)

	for {
		select {
		case <-ctxSubscribeConnection.Done():
			return status.FromContextError(ctxSubscribeConnection.Err()).Err()
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

	data, err := toStruct(msg.Data)
	if err != nil {
		return nil, err
	}

	return &pb.SubscribeResponse{Query: msg.Query, Data: data, Events: events}, nil
}
