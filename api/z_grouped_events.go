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
  
func GroupedEvents(height int64) (*[]EventsResponseValidator, error) {

	var recipients [4]string
	recipients[0] = "Validator"
	recipients[1] = "Delegator"
	recipients[2] = "DAO"
	recipients[3] = "Developers"

	result := []EventsResponseValidator {
	}

	if height == 0 {
		height = int64(blockchain.Height())
	}

	tmVals, err := client.Validators(&height, 1, 256)
	if err != nil {
		return nil, err
	}

	responseValidators := make([]EventsResponseValidator , len(tmVals.Validators))
	for i, val := range tmVals.Validators {
		var pk types.Pubkey
		copy(pk[:], val.PubKey.Bytes()[5:])
		responseValidators[i].Pubkey = pk.String()
	}

	events:=blockchain.GetEventsDB().LoadEvents(uint32(height))

	for _, event:= range events {
		finded:=false
		str := fmt.Sprintf("%v", event)
		for _,recipient:= range recipients{
			if strings.Contains(str, recipient){
				finded=true
			}	
		}
		t1:=big.NewInt(0)
		t2:=big.NewInt(0)
		if finded {
			validatorpubkey:= strings.Split(strings.Split(str, "] ")[1]," ")[1][:66]
			rolevalue:= strings.Split(strings.Split(str, "] ")[1]," ")[0]
			for i,_:= range responseValidators{
				if validatorpubkey == responseValidators[i].Pubkey{
					if strings.Contains(str, recipients[0]){
						responseValidators[i].ValidatorRole = rolevalue
					}	 
					if strings.Contains(str, recipients[1]){
						if len(responseValidators[i].DelegatorRole) == 0 {
							responseValidators[i].DelegatorRole = rolevalue
						}else{
							fmt.Sscan(responseValidators[i].DelegatorRole, t1)
							fmt.Sscan(rolevalue, t2)
							t1.Add(t1,t2)
							responseValidators[i].DelegatorRole = t1.String()
						}
					}
					if strings.Contains(str, recipients[2]){
						responseValidators[i].DaoRole = rolevalue
					}
					if strings.Contains(str, recipients[3]){
						responseValidators[i].DeveloperRole = rolevalue
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
