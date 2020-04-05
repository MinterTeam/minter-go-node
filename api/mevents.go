package api

import (
	eventsdb "github.com/MinterTeam/events-db"
//	"github.com/MinterTeam/minter-go-sdk/wallet"
	"fmt"
	"strings"


)

type mEventsResponse struct {
	Events eventsdb.Events `json:"events"`
}




func mEvents(height uint64, find_ui []string) (*mEventsResponse, error) {
	var kkk mEventsResponse
	
	lll:=blockchain.GetEventsDB().LoadEvents(uint32(height))

	ttt:= find_ui
		
		for i,_:= range ttt {
		if strings.Contains(ttt[i], "Mx"){
				bbb, _ := MXconvert(ttt[i])
				ttt[i] = bbb.Wallet
		
			}
		}

	for _, sss:= range lll {
		str := fmt.Sprintf("%v", sss)
		for i,_:= range ttt {
			if strings.Contains(str, ttt[i]) {
				kkk.Events = append(kkk.Events , sss)
	
			}
		}
 
	}



	return &kkk, nil
}
