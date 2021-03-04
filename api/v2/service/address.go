package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/formula"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

type stakeUser struct {
	Value    *big.Int
	BipValue *big.Int
}

// Address returns coins list, balance and transaction count of an address.
func (s *Service) Address(ctx context.Context, req *pb.AddressRequest) (*pb.AddressResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Address[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	balances := cState.Accounts().GetBalances(address)
	var res pb.AddressResponse

	totalStakesGroupByCoin := map[types.CoinID]*big.Int{}

	res.Balance = make([]*pb.AddressBalance, 0, len(balances))
	for _, coin := range balances {
		totalStakesGroupByCoin[coin.Coin.ID] = coin.Value
		coinModel := cState.Coins().GetCoin(coin.Coin.ID)
		res.Balance = append(res.Balance, &pb.AddressBalance{
			Coin: &pb.Coin{
				Id:     uint64(coin.Coin.ID),
				Symbol: coinModel.GetFullSymbol(),
			},
			Value:    coin.Value.String(),
			BipValue: customCoinBipBalance(coin.Value, coinModel).String(),
		})
	}

	if req.Delegated {
		cState.Candidates().LoadCandidates()
		if timeoutStatus := s.checkTimeout(ctx, "LoadCandidates"); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		cState.Candidates().LoadStakes()
		if timeoutStatus := s.checkTimeout(ctx, "LoadStakes"); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		var userDelegatedStakesGroupByCoin = map[types.CoinID]*stakeUser{}
		allCandidates := cState.Candidates().GetCandidates()
		for i, candidate := range allCandidates {
			userStakes := userStakes(candidate.PubKey, address, cState)

			if timeoutStatus := s.checkTimeout(ctx, fmt.Sprintf("userStakes of %s [%d]", candidate.PubKey, i)); timeoutStatus != nil {
				return nil, timeoutStatus.Err()
			}

			for coin, userStake := range userStakes {
				stake, ok := userDelegatedStakesGroupByCoin[coin]
				if !ok {
					stake = &stakeUser{
						Value:    big.NewInt(0),
						BipValue: big.NewInt(0),
					}
				}
				stake.Value.Add(stake.Value, userStake.Value)
				stake.BipValue.Add(stake.BipValue, userStake.BipValue)
				userDelegatedStakesGroupByCoin[coin] = stake
			}
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		res.Delegated = make([]*pb.AddressDelegatedBalance, 0, len(userDelegatedStakesGroupByCoin))
		for coinID, delegatedStake := range userDelegatedStakesGroupByCoin {
			coinModel := cState.Coins().GetCoin(coinID)
			res.Delegated = append(res.Delegated, &pb.AddressDelegatedBalance{
				Coin: &pb.Coin{
					Id:     uint64(coinID),
					Symbol: coinModel.GetFullSymbol(),
				},
				Value:            delegatedStake.Value.String(),
				DelegateBipValue: delegatedStake.BipValue.String(),
				BipValue:         customCoinBipBalance(delegatedStake.Value, coinModel).String(),
			})

			totalStake, ok := totalStakesGroupByCoin[coinID]
			if !ok {
				totalStake = big.NewInt(0)
				totalStakesGroupByCoin[coinID] = totalStake
			}
			totalStake.Add(totalStake, delegatedStake.Value)
		}
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	coinsBipValue := big.NewInt(0)
	res.Total = make([]*pb.AddressBalance, 0, len(totalStakesGroupByCoin))
	for coinID, stake := range totalStakesGroupByCoin {
		coinModel := cState.Coins().GetCoin(coinID)
		balance := customCoinBipBalance(stake, coinModel)
		if req.Delegated {
			res.Total = append(res.Total, &pb.AddressBalance{
				Coin: &pb.Coin{
					Id:     uint64(coinID),
					Symbol: coinModel.GetFullSymbol(),
				},
				Value:    stake.String(),
				BipValue: balance.String(),
			})
		}
		coinsBipValue.Add(coinsBipValue, balance)
	}
	res.BipValue = coinsBipValue.String()
	res.TransactionCount = cState.Accounts().GetNonce(address)
	return &res, nil
}

func customCoinBipBalance(valueToSell *big.Int, coinFrom *coins.Model) *big.Int {
	if coinFrom.ID().IsBaseCoin() {
		return valueToSell
	}

	if coinFrom.IsToken() {
		return big.NewInt(0)
	}

	return formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
}

func userStakes(c types.Pubkey, address types.Address, state *state.CheckState) map[types.CoinID]*stakeUser {
	var userStakes = map[types.CoinID]*stakeUser{}

	stakes := state.Candidates().GetStakes(c)

	for _, stake := range stakes {
		if stake.Owner != address {
			continue
		}
		userStakes[stake.Coin] = &stakeUser{
			Value:    stake.Value,
			BipValue: stake.BipValue,
		}
	}

	return userStakes
}
