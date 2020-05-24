package api

import (
	eventsdb "github.com/MinterTeam/events-db"
	"fmt"
	"strings"
	"strconv"
)

type FindEventsResponse struct {
	Events eventsdb.Events `json:"events"`
}

func FindEvents(height uint64, find []string) (*FindEventsResponse , error) {
	var result FindEventsResponse 
	
	for i,_:= range find{
		if strings.Contains(find[i], "Mx"){
			find[i] = AddressToBytesConvert(find[i])
		}
	}

	events:=blockchain.GetEventsDB().LoadEvents(uint32(height))
	
	for _, event:= range events{
		str := fmt.Sprintf("%v", event)
		for i,_:= range find{
			if strings.Contains(str, find[i]) {
				result.Events = append(result.Events, event)
			}
		}
	}

	return &result, nil
}


func AddressToBytesConvert(address string) (string) {
 s := make([]string, 16)
 	s[0] = "0"
	s[1] = "1"
	s[2] = "2"
	s[3] = "3"
	s[4] = "4"
	s[5] = "5"
	s[6] = "6"
	s[7] = "7"
	s[8] = "8"
	s[9] = "9"
	s[10] = "a"
	s[11] = "b"
	s[12] = "c"
	s[13] = "d"
	s[14] = "e"
	s[15] = "f"

	var result string
	var t1, t2 int

	if strings.Contains(address, "Mx") {
		address=address[2:]
		if len(address)==40 {
			for u := 0; u < 20; u++ {
				tmp := address[:2]
				for i, _:= range s {
					if tmp[:1] == s[i] {
						t1 = i
					}
					if tmp[1:] == s[i] {
						t2 = i
					}
				}
				result = result +" "+ strconv.Itoa(t1*16 +  t2)
				address = address[2:]
			}
		} else {
			result = " incorrect address"
}
	} else {
			result = " incorrect address"
}



	return result[1:]
}
