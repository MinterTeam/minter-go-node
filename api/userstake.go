package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	
)

type UStake struct {
//	Address  string `json:"address"`
//	PubKey   string  `json:"pub_key"`
	Coin     string `json:"coin"`
	Value    string `json:"value"`
	BipValue string `json:"bip_value"`
}



type UserstakeResponse []UStake

func ResponseUserstake(state *state.State, c candidates.Candidate, address string) UserstakeResponse {



	tmp := 0

		stakes := state.Candidates.GetStakes(c.PubKey)


for _, stake := range stakes {
if stake.Owner.String()==address{
tmp = tmp + 1 
}
}

		userstake := make([]UStake, tmp)
	nstk := 0
		for _,stake := range stakes {
			if stake.Owner.String()==address{
				userstake[nstk] = UStake{
//					Address:  stake.Owner.String(),
//					PubKey:	  c.PubKey.String(),
					Coin:     stake.Coin.String(),
					Value:    stake.Value.String(),
					BipValue: stake.BipValue.String(),
				}
			nstk=nstk+1
			
			}
		}
	

	return userstake
}

func Userstake(pubkey types.Pubkey, height int, address string) (*UserstakeResponse, error) {
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
		return nil, rpctypes.RPCError{Code: 404, Message: "Adress stakes not found"}
	}

	response := ResponseUserstake(cState, *candidate, address)
	return &response, nil
}