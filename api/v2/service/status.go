package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

// Status returns current min gas price.
func (s *Service) Status(ctx context.Context, _ *empty.Empty) (*pb.StatusResponse, error) {
	result, err := s.client.Status(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	cState := s.blockchain.CurrentState()

	reward, _ := cState.App().Reward()
	if reward == nil {
		reward = s.blockchain.RewardCounter().GetRewardForBlock(uint64(result.SyncInfo.LatestBlockHeight))
	}
	return &pb.StatusResponse{
		Version:           s.version,
		Network:           result.NodeInfo.Network,
		Moniker:           result.NodeInfo.Moniker,
		LatestBlockHash:   fmt.Sprintf("%X", result.SyncInfo.LatestBlockHash),
		LatestAppHash:     fmt.Sprintf("%X", result.SyncInfo.LatestAppHash),
		LatestBlockHeight: uint64(result.SyncInfo.LatestBlockHeight),
		LatestBlockTime:   result.SyncInfo.LatestBlockTime.Format(time.RFC3339Nano),
		KeepLastStates:    uint64(s.minterCfg.BaseConfig.KeepLastStates),
		TotalSlashed:      cState.App().GetTotalSlashed().String(),
		CurrentEmission:   s.blockchain.GetEmission().String(),
		BlockReward:       reward.String(),
		CatchingUp:        result.SyncInfo.CatchingUp,
		PublicKey:         fmt.Sprintf("Mp%x", result.ValidatorInfo.PubKey.Bytes()[:]),
		NodeId:            string(result.NodeInfo.ID()),
		InitialHeight:     s.blockchain.InitialHeight() + 1,
	}, nil
}
