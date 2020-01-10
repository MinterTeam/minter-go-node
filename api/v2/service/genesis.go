package service

import (
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (s *Service) Genesis(context.Context, *empty.Empty) (*pb.GenesisResponse, error) {
	result, err := s.client.Genesis()
	if err != nil {
		return new(pb.GenesisResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	appState := new(pb.GenesisResponse_AppState)
	err = json.Unmarshal(result.Genesis.AppState, appState)
	if err != nil {
		return new(pb.GenesisResponse), status.Error(codes.FailedPrecondition, err.Error())
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
				MaxAge: fmt.Sprintf("%d", result.Genesis.ConsensusParams.Evidence.MaxAge),
			},
			Validator: &pb.GenesisResponse_ConsensusParams_Validator{
				PublicKeyTypes: result.Genesis.ConsensusParams.Validator.PubKeyTypes,
			},
		},
		AppHash:  result.Genesis.AppHash.String(),
		AppState: appState,
	}, nil
}
