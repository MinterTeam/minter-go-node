package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/tendermint/go-amino"
	tmNode "github.com/tendermint/tendermint/node"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	"google.golang.org/grpc/status"
	"time"
)

// Service is gRPC implementation ApiServiceServer
type Service struct {
	cdc        *amino.Codec
	blockchain *minter.Blockchain
	client     *rpc.Local
	tmNode     *tmNode.Node
	minterCfg  *config.Config
	version    string
	api_pb.UnimplementedApiServiceServer
}

// NewService create gRPC server implementation
func NewService(cdc *amino.Codec, blockchain *minter.Blockchain, client *rpc.Local, node *tmNode.Node, minterCfg *config.Config, version string) *Service {
	return &Service{
		cdc:        cdc,
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

func (s *Service) checkTimeout(ctx context.Context) *status.Status {
	select {
	case <-ctx.Done():
		return status.FromContextError(ctx.Err())
	default:
		return nil
	}
}
