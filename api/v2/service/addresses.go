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

func (s *Service) Addresses(ctx context.Context, req *pb.AddressesRequest) (*pb.AddressesResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.AddressesResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	if req.Height != 0 && req.Delegated {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakes()
		cState.Unlock()
	}

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

		totalStakesGroupByCoin := map[types.CoinSymbol]*big.Int{}

		res.Balance = make(map[string]*pb.AddressBalance, len(balances))
		for coin, value := range balances {
			totalStakesGroupByCoin[coin] = value
			res.Balance[coin.String()] = &pb.AddressBalance{
				Value:    value.String(),
				BipValue: customCoinBipBalance(coin, value, cState).String(),
			}
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.AddressesResponse), timeoutStatus.Err()
		}

		if req.Delegated {
			var userDelegatedStakesGroupByCoin = map[types.CoinSymbol]*UserStake{}
			allCandidates := cState.Candidates().GetCandidates()
			for _, candidate := range allCandidates {
				userStakes := userStakes(candidate.PubKey, address, cState)
				for coin, userStake := range userStakes {
					stake, ok := userDelegatedStakesGroupByCoin[coin]
					if !ok {
						stake = &UserStake{
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

			res.Delegated = make(map[string]*pb.AddressDelegatedBalance, len(userDelegatedStakesGroupByCoin))
			for coin, delegatedStake := range userDelegatedStakesGroupByCoin {
				res.Delegated[coin.String()] = &pb.AddressDelegatedBalance{
					Value:            delegatedStake.Value.String(),
					DelegateBipValue: delegatedStake.BipValue.String(),
					BipValue:         customCoinBipBalance(coin, delegatedStake.Value, cState).String(),
				}

				totalStake, ok := totalStakesGroupByCoin[coin]
				if !ok {
					totalStake = big.NewInt(0)
					totalStakesGroupByCoin[coin] = totalStake
				}
				totalStake.Add(totalStake, delegatedStake.Value)
			}
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return response, timeoutStatus.Err()
		}

		coinsBipValue := big.NewInt(0)
		res.Total = make(map[string]*pb.AddressBalance, len(totalStakesGroupByCoin))
		for coin, stake := range totalStakesGroupByCoin {
			balance := customCoinBipBalance(coin, stake, cState)
			if req.Delegated {
				res.Total[coin.String()] = &pb.AddressBalance{
					Value:    stake.String(),
					BipValue: balance.String(),
				}
			}
			coinsBipValue.Add(coinsBipValue, balance)
		}
		res.BipValue = coinsBipValue.String()
		res.TransactionsCount = fmt.Sprintf("%d", cState.Accounts().GetNonce(address))

		response.Addresses[addr] = &res
	}

	return response, nil
}
