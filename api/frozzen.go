package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type FrozInfo struct {
	Height       int              `json:"height"`
	Address      types.Address    `json:"address"`
	CandidateKey *types.Pubkey    `json:"candidateKey"`
	Coin         types.CoinSymbol `json:"coin"`
	Value        string           `json:"value"`
}

func FrozzedFunds(address types.Address, coin string) ([]*FrozInfo, error) {
	cState := blockchain.CurrentState()
	cState.RLock()
	defer cState.RUnlock()

	var frozes []*FrozInfo

	appState := new(types.AppState)
	cState.FrozenFunds().Export(appState, blockchain.Height())

	var emptyAddress types.Address

	if coin == "" && address == emptyAddress {
		for _, fund := range appState.FrozenFunds {
			frozes = append(frozes, &FrozInfo{
				Height:       int(fund.Height),
				Address:      fund.Address,
				CandidateKey: fund.CandidateKey,
				Coin:         fund.Coin,
				Value:        fund.Value,
			})
		}
		return frozes, nil
	}

	if coin != "" && address != emptyAddress {
		for _, fund := range appState.FrozenFunds {
			if fund.Coin.String() != coin || fund.Address != address {
				continue
			}
			frozes = append(frozes, &FrozInfo{
				Height:       int(fund.Height),
				Address:      fund.Address,
				CandidateKey: fund.CandidateKey,
				Coin:         fund.Coin,
				Value:        fund.Value,
			})
		}
		return frozes, nil
	}

	for _, fund := range appState.FrozenFunds {
		if fund.Coin.String() != coin && fund.Address != address {
			continue
		}
		frozes = append(frozes, &FrozInfo{
			Height:       int(fund.Height),
			Address:      fund.Address,
			CandidateKey: fund.CandidateKey,
			Coin:         fund.Coin,
			Value:        fund.Value,
		})
	}
	return frozes, nil
}
