package helpers

import (
	"math/big"
	"testing"
)

func TestIsValidBigInt(t *testing.T) {
	cases := map[string]bool{
		"":   false,
		"1":  true,
		"1s": false,
		"-1": false,
		"123437456298465928764598276349587623948756928764958762934569": true,
	}

	for str, result := range cases {
		if IsValidBigInt(str) != result {
			t.Fail()
		}
	}
}

func TestStringToBigInt(t *testing.T) {
	cases := map[string]bool{
		"":   false,
		"1":  true,
		"1s": false,
		"-1": true,
		"123437456298465928764598276349587623948756928764958762934569": true,
	}

	for str, result := range cases {
		_, err := stringToBigInt(str)

		if err != nil && result || err == nil && !result {
			t.Fatalf("%s %s", err, str)
		}
	}

	result := StringToBigInt("10")
	if result.Cmp(big.NewInt(10)) != 0 {
		t.Fail()
	}
}

func TestBipToPip(t *testing.T) {
	pip := BipToPip(big.NewInt(1))

	if pip.Cmp(big.NewInt(1000000000000000000)) != 0 {
		t.Fail()
	}
}
