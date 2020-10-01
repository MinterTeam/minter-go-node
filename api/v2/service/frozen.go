package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

// Frozen returns frozen balance.
func (s *Service) Frozen(ctx context.Context, req *pb.FrozenRequest) (*pb.FrozenResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	cState := s.blockchain.CurrentState()
	cState.RLock()
	defer cState.RUnlock()

	var reqCoin *coins.Model

	if req.CoinId != nil {
		coinID := types.CoinID(req.CoinId.GetValue())
		reqCoin = cState.Coins().GetCoin(coinID)
		if reqCoin == nil {
			return nil, s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists("", coinID.String())))
		}
	}
	var frozen []*pb.FrozenResponse_Frozen

	cState.FrozenFunds().GetFrozenFunds(s.blockchain.Height())

	for i := s.blockchain.Height(); i <= s.blockchain.Height()+candidates.UnbondPeriod; i++ {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		funds := cState.FrozenFunds().GetFrozenFunds(i)
		if funds == nil {
			continue
		}

		for _, fund := range funds.List {
			if fund.Address.String() != req.Address {
				continue
			}
			coin := reqCoin
			if coin == nil {
				coin = cState.Coins().GetCoin(fund.Coin)
			} else {
				if coin.ID() != fund.Coin {
					continue
				}
			}
			frozen = append(frozen, &pb.FrozenResponse_Frozen{
				Height:       funds.Height(),
				Address:      fund.Address.String(),
				CandidateKey: fund.CandidateKey.String(),
				Coin: &pb.Coin{
					Id:     uint64(fund.Coin),
					Symbol: coin.GetFullSymbol(),
				},
				Value: fund.Value.String(),
			})
		}
	}

	return &pb.FrozenResponse{Frozen: frozen}, nil
}
