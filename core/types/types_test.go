// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"encoding/json"
	"github.com/MinterTeam/go-amino"
	"math/big"
	"strings"
	"testing"
)

func TestBytesConversion(t *testing.T) {
	bytes := []byte{5}
	hash := BytesToHash(bytes)

	var exp Hash
	exp[31] = 5

	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestIsHexAddress(t *testing.T) {
	tests := []struct {
		str string
		exp bool
	}{
		{"Mx5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"Mx5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"MxAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"MxAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"Mx5aaeb6053f3e94c9b9a09f33669435e7ef1beaed1", false},
		{"Mx5aaeb6053f3e94c9b9a09f33669435e7ef1beae", false},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed11", false},
		{"Mxxaaeb6053f3e94c9b9a09f33669435e7ef1beaed", false},
	}

	for _, test := range tests {
		if result := IsHexAddress(test.str); result != test.exp {
			t.Errorf("IsHexAddress(%s) == %v; expected %v",
				test.str, result, test.exp)
		}
	}
}

func TestHashJsonValidation(t *testing.T) {
	var tests = []struct {
		Prefix string
		Size   int
		Error  string
	}{
		{"", 62, "json: cannot unmarshal hex string without Mx prefix into Go value of type types.Hash"},
		{"Mx", 66, "hex string has length 66, want 64 for types.Hash"},
		{"Mx", 63, "json: cannot unmarshal hex string of odd length into Go value of type types.Hash"},
		{"Mx", 0, "hex string has length 0, want 64 for types.Hash"},
		{"Mx", 64, ""},
		{"Mx", 64, ""},
	}
	for _, test := range tests {
		input := `"` + test.Prefix + strings.Repeat("0", test.Size) + `"`
		var v Hash
		err := json.Unmarshal([]byte(input), &v)
		if err == nil {
			if test.Error != "" {
				t.Errorf("%s: error mismatch: have nil, want %q", input, test.Error)
			}
		} else {
			if err.Error() != test.Error {
				t.Errorf("%s: error mismatch: have %q, want %q", input, err, test.Error)
			}
		}
	}
}

func TestAddressUnmarshalJSON(t *testing.T) {
	var tests = []struct {
		Input     string
		ShouldErr bool
		Output    *big.Int
	}{
		{"", true, nil},
		{`""`, true, nil},
		{`"Mx"`, true, nil},
		{`"Mx00"`, true, nil},
		{`"MxG000000000000000000000000000000000000000"`, true, nil},
		{`"Mx0000000000000000000000000000000000000000"`, false, big.NewInt(0)},
		{`"Mx0000000000000000000000000000000000000010"`, false, big.NewInt(16)},
	}
	for i, test := range tests {
		var v Address
		err := json.Unmarshal([]byte(test.Input), &v)
		if err != nil && !test.ShouldErr {
			t.Errorf("test #%d: unexpected error: %v", i, err)
		}
		if err == nil {
			if test.ShouldErr {
				t.Errorf("test #%d: expected error, got none", i)
			}
			if v.Big().Cmp(test.Output) != 0 {
				t.Errorf("test #%d: address mismatch: have %v, want %v", i, v.Big(), test.Output)
			}
		}
	}
}

func BenchmarkAddressHex(b *testing.B) {
	testAddr := HexToAddress("Mx5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	for n := 0; n < b.N; n++ {
		testAddr.Hex()
	}
}

func TestAppState(t *testing.T) {
	testAddr := HexToAddress("Mx5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	pubkey := Pubkey{1, 2, 3}
	ba := NewBitArray(24)
	ba.SetIndex(3, true)

	appState := AppState{
		Validators: []Validator{
			{
				RewardAddress: testAddr,
				TotalBipStake: big.NewInt(1),
				PubKey:        pubkey,
				Commission:    1,
				AccumReward:   big.NewInt(1),
				AbsentTimes:   ba,
			},
		},
		Candidates: []Candidate{
			{
				RewardAddress: testAddr,
				OwnerAddress:  testAddr,
				TotalBipStake: big.NewInt(1),
				PubKey:        pubkey,
				Commission:    1,
				Stakes: []Stake{
					{
						Owner:    testAddr,
						Coin:     GetBaseCoin(),
						Value:    big.NewInt(1),
						BipValue: big.NewInt(1),
					},
				},
				CreatedAtBlock: 1,
				Status:         1,
			},
		},
		Accounts: []Account{
			{
				Address: testAddr,
				Balance: []Balance{
					{
						Coin:  GetBaseCoin(),
						Value: big.NewInt(1),
					},
				},
				Nonce: 1,
				MultisigData: &Multisig{
					Weights:   []uint{1, 2, 3},
					Threshold: 1,
					Addresses: []Address{testAddr, testAddr},
				},
			},
		},
		Coins: []Coin{
			{
				Name:           "ASD",
				Symbol:         GetBaseCoin(),
				Volume:         big.NewInt(1),
				Crr:            1,
				ReserveBalance: big.NewInt(1),
			},
		},
		FrozenFunds: []FrozenFund{
			{
				Height:       1,
				Address:      testAddr,
				CandidateKey: pubkey,
				Coin:         GetBaseCoin(),
				Value:        big.NewInt(1),
			},
		},
		UsedChecks: []UsedCheck{
			"123",
		},
		MaxGas: 10,
	}

	cdc := amino.NewCodec()

	b1, err := cdc.MarshalJSON(appState)
	if err != nil {
		panic(err)
	}

	newAppState := AppState{}
	err = cdc.UnmarshalJSON(b1, &newAppState)
	if err != nil {
		panic(err)
	}

	b2, err := cdc.MarshalJSON(newAppState)
	if err != nil {
		panic(err)
	}

	if bytes.Compare(b1, b2) != 0 {
		t.Errorf("Bytes are not the same")
	}
}
