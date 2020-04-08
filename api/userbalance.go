package api

import (
 	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
	"fmt"
)

 

 
type UserbalanceResponse struct {
	Freecoins         	[]UStake  `json:"freecoins"`
	Delegated         	[]UStake  `json:"delegated"`
	TransactionCount 	uint64         `json:"transaction_count"`
	Bipvalue     		*big.Int       `json:"bipvalue"`
} 





func CustomCoinBipBalance(coinToSellString string, valueToSell *big.Int, height int)(*big.Int) {
cState, err := GetStateForHeight(height)
	err = err

	var result *big.Int

	coinToSell := types.StrToCoinSymbol(coinToSellString)
	coinToBuy := types.StrToCoinSymbol("BIP")

	if coinToSell == coinToBuy {
		result = valueToSell
	}else{
		switch {
		case coinToSell == types.GetBaseCoin():
			coin := cState.Coins.GetCoin(coinToBuy)
			result = formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
		case coinToBuy == types.GetBaseCoin():
			coin := cState.Coins.GetCoin(coinToSell)
			result = formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
		default:
			coinFrom := cState.Coins.GetCoin(coinToSell)
			coinTo := cState.Coins.GetCoin(coinToBuy)
			basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
			result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
		}

	}

return result 
}

 
 

func Userbalance(
	height int, address types.Address) (*UserbalanceResponse,
	error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}
		cState.RLock()
		defer cState.RUnlock()

	balances := cState.Accounts.GetBalances(address)
	var response UserbalanceResponse
	coinsbipvalue := big.NewInt(0)
 	response.Freecoins = make([]UStake, len(balances))
	n := 0 	
	for k, valueToSell := range balances {
		coinToSell := k.String()
		result := CustomCoinBipBalance(coinToSell, valueToSell, height)
		response.Freecoins[n] = UStake{
			Coin: 	k.String(),
			Value: valueToSell.String(),
			BipValue: result.String(),
			}
		n = n + 1
 
	} 
 	
	
	response.Delegated = GroupUserStakes(height, address.String())


	for _, coin := range response.Freecoins{
		response.Total = append(response.Total,coin)
	}

	n = 0 
	t1:=big.NewInt(0)
	t2:=big.NewInt(0)

	for _,coin:= range response.Delegated {
		for i,_:= range response.Total{
			if response.Total[i].Coin == coin.Coin {
				fmt.Sscan(response.Total[i].Value, t1)
				fmt.Sscan(coin.Value, t2)
				t1.Add(t1,t2)
				response.Total[i].Value = t1.String()
				n=1
				break
			}
		}
		if n == 0 {
			response.Total= append(response.Total,coin)
		}
	}
	coinsbipvalue = big.NewInt(0)
	n = 0 	
	for _, coin := range response.Total {
		coinToSell := coin.Coin
		valueToSell:=big.NewInt(0)
		fmt.Sscan(coin.Value, valueToSell)
		result := CustomCoinBipBalance(coinToSell, valueToSell, height)

		if coinsbipvalue == nil {
			coinsbipvalue = result 
		} else {
			coinsbipvalue.Add(coinsbipvalue , result)
		}
		response.Total[n] = UStake{
				Coin: coinToSell,
				Value: valueToSell.String(),
				BipValue: result.String(),
			}
		
			n = n + 1
 
	} 



	response.TransactionCount = cState.Accounts.GetNonce(address)
	response.Bipvalue = coinsbipvalue

	return &response, nil
}
