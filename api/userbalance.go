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
	
	balances := cState.Accounts.GetBalances(address)
	
	var coinsbipvalue *big.Int
	var response UserbalanceResponse
 
		cState.RLock()
		defer cState.RUnlock()
		
	for k, valueToSell := range balances {
		coinToSell := k.String()
		result := CustomCoinBipBalance(coinToSell, valueToSell, height)

		if coinsbipvalue == nil {
			coinsbipvalue = result 
		} else {
			coinsbipvalue.Add(coinsbipvalue , result)
		}
	} 
 
	balances = cState.Accounts.GetBalances(address)
	response.Freecoins = make([]UStake, len(balances))

		fff := 0 
		for i, k := range balances{
			response.Freecoins[fff] = UStake{
				Coin: 	i.String(),
				Value: k.String(),
				BipValue: CustomCoinBipBalance(i.String(), k, height).String(),
			}
			fff= fff+1
		}


response.Delegated = GroupUserStakes(height, address.String())

qqq:=response.Freecoins
ddd:=response.Delegated

	t1:=big.NewInt(0)
	t2:=big.NewInt(0)

	for k:= range ddd {
		rrr:=0
	
//		t1:=big.NewInt(ddd[k].Value)
		fmt.Sscan(ddd[k].Value, t2)
		for m:= range qqq {
			if ddd[k].Coin == qqq[m].Coin{
				fmt.Sscan(qqq[m].Value, t2)
//				t1:=big.NewInt(qqq[m].Value)
				t1.Add(t1,t2)
				qqq[m].Value = t1.String()
				rrr = 1
			}
		}
		if rrr == 0 {
			qqq = append(qqq , ddd[k])
		}
	}


coinsbipvalue = big.NewInt(0)
result1:= new(big.Int)
	for i:= range qqq {
		fmt.Sscan(qqq[i].Value, t1)
		if len(qqq[i].Coin) != 0 {
			result1 = CustomCoinBipBalance(qqq[i].Coin, t1, height)
			if coinsbipvalue == nil {
				coinsbipvalue = result1 
			} else {
				coinsbipvalue.Add(coinsbipvalue , result1)
			}
		}
	}


	response.TransactionCount = cState.Accounts.GetNonce(address)
	response.Bipvalue = coinsbipvalue

	return &response, nil
}
