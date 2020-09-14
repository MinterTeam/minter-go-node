package helpers

import "testing"

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
