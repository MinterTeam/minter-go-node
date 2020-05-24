package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
	"strings"
)



type CandidateResponseAlt struct {
	PubKey        	string  `json:"pub_key"`
	TotalStake    	string  `json:"total_stake"`
	Commission    	uint    `json:"commission"`
	UsedSlots 	int 	`json:"used_slots"`
	UniqUsers 	int 	`json:"uniq_users"`
	MinStake	string	`json:"minstake"` 
	Status        	byte    `json:"status"`
}
const (
//	CandidateOff		= 0x01
//      CandidateOn		= 0x02	
	ValidatorOn		= 0x03
	ValidatorsMaxSlots 	= 1000
)

func ResponseCandidateAlt(state *state.State, c candidates.Candidate) CandidateResponseAlt {
	var candidate CandidateResponseAlt
	var addresses string
 	minstake:=big.NewInt(0)

	candidate.TotalStake = state.Candidates.GetTotalStake(c.PubKey).String()
	candidate.PubKey = c.PubKey.String()
	candidate.Commission = c.Commission
	candidate.Status = c.Status

	for _,validator := range state.Validators.GetValidators(){
		if validator.PubKey.String() == candidate.PubKey {
			candidate.Status = ValidatorOn
			break
		}  
	}
		 
	stakes := state.Candidates.GetStakes(c.PubKey)
	candidate.UsedSlots = len(stakes)

	for i, stake := range stakes {
		if candidate.UsedSlots >= ValidatorsMaxSlots {
			if i == 0 {
				minstake= stake.BipValue
			}else{
				for minstake.Cmp(stake.BipValue) > 0 {
					minstake=stake.BipValue
				}
			}
		}

		if !strings.Contains(addresses, stake.Owner.String()){
			addresses = addresses + " " + stake.Owner.String()
		}
	}

	candidate.UniqUsers = strings.Count(addresses, " ")
	candidate.MinStake = minstake.String()

	return candidate
}

func CandidateAlt(pubkey types.Pubkey, height int) (*CandidateResponseAlt, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	} 

	if height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		cState.Candidates.LoadStakes()
	 	cState.Validators.LoadValidators()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidate := cState.Candidates.GetCandidate(pubkey)
	if candidate == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Candidate not found"}
	}

	response := ResponseCandidateAlt(cState, *candidate)
	return &response, nil
}

func CandidatesAlt(height int, status int) (*[]CandidateResponseAlt, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	if height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		cState.Validators.LoadValidators()
		cState.Candidates.LoadStakes()
		cState.Unlock()
	} 

	cState.RLock()
	defer cState.RUnlock()

	candidates := cState.Candidates.GetCandidates()
	var result []CandidateResponseAlt
	for _, candidate := range candidates {
		candadateinfo:=ResponseCandidateAlt(cState, *candidate)
		if status > 0 {
			if candadateinfo.Status == byte(status) {
				result=append(result,candadateinfo)
			}
		}else{
			result=append(result,candadateinfo)
		}
 	} 
	return &result, nil
}
