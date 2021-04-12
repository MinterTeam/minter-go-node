package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/MinterTeam/minter-go-node/coreV2/rewards"
	"github.com/MinterTeam/node-grpc-gateway/api_pb"
	tmNode "github.com/tendermint/tendermint/node"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

// Service is gRPC implementation ApiServiceServer
type Service struct {
	blockchain *minter.Blockchain
	client     *rpc.Local
	tmNode     *tmNode.Node
	minterCfg  *config.Config
	version    string
	rewards    *rewards.Reward
	api_pb.UnimplementedApiServiceServer
}

// NewService create gRPC server implementation
func NewService(blockchain *minter.Blockchain, client *rpc.Local, node *tmNode.Node, minterCfg *config.Config, version string, reward *rewards.Reward) *Service {
	return &Service{
		rewards:    reward,
		blockchain: blockchain,
		client:     client,
		minterCfg:  minterCfg,
		version:    version,
		tmNode:     node,
	}
}

// TimeoutDuration returns timeout gRPC request
func (s *Service) TimeoutDuration() time.Duration {
	return s.minterCfg.APIv2TimeoutDuration
}

// Version returns version app
func (s *Service) Version() string {
	return s.version
}

func (s *Service) createError(statusErr *status.Status, data string) error {
	if len(data) == 0 {
		return statusErr.Err()
	}

	detailsMap, err := encodeToStruct([]byte(data))
	if err != nil {
		s.client.Logger.Error(err.Error())
		return statusErr.Err()
	}

	withDetails, err := statusErr.WithDetails(detailsMap)
	if err != nil {
		s.client.Logger.Error(err.Error())
		return statusErr.Err()
	}

	return withDetails.Err()
}

func (s *Service) checkTimeout(ctx context.Context, operations ...string) *status.Status {
	select {
	case <-ctx.Done():
		timeoutResponse := status.FromContextError(ctx.Err())
		if len(operations) == 0 {
			return timeoutResponse
		}

		detailsMap := map[string]string{}
		for i, msg := range operations {
			postfix := ""
			if i > 0 {
				postfix = strconv.Itoa(i)
			}
			detailsMap["operation"+postfix] = msg
		}
		details, err := toStruct(detailsMap)
		if err != nil {
			grpclog.Infof("Failed to write response timeout details: %v", err)
			return timeoutResponse
		}

		timeoutResponseWithDetails, err := timeoutResponse.WithDetails(details)
		if err != nil {
			grpclog.Infof("Failed to write response timeout details: %v", err)
			return timeoutResponse
		}

		return timeoutResponseWithDetails
	default:
		return nil
	}
}
