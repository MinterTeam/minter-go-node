package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"time"
)

// Genesis returns genesis file.
func (s *Service) Genesis(ctx context.Context, _ *empty.Empty) (*pb.GenesisResponse, error) {
	result, err := s.client.Genesis()
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var appState pb.GenesisResponse_AppState
	err = protojson.Unmarshal(result.Genesis.AppState, &appState)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	return &pb.GenesisResponse{
		GenesisTime: result.Genesis.GenesisTime.Format(time.RFC3339Nano),
		ChainId:     result.Genesis.ChainID,
		ConsensusParams: &pb.GenesisResponse_ConsensusParams{
			Block: &pb.GenesisResponse_ConsensusParams_Block{
				MaxBytes:   result.Genesis.ConsensusParams.Block.MaxBytes,
				MaxGas:     result.Genesis.ConsensusParams.Block.MaxGas,
				TimeIotaMs: result.Genesis.ConsensusParams.Block.TimeIotaMs,
			},
			Evidence: &pb.GenesisResponse_ConsensusParams_Evidence{
				MaxAgeNumBlocks: result.Genesis.ConsensusParams.Evidence.MaxAgeNumBlocks,
				MaxAgeDuration:  int64(result.Genesis.ConsensusParams.Evidence.MaxAgeDuration),
			},
			Validator: &pb.GenesisResponse_ConsensusParams_Validator{
				PubKeyTypes: result.Genesis.ConsensusParams.Validator.PubKeyTypes,
			},
		},
		AppHash:  result.Genesis.AppHash.String(),
		AppState: &appState,
	}, nil
}
