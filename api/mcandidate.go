package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
//	"fmt"
	"strings"
)



type mCandidateResponse struct {
	PubKey        	string  `json:"pub_key"`
	TotalStake    	string  `json:"total_stake"`
	Commission    	uint    `json:"commission"`
	UsedSlots 	int 	`json:"used_slots"`
	UniqUsers 	int 	`json:"uniq_users"`
	MinStake	string	`json:"minstake"`
	Status        	byte    `json:"status"`
}

func mmakeResponseCandidate(state *state.State, c candidates.Candidate) mCandidateResponse {
var candidate mCandidateResponse 

		candidate.TotalStake = state.Candidates.GetTotalStake(c.PubKey).String()
		candidate.PubKey = c.PubKey.String()
		candidate.Commission = c.Commission
		candidate.Status = c.Status
 		mnum:=big.NewInt(0)
		var hhh string
		kkk:=0
		nnn:=0
		stakes := state.Candidates.GetStakes(c.PubKey)
		candidate.UsedSlots = len(stakes)

		for _, stake := range stakes {
			if kkk ==0 {
				hhh = stake.Owner.String()
				mnum = stake.BipValue
				kkk=1
				nnn=nnn+1
			} else {

				for mnum.Cmp(stake.BipValue) > 0 {
					mnum=stake.BipValue
				}
			
				if strings.Contains(hhh, stake.Owner.String()){
				}else{
					hhh = hhh + " " + stake.Owner.String()
				}

			}	
		}

		candidate.UniqUsers = strings.Count(hhh , " ")+1
		if candidate.UsedSlots > 999 {
			candidate.MinStake = mnum.String()
		} else {
			candidate.MinStake = "0"
		}

	return candidate
}

func mCandidate(pubkey types.Pubkey, height int) (*mCandidateResponse, error) {
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
		return nil, rpctypes.RPCError{Code: 404, Message: "Candidate not found"}
	}

	response := mmakeResponseCandidate(cState, *candidate)
	return &response, nil
}
