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


func FrozzedFunds(address types.Address, coin string) (*[]FrozInfo, error) {
	cState:= blockchain.CurrentState()
	cState.RLock()
	defer cState.RUnlock()  

	appState := new(types.AppState)
	var frozes []FrozInfo
	var froz FrozInfo
	cState.FrozenFunds.Export(appState , blockchain.Height())
	var tmpaddress types.Address
	var tmpcoin string

	
	if coin==tmpcoin && address==tmpaddress{
			
		for _,fund:=range appState.FrozenFunds{
			froz.Height 	  = int(fund.Height)
			froz.Address	  = fund.Address
			froz.CandidateKey = fund.CandidateKey
			froz.Coin 	  = fund.Coin
			froz.Value 	  = fund.Value
			frozes = append(frozes,froz)
		}
			 
	}else if coin!=tmpcoin && address!=tmpaddress{
		for _,fund:=range appState.FrozenFunds{
			if fund.Coin.String() == coin && fund.Address == address {
				froz.Height 	  = int(fund.Height)
				froz.Address	  = fund.Address
				froz.CandidateKey = fund.CandidateKey
				froz.Coin 	  = fund.Coin
				froz.Value 	  = fund.Value
				frozes = append(frozes,froz)
			}

		}
	}else{	
	  for _,fund:=range appState.FrozenFunds{
			if fund.Coin.String() == coin || fund.Address == address {
				froz.Height 	  = int(fund.Height)
				froz.Address	  = fund.Address
				froz.CandidateKey = fund.CandidateKey
				froz.Coin 	  = fund.Coin
				froz.Value 	  = fund.Value
				frozes = append(frozes,froz)
			}
		}
	}
	return &frozes, nil
}
