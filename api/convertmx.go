package api

import (
//	"fmt"
	"strings"
	"strconv"
)


type WalletDecode struct {
	Wallet string `json:"wallet"`
}

func MXconvert(wlt string) (*WalletDecode, error) {

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

	var k, t string
	var t1, t2 int


	if strings.Contains(wlt, "Mx") {
		wlt=wlt[2:]
		if len(wlt)==40 {
			u:=0
			for u < 20{
				t = wlt[:2]
				for i, _:= range s {
					if t[:1] == s[i] {
						t1 = i
					}
					if t[1:] == s[i] {
						t2 = i
					}
				}

				if u == 0 {
					k = strconv.Itoa(t1*16 +  t2)
				}else{
					k = k +" "+ strconv.Itoa(t1*16 +  t2)
				}
				wlt= wlt[2:]
				u=u+1
			}
		} else {
k = "incorrect wallet"
}
	} else {
k = "incorrect wallet"
}



	return &WalletDecode {
		Wallet: k ,
	}, nil
}
