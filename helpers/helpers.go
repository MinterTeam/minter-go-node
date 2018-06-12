package helpers

import "math/big"

func BipToPip(bip *big.Int) *big.Int {
	p := big.NewInt(10)
	p.Exp(p, big.NewInt(18), nil)
	p.Mul(p, bip)

	return p
}
