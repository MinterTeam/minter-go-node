package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

type UserStake struct {
	Value    *big.Int
	BipValue *big.Int
}

func (s *Service) Address(ctx context.Context, req *pb.AddressRequest) (*pb.AddressResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
		return new(pb.AddressResponse), status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Address[2:])
	if err != nil {
		return new(pb.AddressResponse), status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.AddressResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	if req.Height != 0 && req.Delegated {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakes()
		cState.Unlock()
	}

	balances := cState.Accounts().GetBalances(address)
	var response pb.AddressResponse

	totalStakesGroupByCoin := map[types.CoinSymbol]*big.Int{}

	response.Balance = make(map[string]*pb.AddressBalance, len(balances))
	for coin, value := range balances {
		totalStakesGroupByCoin[coin] = value
		response.Balance[coin.String()] = &pb.AddressBalance{
			Value:    value.String(),
			BipValue: customCoinBipBalance(coin, value, cState).String(),
		}
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return new(pb.AddressResponse), timeoutStatus.Err()
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
			return new(pb.AddressResponse), timeoutStatus.Err()
		}

		response.Delegated = make(map[string]*pb.AddressDelegatedBalance, len(userDelegatedStakesGroupByCoin))
		for coin, delegatedStake := range userDelegatedStakesGroupByCoin {
			response.Delegated[coin.String()] = &pb.AddressDelegatedBalance{
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
		return new(pb.AddressResponse), timeoutStatus.Err()
	}

	coinsBipValue := big.NewInt(0)
	response.Total = make(map[string]*pb.AddressBalance, len(totalStakesGroupByCoin))
	for coin, stake := range totalStakesGroupByCoin {
		balance := customCoinBipBalance(coin, stake, cState)
		if req.Delegated {
			response.Total[coin.String()] = &pb.AddressBalance{
				Value:    stake.String(),
				BipValue: balance.String(),
			}
		}
		coinsBipValue.Add(coinsBipValue, balance)
	}
	response.BipValue = coinsBipValue.String()

	response.TransactionsCount = fmt.Sprintf("%d", cState.Accounts().GetNonce(address))
	return &response, nil
}

func customCoinBipBalance(coinToSell types.CoinSymbol, valueToSell *big.Int, cState *state.CheckState) *big.Int {
	coinToBuy := types.StrToCoinSymbol("BIP")

	if coinToSell == coinToBuy {
		return valueToSell
	}

	if coinToSell == types.GetBaseCoin() {
		coin := cState.Coins().GetCoin(coinToBuy)
		return formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	}

	if coinToBuy == types.GetBaseCoin() {
		coin := cState.Coins().GetCoin(coinToSell)
		return formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	}

	coinFrom := cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)
	basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
	return formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
}

func userStakes(c types.Pubkey, address types.Address, state *state.CheckState) map[types.CoinSymbol]*UserStake {
	var userStakes = map[types.CoinSymbol]*UserStake{}

	stakes := state.Candidates().GetStakes(c)

	for _, stake := range stakes {
		if stake.Owner != address {
			continue
		}
		userStakes[stake.Coin] = &UserStake{
			Value:    stake.Value,
			BipValue: stake.BipValue,
		}
	}

	return userStakes
}
