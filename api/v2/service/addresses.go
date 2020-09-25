package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

// Addresses returns list of addresses.
func (s *Service) Addresses(ctx context.Context, req *pb.AddressesRequest) (*pb.AddressesResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.AddressesResponse), status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 && req.Delegated {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakes()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	response := &pb.AddressesResponse{
		Addresses: make(map[string]*pb.AddressesResponse_Result, len(req.Addresses)),
	}

	for _, addr := range req.Addresses {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return response, timeoutStatus.Err()
		}

		if len(addr) < 3 {
			return new(pb.AddressesResponse), status.Error(codes.InvalidArgument, fmt.Sprintf("invalid address %s", addr))
		}

		decodeString, err := hex.DecodeString(addr[2:])
		if err != nil {
			return new(pb.AddressesResponse), status.Error(codes.InvalidArgument, err.Error())
		}
		address := types.BytesToAddress(decodeString)

		balances := cState.Accounts().GetBalances(address)
		var res pb.AddressesResponse_Result

		totalStakesGroupByCoin := map[types.CoinID]*big.Int{}

		res.Balance = make([]*pb.AddressBalance, 0, len(balances))
		for _, coin := range balances {
			totalStakesGroupByCoin[coin.Coin.ID] = coin.Value
			res.Balance = append(res.Balance, &pb.AddressBalance{
				Coin: &pb.Coin{
					Id:     uint64(coin.Coin.ID),
					Symbol: cState.Coins().GetCoin(coin.Coin.ID).Symbol().String(),
				},
				Value:    coin.Value.String(),
				BipValue: customCoinBipBalance(coin.Coin.ID, coin.Value, cState).String(),
			})
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.AddressesResponse), timeoutStatus.Err()
		}

		if req.Delegated {
			var userDelegatedStakesGroupByCoin = map[types.CoinID]*stakeUser{}
			allCandidates := cState.Candidates().GetCandidates()
			for _, candidate := range allCandidates {
				userStakes := userStakes(candidate.PubKey, address, cState)
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
				return new(pb.AddressesResponse), timeoutStatus.Err()
			}

			res.Delegated = make([]*pb.AddressDelegatedBalance, 0, len(userDelegatedStakesGroupByCoin))
			for coinID, delegatedStake := range userDelegatedStakesGroupByCoin {
				res.Delegated = append(res.Delegated, &pb.AddressDelegatedBalance{
					Coin: &pb.Coin{
						Id:     uint64(coinID),
						Symbol: cState.Coins().GetCoin(coinID).Symbol().String(),
					},
					Value:            delegatedStake.Value.String(),
					DelegateBipValue: delegatedStake.BipValue.String(),
					BipValue:         customCoinBipBalance(coinID, delegatedStake.Value, cState).String(),
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
			return new(pb.AddressesResponse), timeoutStatus.Err()
		}

		coinsBipValue := big.NewInt(0)
		res.Total = make([]*pb.AddressBalance, 0, len(totalStakesGroupByCoin))
		for coinID, stake := range totalStakesGroupByCoin {
			balance := customCoinBipBalance(coinID, stake, cState)
			if req.Delegated {
				res.Total = append(res.Total, &pb.AddressBalance{
					Coin: &pb.Coin{
						Id:     uint64(coinID),
						Symbol: cState.Coins().GetCoin(coinID).Symbol().String(),
					},
					Value:    stake.String(),
					BipValue: balance.String(),
				})
			}
			coinsBipValue.Add(coinsBipValue, balance)
		}
		res.BipValue = coinsBipValue.String()
		res.TransactionCount = uint64(cState.Accounts().GetNonce(address))

		response.Addresses[addr] = &res
	}

	return response, nil
}
