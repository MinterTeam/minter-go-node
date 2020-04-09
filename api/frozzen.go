package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type FrozInfo struct {
	Height 		int 			`json:"height"`
	Address  	types.Address 		`json:"address"`
	CandidateKey 	*types.Pubkey 		`json:"candidateKey"`
	Coin 		types.CoinSymbol 	`json:"coin"`
	Value 		string 			`json:"value"`
}


func Frozzen(address types.Address) (*[]FrozInfo, error) {
	cState:= blockchain.CurrentState()
	cState.RLock()
	defer cState.RUnlock()  

	appState := new(types.AppState)
	var froz []FrozInfo
	cState.FrozenFunds.Export(appState , blockchain.Height())
	var tmp types.Address
	
	if address.String()==tmp.String(){
		froz =make([]FrozInfo, len(appState.FrozenFunds))
		for i,fund:=range appState.FrozenFunds{
			froz[i] = FrozInfo{
				Height: 	int(fund.Height),
				Address: 	fund.Address,
				CandidateKey:	fund.CandidateKey,
				Coin:		fund.Coin,
				Value:		fund.Value,
			}
		}
				
	}else{
		n:=0
		tmp:=FrozInfo{}
		for _,fund:=range appState.FrozenFunds{
			if fund.Address == address {
				froz = append(froz,tmp)
				froz[n] = FrozInfo{
					Height: 	int(fund.Height),
					Address: 	fund.Address,
					CandidateKey:	fund.CandidateKey,
					Coin:		fund.Coin,
					Value:		fund.Value,
				}
				n=n+1
			}

		}
	}
	return &froz, nil
}
