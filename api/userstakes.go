package api

import (
	"math/big"
	"fmt"
	
)


type AllUserStakes struct {
	PubKey        string	`json:"pub_key"`
	Stakes        []UStake 	`json:"stakes,omitempty"`
}






func Userstakes(height int, address string) (*[]AllUserStakes , error) {
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
 
	hhh := make([]UserstakeResponse, 1)

	tmp := 0
		for _, candidate := range candidates {
			if len(ResponseUserstake(cState, *candidate, address)) > len(hhh)  {
				tmp = tmp + 1 
			}
		}

	result := make([]AllUserStakes, tmp)
 	tmp = 0

	for i, candidate := range candidates {
		if len(ResponseUserstake(cState, *candidate, address)) > len(hhh) {
			result[tmp].PubKey	= candidate.PubKey.String()
			result[tmp].Stakes	= ResponseUserstake(cState, *candidate, address)
			i=i
			tmp = tmp + 1
		}
	}

	return &result, nil
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
 
	hhh := make([]UserstakeResponse, 1)
	result := []UStake{
}
		
					t1:=big.NewInt(0)
					t2:=big.NewInt(0)
					t3:=big.NewInt(0)
					t4:=big.NewInt(0)
ttt:=0
	for _, candidate := range candidates {
		kkk:=ResponseUserstake(cState, *candidate, address)
		if len(kkk) > len(hhh) {
			for l:= range kkk{
				if ttt==0 {
					result = append(result , kkk[l])
					ttt=ttt+1
				} else {
					rrr:=0

					fmt.Sscan(kkk[l].Value, t1)
					fmt.Sscan(kkk[l].BipValue, t2)

					for f:= range result{
						if result[f].Coin == kkk[l].Coin {
							fmt.Sscan(result[f].Value, t3)
							fmt.Sscan(result[f].BipValue, t4)
							t1.Add(t1,t3)
							t2.Add(t2,t4)
							result[f].Value = t1.String()
							result[f].BipValue = t2.String()						
							rrr = 1	
							break
						}

					}

					if rrr == 0 {
						result = append(result , kkk[l])
					}
 
				}

				
			}
		}
	}

	return result
}
