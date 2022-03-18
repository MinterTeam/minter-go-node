package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"strings"
)

// Frozen returns frozen balance.
func (s *Service) Frozen(ctx context.Context, req *pb.FrozenRequest) (*pb.FrozenResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	var reqCoin *coins.Model

	if req.CoinId != nil {
		coinID := types.CoinID(req.CoinId.GetValue())
		reqCoin = cState.Coins().GetCoin(coinID)
		if reqCoin == nil {
			return nil, s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists("", coinID.String())))
		}
	}
	var frozen []*pb.FrozenResponse_Frozen

	for i := s.blockchain.Height(); i <= s.blockchain.Height()+types.GetUnbondPeriod(); i++ {

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
			var moveToCandidateKey *wrapperspb.StringValue
			if len(fund.MoveToCandidate) != 0 {
				moveToCandidateKey = wrapperspb.String(cState.Candidates().PubKey(fund.MoveToCandidate[0]).String())
			}
			var fromCandidateKey *wrapperspb.StringValue
			if fund.CandidateKey != nil {
				fromCandidateKey = wrapperspb.String(fund.CandidateKey.String())
			}
			frozen = append(frozen, &pb.FrozenResponse_Frozen{
				Height:       funds.Height(),
				Address:      fund.Address.String(),
				CandidateKey: fromCandidateKey,
				Coin: &pb.Coin{
					Id:     uint64(fund.Coin),
					Symbol: coin.GetFullSymbol(),
				},
				Value:              fund.Value.String(),
				MoveToCandidateKey: moveToCandidateKey,
			})
		}
	}

	return &pb.FrozenResponse{Frozen: frozen}, nil
}

func (s *Service) FrozenAll(ctx context.Context, req *pb.FrozenAllRequest) (*pb.FrozenResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	addresses := make(map[string]struct{}, len(req.Addresses))
	for _, address := range req.Addresses {
		if !strings.HasPrefix(strings.Title(address), "Mx") {
			return nil, status.Error(codes.InvalidArgument, "invalid address")
		}
		addresses[address] = struct{}{}
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	coinIDs := make(map[types.CoinID]*coins.Model, len(req.CoinIds))
	for _, coinId := range req.CoinIds {
		coinID := types.CoinID(coinId)
		reqCoin := cState.Coins().GetCoin(coinID)
		if reqCoin == nil {
			return nil, s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists("", coinID.String())))
		}
		coinIDs[coinID] = reqCoin
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	startHeight := req.StartHeight
	if startHeight == 0 {
		startHeight = s.blockchain.Height()
	}
	endHeight := req.EndHeight
	if endHeight == 0 {
		endHeight = startHeight + types.GetUnbondPeriod()
	}

	var frozen []*pb.FrozenResponse_Frozen

	fundsAll := cState.FrozenFunds().GetFrozenFundsAll(ctx, startHeight, endHeight+1)
	for _, funds := range fundsAll {
		if funds == nil {
			continue
		}

		for _, fund := range funds.List {
			if len(req.Addresses) != 0 {
				if _, ok := addresses[fund.Address.String()]; !ok {
					continue
				}
			}

			var coin *coins.Model
			if len(req.CoinIds) != 0 {
				reqCoin, ok := coinIDs[fund.Coin]
				if !ok {
					continue
				}
				coin = reqCoin
			} else {
				reqCoin, ok := coinIDs[fund.Coin]
				if ok {
					coin = reqCoin
				} else {
					coin = cState.Coins().GetCoin(fund.Coin)
					coinIDs[fund.Coin] = coin
				}

			}

			var moveToCandidateKey *wrapperspb.StringValue
			if len(fund.MoveToCandidate) != 0 {
				moveToCandidateKey = wrapperspb.String(cState.Candidates().PubKey(fund.MoveToCandidate[0]).String())
			}
			var fromCandidateKey *wrapperspb.StringValue
			if fund.CandidateKey != nil {
				fromCandidateKey = wrapperspb.String(fund.CandidateKey.String())
			}
			frozen = append(frozen, &pb.FrozenResponse_Frozen{
				Height:       funds.Height(),
				Address:      fund.Address.String(),
				CandidateKey: fromCandidateKey,
				Coin: &pb.Coin{
					Id:     uint64(fund.Coin),
					Symbol: coin.GetFullSymbol(),
				},
				Value:              fund.Value.String(),
				MoveToCandidateKey: moveToCandidateKey,
			})
		}
	}

	return &pb.FrozenResponse{Frozen: frozen}, nil
}
