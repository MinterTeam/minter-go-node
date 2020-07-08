package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (s *Service) Genesis(ctx context.Context, _ *empty.Empty) (*pb.GenesisResponse, error) {
	result, err := s.client.Genesis()
	if err != nil {
		return new(pb.GenesisResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return new(pb.GenesisResponse), timeoutStatus.Err()
	}

	appState := &_struct.Struct{}
	if err := appState.UnmarshalJSON(result.Genesis.AppState); err != nil {
		return new(pb.GenesisResponse), status.Error(codes.Internal, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return new(pb.GenesisResponse), timeoutStatus.Err()
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
