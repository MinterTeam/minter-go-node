package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/golang/protobuf/jsonpb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) SendTransaction(_ context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	result, err := s.client.BroadcastTxSync([]byte(req.Tx))
	if err != nil {
		statusErr := status.New(codes.InvalidArgument, err.Error())
		detailsMap := make(map[string]*_struct.Value)
		//todo add parser method
		withDetails, err := statusErr.WithDetails(&_struct.Struct{Fields: detailsMap})
		if err != nil {
			s.client.Logger.Error(err.Error())
			return new(pb.SendTransactionResponse), statusErr.Err()
		}
		return new(pb.SendTransactionResponse), withDetails.Err()
	}

	if result.Code != 0 {
		statusErr := status.New(codes.InvalidArgument, result.Log)
		detailsMap := &_struct.Struct{Fields: make(map[string]*_struct.Value)}
		marshal, err := json.Marshal(map[string]string{
			"code": fmt.Sprintf("%d", result.Code),
			"log":  result.Log,
		})
		if err != nil {
			s.client.Logger.Error(err.Error())
			return new(pb.SendTransactionResponse), statusErr.Err()
		}

		var bb bytes.Buffer
		bb.Write(marshal)
		if err := (&jsonpb.Unmarshaler{}).Unmarshal(&bb, detailsMap); err != nil {
			return nil, err
		}

		withDetails, err := statusErr.WithDetails(detailsMap)
		if err != nil {
			s.client.Logger.Error(err.Error())
			return new(pb.SendTransactionResponse), statusErr.Err()
		}
		return &pb.SendTransactionResponse{}, withDetails.Err()
	}

	return &pb.SendTransactionResponse{
		Code: fmt.Sprintf("%d", result.Code),
		Data: result.Data.String(),
		Log:  result.Log,
		Hash: result.Hash.String(),
	}, nil
}
