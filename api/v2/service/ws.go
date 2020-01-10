package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/golang/protobuf/jsonpb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/uuid"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

const (
	SubscribeTimeout = 5 * time.Second
)

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
		if err := s.client.UnsubscribeAll(stream.Context(), subscriber); err != nil {
			s.client.Logger.Error(err.Error())
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err() //todo: when to do it?
		case msg, ok := <-sub:
			if !ok {
				return nil
			}
			res, err := subscribeResponse(msg)
			if err := stream.Send(res); err != nil {
				return err
			}
			if err != nil {
				s.client.Logger.Error(err.Error())
				break
			}
		}
	}

	return nil
}

func subscribeResponse(msg core_types.ResultEvent) (*pb.SubscribeResponse, error) {
	var events []*pb.SubscribeResponse_Event

	byteData, err := json.Marshal(msg.Data)
	if err != nil {
		return nil, err
	}

	var bb bytes.Buffer
	bb.Write(byteData)

	data := &_struct.Struct{Fields: make(map[string]*_struct.Value)}
	if err := (&jsonpb.Unmarshaler{}).Unmarshal(&bb, data); err != nil {
		return nil, err
	}

	return &pb.SubscribeResponse{Query: msg.Query, Data: data, Events: events}, nil
}
