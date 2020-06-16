package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
)

type CoinBalance struct {
	Coin             string `json:"coin"`
	Value            string `json:"value"`
	BipValue         string `json:"bip_value"`
	DelegateBipValue string `json:"delegate_bip_value,omitempty"`
}

type AddressBalanceResponse struct {
	Freecoins        []*CoinBalance `json:"freecoins"`
	Delegated        []*CoinBalance `json:"delegated"`
	Total            []*CoinBalance `json:"total"`
	TransactionCount uint64         `json:"transaction_count"`
	Bipvalue         string         `json:"bipvalue"`
}

type AddressesBalancesResponse struct {
	Address types.Address           `json:"address"`
	Balance *AddressBalanceResponse `json:"balance"`
}

type UserStake struct {
	Value    *big.Int
	BipValue *big.Int
}

func CustomCoinBipBalance(coinToSell types.CoinSymbol, valueToSell *big.Int, cState *state.CheckState) *big.Int {
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

func MakeAddressBalance(address types.Address, height int) (*AddressBalanceResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	if height != 0 {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakes()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	balances := cState.Accounts().GetBalances(address)
	var response AddressBalanceResponse

	totalStakesGroupByCoin := map[types.CoinSymbol]*big.Int{}

	response.Freecoins = make([]*CoinBalance, 0, len(balances))
	for coin, value := range balances {
		totalStakesGroupByCoin[coin] = value
		response.Freecoins = append(response.Freecoins, &CoinBalance{
			Coin:     coin.String(),
			Value:    value.String(),
			BipValue: CustomCoinBipBalance(coin, value, cState).String(),
		})
	}

	var userDelegatedStakesGroupByCoin = map[types.CoinSymbol]*UserStake{}
	allCandidates := cState.Candidates().GetCandidates()
	for _, candidate := range allCandidates {
		userStakes := UserStakes(candidate.PubKey, address, cState)
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

	response.Delegated = make([]*CoinBalance, 0, len(userDelegatedStakesGroupByCoin))
	for coin, delegatedStake := range userDelegatedStakesGroupByCoin {
		response.Delegated = append(response.Delegated, &CoinBalance{
			Coin:             coin.String(),
			Value:            delegatedStake.Value.String(),
			DelegateBipValue: delegatedStake.BipValue.String(),
			BipValue:         CustomCoinBipBalance(coin, delegatedStake.Value, cState).String(),
		})

		totalStake, ok := totalStakesGroupByCoin[coin]
		if !ok {
			totalStake = big.NewInt(0)
			totalStakesGroupByCoin[coin] = totalStake
		}
		totalStake.Add(totalStake, delegatedStake.Value)
	}

	coinsBipValue := big.NewInt(0)
	response.Total = make([]*CoinBalance, 0, len(totalStakesGroupByCoin))
	for coin, stake := range totalStakesGroupByCoin {
		balance := CustomCoinBipBalance(coin, stake, cState)
		response.Total = append(response.Total, &CoinBalance{
			Coin:     coin.String(),
			Value:    stake.String(),
			BipValue: balance.String(),
		})
		coinsBipValue.Add(coinsBipValue, balance)
	}

	response.TransactionCount = cState.Accounts().GetNonce(address)
	response.Bipvalue = coinsBipValue.String()

	return &response, nil
}

func UserStakes(c types.Pubkey, address types.Address, state *state.CheckState) map[types.CoinSymbol]*UserStake {
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
