package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
)

type CStake struct {
	Address  string `json:"address"`
	PubKey   string `json:"pub_key"`
	Coin     string `json:"coin"`
	Value    string `json:"value"`
	BipValue string `json:"bip_value"`
}

func ResponseStakes(state *state.CheckState, c *candidates.Candidate, coin string, address types.Address) []*CStake {
	var coinStakes []*CStake

	var multi bool
	var allPubkeyStakes bool

	var emptyAddress types.Address

	if coin != "" && address != emptyAddress {
		multi = true
	}
	if coin == "" && address == emptyAddress {
		allPubkeyStakes = true
	}

	stakes := state.Candidates().GetStakes(c.PubKey)
	for _, stake := range stakes {
		if !((multi && stake.Coin.String() == coin && stake.Owner == address) || (!multi && (stake.Coin.String() == coin || stake.Owner == address || allPubkeyStakes))) {
			continue
		}
		coinStakes = append(coinStakes, &CStake{
			Address:  stake.Owner.String(),
			PubKey:   c.PubKey.String(),
			Coin:     stake.Coin.String(),
			Value:    stake.Value.String(),
			BipValue: stake.BipValue.String(),
		})
	}

	return coinStakes
}

func GetStakes(pubkey types.Pubkey, height int, coin string, address types.Address) ([]*CStake, error) {
	var coinStakes []*CStake

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

	var emptyPyb types.Pubkey

	var allCandidates []*candidates.Candidate
	if pubkey == emptyPyb {
		allCandidates = cState.Candidates().GetCandidates()
	} else {
		allCandidates = []*candidates.Candidate{cState.Candidates().GetCandidate(pubkey)}
	}

	for _, candidate := range allCandidates {
		tmresponse := ResponseStakes(cState, candidate, coin, address)
		for _, coinStake := range tmresponse {
			coinStakes = append(coinStakes, coinStake)
		}
	}

	return coinStakes, nil
}
