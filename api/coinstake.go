package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
)

type CStake struct {
	Coin     string `json:"coin"`
	PubKey   string  `json:"pub_key"`
	Address  string `json:"address"`
	Value    string `json:"value"`
	BipValue string `json:"bip_value"`
}

type CoinstakeResponse []CStake

func makeResponseCoinstake(state *state.State, c candidates.Candidate, symbol string) CoinstakeResponse {



		tmp := "1"
	
		stakes := state.Candidates.GetStakes(c.PubKey)

bbb := symbol

for i, stake := range stakes {
if stake.Coin.String()==bbb{
tmp = tmp + "1" 
i=i
}
}

		coinstake := make([]CStake, len(tmp)-1)
		kkk := 0

		for i, stake := range stakes {
			if stake.Coin.String() == bbb{
				coinstake[kkk] = CStake{
					Coin:     stake.Coin.String(),
					PubKey:	  c.PubKey.String(),
					Address:    stake.Owner.String(),
					Value:    stake.Value.String(),
					BipValue: stake.BipValue.String(),
				}
				i=i
				kkk = kkk + 1
			}
		}

	return coinstake
}

func Coinstake(pubkey types.Pubkey, height int, symbol string) (*CoinstakeResponse, error) {
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

	candidate := cState.Candidates.GetCandidate(pubkey)
	if candidate == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin stakes not found"}
	}

	response := makeResponseCoinstake(cState, *candidate, symbol)
	return &response, nil
}
