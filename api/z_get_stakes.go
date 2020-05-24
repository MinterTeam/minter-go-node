package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
)

type CStake struct {
	Address  string  `json:"address"`
	PubKey   string  `json:"pub_key"`
	Coin     string  `json:"coin"`
	Value    string  `json:"value"`
	BipValue string  `json:"bip_value"`
}

func ResponseStakes(state *state.State, c candidates.Candidate, coin string, address types.Address) []CStake{
	var coinstake CStake
	var coinstakes []CStake
	var tmsymbol string
	var tmaddress types.Address

	multiresponse := false
	allpubkeystakes := false

	if  coin != tmsymbol{
		if address != tmaddress{
			multiresponse=true
		}
	}else if address == tmaddress{
			allpubkeystakes = true
	}
	stakes := state.Candidates.GetStakes(c.PubKey)
	for _, stake := range stakes {
		loadstake := false 
		if multiresponse == true {
			if stake.Coin.String() == coin && stake.Owner == address{
				loadstake = true				
			}
		}else{
			if stake.Coin.String() == coin{
				loadstake = true			
			}
			if stake.Owner == address{
				loadstake = true	
			}
			if allpubkeystakes{
				loadstake = true			
			}
		}  
		if loadstake == true {
			coinstake.Address = stake.Owner.String()
			coinstake.PubKey = c.PubKey.String()
			coinstake.Coin = stake.Coin.String()
			coinstake.Value = stake.Value.String()
			coinstake.BipValue = stake.BipValue.String()
			coinstakes = append(coinstakes,coinstake)
		}
	}
	return coinstakes 
}


func GetStakes(pubkey types.Pubkey, height int, coin string, address types.Address) ([]CStake, error) {
	var coinstakes []CStake
	var tmpubkey types.Pubkey
	var candidates []*candidates.Candidate

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
	
	if pubkey == tmpubkey {
		candidates = cState.Candidates.GetCandidates()
	}else{
		candidates =append(candidates,cState.Candidates.GetCandidate(pubkey))
	}

	for _, candidate := range candidates {
		tmresponse:=ResponseStakes(cState, *candidate, coin ,address) 
		for _,coinstake:= range tmresponse {
			coinstakes = append(coinstakes,coinstake)
		}
	}

	return coinstakes, nil
}
