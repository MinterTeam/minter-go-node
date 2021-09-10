package service

import (
	"context"
	"encoding/base64"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	tmjson "github.com/tendermint/tendermint/libs/json"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"time"
)

// Genesis returns genesis file.
func (s *Service) Genesis(ctx context.Context, _ *empty.Empty) (*pb.GenesisResponse, error) {
	var result = &ctypes.ResultGenesis{}
	if true { // TODO: add GenesisChunked endpoint
		var chanks []byte
		chank, err := s.client.GenesisChunked(ctx, 0)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		data, err := base64.StdEncoding.DecodeString(chank.Data)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		chanks = append(chanks, data...)
		for i := 1; i < chank.TotalChunks; i++ {
			chank, err := s.client.GenesisChunked(ctx, uint(i))
			if err != nil {
				return nil, status.Error(codes.FailedPrecondition, err.Error())
			}
			data, err := base64.StdEncoding.DecodeString(chank.Data)
			if err != nil {
				return nil, status.Error(codes.FailedPrecondition, err.Error())
			}
			chanks = append(chanks, data...)
		}

		genesis := &types.GenesisDoc{}
		err = tmjson.Unmarshal(chanks, genesis)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		result.Genesis = genesis
	} else {
		var err error
		result, err = s.client.Genesis(ctx)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var appState pb.GenesisResponse_AppState
	err := protojson.Unmarshal(result.Genesis.AppState, &appState)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	return &pb.GenesisResponse{
		GenesisTime:   result.Genesis.GenesisTime.Format(time.RFC3339Nano),
		ChainId:       result.Genesis.ChainID,
		InitialHeight: uint64(result.Genesis.InitialHeight),
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
