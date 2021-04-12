package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

// Addresses returns list of addresses.
func (s *Service) Addresses(ctx context.Context, req *pb.AddressesRequest) (*pb.AddressesResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 && req.Delegated {
		cState.Candidates().LoadCandidates()
		if timeoutStatus := s.checkTimeout(ctx, "LoadCandidates"); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		cState.Candidates().LoadStakes()
		if timeoutStatus := s.checkTimeout(ctx, "LoadStakes"); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
	}

	response := &pb.AddressesResponse{
		Addresses: make(map[string]*pb.AddressesResponse_Result, len(req.Addresses)),
	}

	for _, addr := range req.Addresses {

		if !strings.HasPrefix(strings.Title(addr), "Mx") {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid address %s", addr))
		}

		decodeString, err := hex.DecodeString(addr[2:])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		address := types.BytesToAddress(decodeString)

		balances := cState.Accounts().GetBalances(address)
		var res pb.AddressesResponse_Result

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
			if timeoutStatus := s.checkTimeout(ctx, "Delegated"); timeoutStatus != nil {
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

		response.Addresses[addr] = &res
	}

	return response, nil
}
