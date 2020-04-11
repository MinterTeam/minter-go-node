package api

import (
	"math/big"
	"fmt"
	
)

type AllUserStakes struct {
	PubKey        string	`json:"pub_key"`
	Stakes        []UStake 	`json:"stakes,omitempty"`
}

func Userstakes(height int, address string) ([]AllUserStakes , error) {
	var	tmresult AllUserStakes
	var 	result  []AllUserStakes

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

	candidates := cState.Candidates.GetCandidates()
 
	for _, candidate := range candidates {
		if len(ResponseUserstake(cState, *candidate, address)) > 0 {
			tmresult.PubKey	= candidate.PubKey.String()
			tmresult.Stakes	= ResponseUserstake(cState, *candidate, address)
			result = append(result, tmresult)
		}
	} 
	return result, nil
}
 

func GroupUserStakes(height int, address string) ([]UStake) {
	
	cState, _ := GetStateForHeight(height)

	if height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		cState.Candidates.LoadStakes()
		cState.Unlock()
	} 

	cState.RLock()
	defer cState.RUnlock()

	candidates := cState.Candidates.GetCandidates()
 	var result []UStake
	t1:=big.NewInt(0)
	t2:=big.NewInt(0)
	t3:=big.NewInt(0)
	t4:=big.NewInt(0)

	hhh := make([]UserstakeResponse, 1)
	result := []UStake{
}
		
					t1:=big.NewInt(0)
					t2:=big.NewInt(0)
					t3:=big.NewInt(0)
					t4:=big.NewInt(0)
for _, candidate := range candidates {
		tmstakes :=ResponseUserstake(cState, *candidate, address)
		if len(tmstakes) > 0 {
			for _,tmstake:= range tmstakes{
				fmt.Sscan(tmstake.Value, t1)
				fmt.Sscan(tmstake.BipValue, t2)
				r:=0
				for _,res:= range result{
					if res.Coin == tmstake.Coin {
						fmt.Sscan(res.Value, t3)
						fmt.Sscan(res.BipValue, t4)
						t1.Add(t1,t3)
						t2.Add(t2,t4)
						res.Value = t1.String()
						res.BipValue = t2.String()						
						r = 1	
						break
					}
				}
				if r == 0 {
					result = append(result , tmstake)
				}

			}
		}
	}

	return result
}
