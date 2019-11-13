package helpers

import (
	"fmt"
	"math/big"
)

func BipToPip(bip *big.Int) *big.Int {
	p := big.NewInt(10)
	p.Exp(p, big.NewInt(18), nil)
	p.Mul(p, bip)

	return p
}

func StringToBigInt(s string) *big.Int {
	b, success := big.NewInt(0).SetString(s, 10)
	if !success {
		panic(fmt.Sprintf("Cannot decode %s into big.Int", s))
	}

	return b
}
