package service

import (
	"bytes"
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/empty"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (s *Service) Genesis(context.Context, *empty.Empty) (*pb.GenesisResponse, error) {
	result, err := s.client.Genesis()
	if err != nil {
		return new(pb.GenesisResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	var bb bytes.Buffer
	if _, err := bb.Write(result.Genesis.AppState); err != nil {
		return new(pb.GenesisResponse), status.Error(codes.Internal, err.Error())
	}
	appState := &_struct.Struct{Fields: make(map[string]*_struct.Value)}
	if err := (&jsonpb.Unmarshaler{}).Unmarshal(&bb, appState); err != nil {
		return new(pb.GenesisResponse), status.Error(codes.Internal, err.Error())
	}

	return &pb.GenesisResponse{
		GenesisTime: result.Genesis.GenesisTime.Format(time.RFC3339Nano),
		ChainId:     result.Genesis.ChainID,
		ConsensusParams: &pb.GenesisResponse_ConsensusParams{
			Block: &pb.GenesisResponse_ConsensusParams_Block{
				MaxBytes:   fmt.Sprintf("%d", result.Genesis.ConsensusParams.Block.MaxBytes),
				MaxGas:     fmt.Sprintf("%d", result.Genesis.ConsensusParams.Block.MaxGas),
				TimeIotaMs: fmt.Sprintf("%d", result.Genesis.ConsensusParams.Block.TimeIotaMs),
			},
			Evidence: &pb.GenesisResponse_ConsensusParams_Evidence{
				MaxAgeNumBlocks: fmt.Sprintf("%d", result.Genesis.ConsensusParams.Evidence.MaxAgeNumBlocks),
				MaxAgeDuration:  fmt.Sprintf("%d", result.Genesis.ConsensusParams.Evidence.MaxAgeDuration),
			},
			Validator: &pb.GenesisResponse_ConsensusParams_Validator{
				PublicKeyTypes: result.Genesis.ConsensusParams.Validator.PubKeyTypes,
			},
		},
		AppHash:  result.Genesis.AppHash.String(),
		AppState: appState,
	}, nil
}
