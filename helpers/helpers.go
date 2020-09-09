package helpers

import (
	"fmt"
	"math/big"
)

// BipToPip converts BIP to PIP (multiplies input by 1e18)
func BipToPip(bip *big.Int) *big.Int {
	p := big.NewInt(10)
	p.Exp(p, big.NewInt(18), nil)
	p.Mul(p, bip)

	return p
}

// StringToBigInt converts string to BigInt, panics on empty strings and errors
func StringToBigInt(s string) *big.Int {
	if s == "" {
		panic("string is empty")
	}

	b, success := big.NewInt(0).SetString(s, 10)
	if !success {
		panic(fmt.Sprintf("Cannot decode %s into big.Int", s))
	}

	return b
}

// IsValidBigInt verifies that string is a valid int
func IsValidBigInt(s string) bool {
	if s == "" {
		return false
	}

	b, success := big.NewInt(0).SetString(s, 10)
	if !success {
		return false
	}

	if b.Cmp(big.NewInt(0)) == -1 {
		return false
	}

	return true
}
