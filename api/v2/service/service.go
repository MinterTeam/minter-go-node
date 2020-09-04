package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/tendermint/go-amino"
	tmNode "github.com/tendermint/tendermint/node"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

type Service struct {
	cdc        *amino.Codec
	blockchain *minter.Blockchain
	client     *rpc.Local
	tmNode     *tmNode.Node
	minterCfg  *config.Config
	version    string
}

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

func (s *Service) TimeoutDuration() time.Duration {
	return s.minterCfg.APIv2TimeoutDuration
}

func (s *Service) checkTimeout(ctx context.Context) *status.Status {
	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			return status.New(codes.Canceled, ctx.Err().Error())
		}

		return status.New(codes.DeadlineExceeded, ctx.Err().Error())
	default:
		return nil
	}
}
