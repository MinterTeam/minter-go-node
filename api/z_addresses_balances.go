package api

import (
 	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"math/big"
	"fmt"
)

type CoinBalance struct {
	Coin     string `json:"coin"`
	Value    string `json:"value"`
	BipValue string `json:"bip_value"`
}

type AddressBalanceResponse struct {
	Freecoins         	[]CoinBalance 	`json:"freecoins"`
	Delegated         	[]CoinBalance 	`json:"delegated"`
	Total         		[]CoinBalance 	`json:"total"`
	TransactionCount 	uint64         	`json:"transaction_count"`
	Bipvalue     		*big.Int       	`json:"bipvalue"`
} 

type AddressesBalancesResponse struct {
	Address 	types.Address 		`json:"address"`
	Balance		AddressBalanceResponse	`json:"balance"`

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


func MakeAddressBalance(height int, address types.Address) (*AddressBalanceResponse,
	error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}
	
	if height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		cState.Candidates.LoadStakes()
		cState.Unlock()
	} 

		cState.RLock()
		defer cState.RUnlock()

	balances := cState.Accounts.GetBalances(address)
	var response AddressBalanceResponse
	coinsbipvalue := big.NewInt(0)

	for coin, value := range balances {
		result := CustomCoinBipBalance(coin.String(), value, height)
		freecoin:=CoinBalance{
			Coin: 		coin.String(),
			Value: 		value.String(),
			BipValue: 	result.String(),
			}
		response.Freecoins = append(response.Freecoins,freecoin) 
	} 
 	
	
	candidates := cState.Candidates.GetCandidates()
	t1:=big.NewInt(0)
	t2:=big.NewInt(0)
	t3:=big.NewInt(0)
	t4:=big.NewInt(0)  
	for _, candidate := range candidates {
		tmstakes :=ResponseUserstake(cState, *candidate, address.String())
		if len(tmstakes) > 0 {
			for _,tmstake:= range tmstakes{
				fmt.Sscan(tmstake.Value, t1) 
				fmt.Sscan(tmstake.BipValue, t2)
				coinfound:=false
				for _,res:= range response.Delegated{
					if res.Coin == tmstake.Coin {
						fmt.Sscan(res.Value, t3)
						fmt.Sscan(res.BipValue, t4)
						t1.Add(t1,t3)
						t2.Add(t2,t4)
						res.Value = t1.String()
						res.BipValue = t2.String()						
						coinfound = true	
						break
					}
				}
				if !coinfound {
					response.Delegated = append(response.Delegated, tmstake)
				}

			}
		}
	}

	for _, coin := range response.Freecoins{
		response.Total = append(response.Total,coin)
	}


	t1=big.NewInt(0)
	t2=big.NewInt(0)

	for _,coin:= range response.Delegated {
		coinfound:=false 
		for i,_:= range response.Total{
			if response.Total[i].Coin == coin.Coin {
				fmt.Sscan(response.Total[i].Value, t1)
				fmt.Sscan(coin.Value, t2)
				t1.Add(t1,t2)
				response.Total[i].Value = t1.String()
				coinfound = true
				break
			}
		}
		if !coinfound {
			response.Total= append(response.Total,coin)
		}
	}


	coinsbipvalue = big.NewInt(0)

	for i, coin := range response.Total {
		value:=big.NewInt(0)
		fmt.Sscan(coin.Value, value)
		result := CustomCoinBipBalance(coin.Coin, value, height)

		if coinsbipvalue == nil {
			coinsbipvalue = result 
		} else {
			coinsbipvalue.Add(coinsbipvalue , result)
		}
		response.Total[i] = CoinBalance{
				Coin: coin.Coin,
				Value: value.String(),
				BipValue: result.String(),
			}
	} 

	response.TransactionCount = cState.Accounts.GetNonce(address)
	response.Bipvalue = coinsbipvalue

	return &response, nil
}


func ResponseUserstake(state *state.State, c candidates.Candidate, address string) []CoinBalance {
	var tmstakes CoinBalance 
	var userstake []CoinBalance 

	stakes := state.Candidates.GetStakes(c.PubKey)

	for _,stake := range stakes {
		if stake.Owner.String()==address{
			tmstakes.Coin    = stake.Coin.String()
			tmstakes.Value   = stake.Value.String() 
			tmstakes.BipValue= stake.BipValue.String()
			userstake=append(userstake,tmstakes)
		}
	}
	
	return userstake
}
