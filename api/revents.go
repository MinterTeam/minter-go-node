package api

import (
	"fmt"
	"strings"
	"math/big"
	"github.com/MinterTeam/minter-go-node/core/types"
)

type EventsResponseValidator struct {
	Pubkey string `json:"pub_key"`
	DaoRole string `json:"dao"`
	DeveloperRole string `json:"developers"`
	ValidatorRole string `json:"validator"`
	DelegatorRole string `json:"delegator"`
}

type EventResponseValidators []EventsResponseValidator 


func rEvents(height uint64) (*EventResponseValidators, error) {

var fff [4]string
fff[0] = "Validator"
fff[1] = "Delegator"
fff[2] = "DAO"
fff[3] = "Developers"

result := EventResponseValidators {
}

if height == 0 {
		height = blockchain.Height()
	}

	h := int64(height)
	tmVals, err := client.Validators(&h, 1, 256)
	if err != nil {
		return nil, err
	}
	responseValidators := make(EventResponseValidators , len(tmVals.Validators))
	for i, val := range tmVals.Validators {
		var pk types.Pubkey
		copy(pk[:], val.PubKey.Bytes()[5:])
		responseValidators[i].Pubkey = pk.String()
	}

lll:=blockchain.GetEventsDB().LoadEvents(uint32(height))

	for _, sss:= range lll {
		kkk:=0
		str := fmt.Sprintf("%v", sss)
		for _,eee:= range fff{
			if strings.Contains(str, eee){
				kkk=1
			}	
		}
		t1:=big.NewInt(0)
		t2:=big.NewInt(0)
		if kkk == 1 {
			ppp:= strings.Split(strings.Split(str, "] ")[1]," ")[1][:66]
			vvv:= strings.Split(strings.Split(str, "] ")[1]," ")[0]
			for i,_:= range responseValidators{
				if ppp == responseValidators[i].Pubkey{
					if strings.Contains(str, fff[0]){
						responseValidators[i].ValidatorRole = vvv
					}	
					if strings.Contains(str, fff[1]){
						if len(responseValidators[i].DelegatorRole) == 0 {
							responseValidators[i].DelegatorRole = vvv
						}else{
							fmt.Sscan(responseValidators[i].DelegatorRole, t1)
							fmt.Sscan(vvv, t2)
							t1.Add(t1,t2)
							responseValidators[i].DelegatorRole = t1.String()
						}
					}
					if strings.Contains(str, fff[2]){
						responseValidators[i].DaoRole = vvv
					}
					if strings.Contains(str, fff[3]){
						responseValidators[i].DeveloperRole = vvv
					}
					
				}
			}
		} 
	}

	for i,_:=range responseValidators {
		if len(responseValidators[i].DaoRole) > 0 {
			result=append(result,responseValidators[i])
		}
	}

	return &result, nil
}
