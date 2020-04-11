package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	
)

type UStake struct {
	Coin     string `json:"coin"`
	Value    string `json:"value"`
	BipValue string `json:"bip_value"`
}

func ResponseUserstake(state *state.State, c candidates.Candidate, address string) []UStake{
var tmstakes UStake
	var userstake []UStake

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

func Userstake(pubkey types.Pubkey, height int, address string) ([]UStake, error) {
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
		return nil, rpctypes.RPCError{Code: 404, Message: "Stakes of address not found"}
	}

	response := ResponseUserstake(cState, *candidate, address)
	return response, nil
}
