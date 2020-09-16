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
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/go-amino"
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
				TotalBipStake: big.NewInt(1).String(),
				PubKey:        pubkey,
				AccumReward:   big.NewInt(1).String(),
				AbsentTimes:   ba,
			},
		},
		Candidates: []Candidate{
			{
				RewardAddress: testAddr,
				OwnerAddress:  testAddr,
				TotalBipStake: big.NewInt(1).String(),
				PubKey:        pubkey,
				Commission:    1,
				Stakes: []Stake{
					{
						Owner:    testAddr,
						Coin:     GetBaseCoinID(),
						Value:    big.NewInt(1).String(),
						BipValue: big.NewInt(1).String(),
					},
				},
				Status: 1,
			},
		},
		Accounts: []Account{
			{
				Address: testAddr,
				Balance: []Balance{
					{
						Coin:  GetBaseCoinID(),
						Value: big.NewInt(1).String(),
					},
					{
						Coin:  GetBaseCoinID() + 1,
						Value: big.NewInt(1).String(),
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
				ID:        GetBaseCoinID() + 1,
				Name:      "ASD",
				Symbol:    StrToCoinSymbol("TEST"),
				Volume:    big.NewInt(1).String(),
				Crr:       1,
				Reserve:   helpers.BipToPip(big.NewInt(100000)).String(),
				MaxSupply: helpers.BipToPip(big.NewInt(100000)).String(),
			},
		},
		FrozenFunds: []FrozenFund{
			{
				Height:       1,
				Address:      testAddr,
				CandidateKey: &pubkey,
				Coin:         GetBaseCoinID(),
				Value:        big.NewInt(1).String(),
			},
		},
		UsedChecks: []UsedCheck{
			"00004601d10c33eda76bb16a54a0d8882a57ec34e964aa23e2b5d9aa10957fee",
		},
		MaxGas:       10,
		TotalSlashed: big.NewInt(1e18).String(),
	}

	cdc := amino.NewCodec()

	if err := appState.Verify(); err != nil {
		t.Error(err)
	}

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

	if !bytes.Equal(b1, b2) {
		t.Errorf("Bytes are not the same")
	}
}

func TestAppStateToInvalidState(t *testing.T) {
	testAddr := HexToAddress("Mx5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	pubkey := Pubkey{1, 2, 3}
	ba := NewBitArray(24)
	ba.SetIndex(3, true)

	appState := AppState{}
	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState = AppState{
		TotalSlashed: big.NewInt(1e18).String(),
	}
	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState = AppState{
		TotalSlashed: big.NewInt(1e18).String(),
		Validators: []Validator{
			{
				TotalBipStake: big.NewInt(1).String(),
				PubKey:        pubkey,
				AccumReward:   big.NewInt(1).String(),
				AbsentTimes:   ba,
			},
			{
				TotalBipStake: big.NewInt(1).String(),
				PubKey:        pubkey,
				AccumReward:   big.NewInt(1).String(),
				AbsentTimes:   ba,
			},
		},
		Candidates: []Candidate{
			{
				RewardAddress: testAddr,
				OwnerAddress:  testAddr,
				TotalBipStake: big.NewInt(1).String(),
				PubKey:        pubkey,
				Commission:    1,
				Stakes: []Stake{
					{
						Owner:    testAddr,
						Coin:     GetBaseCoinID(),
						Value:    big.NewInt(1).String(),
						BipValue: big.NewInt(1).String(),
					},
				},
				Status: 1,
			},
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState = AppState{
		TotalSlashed: big.NewInt(1e18).String(),
		Validators: []Validator{
			{
				TotalBipStake: big.NewInt(1).String(),
				PubKey:        pubkey,
				AccumReward:   big.NewInt(1).String(),
				AbsentTimes:   ba,
			},
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState = AppState{
		TotalSlashed: big.NewInt(1e18).String(),
		Validators: []Validator{
			{
				TotalBipStake: "",
				PubKey:        pubkey,
				AccumReward:   big.NewInt(1).String(),
				AbsentTimes:   ba,
			},
		},
		Candidates: []Candidate{
			{
				RewardAddress: testAddr,
				OwnerAddress:  testAddr,
				TotalBipStake: big.NewInt(1).String(),
				PubKey:        pubkey,
				Commission:    1,
				Stakes: []Stake{
					{
						Owner:    testAddr,
						Coin:     GetBaseCoinID(),
						Value:    big.NewInt(1).String(),
						BipValue: big.NewInt(1).String(),
					},
				},
				Status: 1,
			},
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Validators[0].TotalBipStake = big.NewInt(1e18).String()
	appState.Validators[0].AccumReward = ""
	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Validators[0].AccumReward = big.NewInt(1e18).String()
	appState.Validators[0].AbsentTimes = nil
	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Accounts = []Account{
		{
			Address: testAddr,
			Balance: []Balance{
				{
					Coin:  GetBaseCoinID(),
					Value: big.NewInt(1).String(),
				},
			},
			Nonce: 1,
		}, {
			Address: testAddr,
			Balance: []Balance{
				{
					Coin:  GetBaseCoinID(),
					Value: big.NewInt(1).String(),
				},
			},
			Nonce: 1,
		},
	}

	appState.Validators[0].AbsentTimes = ba
	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Accounts = []Account{
		{
			Address: testAddr,
			Balance: []Balance{
				{
					Coin:  GetBaseCoinID(),
					Value: "",
				},
			},
			Nonce: 1,
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Accounts = []Account{
		{
			Address: testAddr,
			Balance: []Balance{
				{
					Coin:  GetBaseCoinID() + 1,
					Value: big.NewInt(1).String(),
				},
			},
			Nonce: 1,
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Accounts = []Account{
		{
			Address: testAddr,
			Balance: []Balance{
				{
					Coin:  GetBaseCoinID(),
					Value: big.NewInt(1).String(),
				},
			},
			Nonce: 1,
		},
	}

	appState.Candidates[0].Stakes = []Stake{
		{
			Owner:    testAddr,
			Coin:     GetBaseCoinID(),
			Value:    big.NewInt(1).String(),
			BipValue: big.NewInt(1).String(),
		},
		{
			Owner:    testAddr,
			Coin:     GetBaseCoinID(),
			Value:    big.NewInt(1).String(),
			BipValue: big.NewInt(1).String(),
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Candidates[0].Stakes = []Stake{
		{
			Owner:    testAddr,
			Coin:     GetBaseCoinID() + 1,
			Value:    big.NewInt(1).String(),
			BipValue: big.NewInt(1).String(),
		},
	}

	appState.Coins = []Coin{
		{
			ID: GetBaseCoinID() + 2,
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Candidates[0].Stakes = []Stake{
		{
			Owner:    testAddr,
			Coin:     GetBaseCoinID() + 1,
			Value:    big.NewInt(1).String(),
			BipValue: big.NewInt(1).String(),
		},
	}

	appState.Coins = []Coin{
		{
			ID:        GetBaseCoinID() + 1,
			Name:      "ASD",
			Symbol:    StrToCoinSymbol("TEST"),
			Volume:    big.NewInt(1).String(),
			Crr:       1,
			Reserve:   helpers.BipToPip(big.NewInt(100000)).String(),
			MaxSupply: helpers.BipToPip(big.NewInt(100000)).String(),
		},
		{
			ID:     GetBaseCoinID(),
			Symbol: GetBaseCoin(),
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Coins = []Coin{
		{
			ID:        GetBaseCoinID() + 1,
			Name:      "ASD",
			Symbol:    StrToCoinSymbol("TEST"),
			Volume:    big.NewInt(1).String(),
			Crr:       1,
			Reserve:   helpers.BipToPip(big.NewInt(100000)).String(),
			MaxSupply: helpers.BipToPip(big.NewInt(100000)).String(),
		},
		{
			ID:        GetBaseCoinID() + 1,
			Name:      "ASD",
			Symbol:    StrToCoinSymbol("TEST"),
			Volume:    big.NewInt(1).String(),
			Crr:       1,
			Reserve:   helpers.BipToPip(big.NewInt(100000)).String(),
			MaxSupply: helpers.BipToPip(big.NewInt(100000)).String(),
		},
		{
			ID:     GetBaseCoinID(),
			Symbol: GetBaseCoin(),
		},
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}

	appState.Coins = []Coin{
		{
			ID:        GetBaseCoinID() + 1,
			Name:      "ASD",
			Symbol:    StrToCoinSymbol("TEST"),
			Volume:    big.NewInt(1).String(),
			Crr:       1,
			Reserve:   helpers.BipToPip(big.NewInt(100000)).String(),
			MaxSupply: helpers.BipToPip(big.NewInt(100000)).String(),
		},
	}

	appState.UsedChecks = []UsedCheck{
		"00004601d10c33eda76bb16a54a0asddsd8882a57ec34e964aa23e2b5d9aa10957feea",
	}

	if appState.Verify() == nil {
		t.Error("State is not correct")
	}
}

func TestHashToString(t *testing.T) {
	hash := Hash{5}
	if hash.String() != "Mx0500000000000000000000000000000000000000000000000000000000000000" {
		t.Error("Hash hex not the same")
	}
}

func TestHashToBytes(t *testing.T) {
	b := []byte{5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hash := BytesToHash(b)

	if !bytes.Equal(b, hash.Bytes()) {
		t.Error("Bytes are the same")
	}
}

func TestEmptyHash(t *testing.T) {
	if EmptyHash(Hash{}) != true {
		t.Error("Hash is not empty")
	}
}

func TestHashToSetBytes(t *testing.T) {
	b1 := []byte{1, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	h := Hash{}
	h.SetBytes(b1)

	if !bytes.Equal(b1, h.Bytes()) {
		t.Error("Bytes are not the same")
	}

	b2 := []byte{2, 1, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	h.SetBytes(b2)

	if !bytes.Equal(b1, h.Bytes()) {
		t.Error("Bytes are not the same")
	}
}

func TestHashToSet(t *testing.T) {
	h1, h2 := Hash{5}, Hash{}
	h2.Set(h1)

	if !bytes.Equal(h1.Bytes(), h2.Bytes()) {
		t.Error("Bytes are not the same")
	}
}

func TestHashToMarshalText(t *testing.T) {
	b := []byte{77, 120, 48, 53, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48}
	h := Hash{5}
	hashBytes, err := h.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(hashBytes, b) {
		t.Error("Bytes are not the same")
	}

	h2 := Hash{}
	if err = h2.UnmarshalText(hashBytes); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(h2.Bytes(), h.Bytes()) {
		t.Error("Bytes are not the same")
	}
}

func TestGetVersionFromSymbol(t *testing.T) {
	if GetVersionFromSymbol("BIP-5") != 5 {
		t.Error("Coin version is incorrect")
	}

	if GetVersionFromSymbol("BIP") != 0 {
		t.Error("Coin version is incorrect")
	}
}

func TestAddressToSetBytes(t *testing.T) {
	b1 := []byte{5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	a := Address{}
	a.SetBytes(b1)
	if !bytes.Equal(b1, a.Bytes()) {
		t.Error("Bytes are not the same")
	}

	b2 := []byte{1, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	a.SetBytes(b2)

	if !bytes.Equal(b1, a.Bytes()) {
		t.Error("Bytes are not the same")
	}
}

func TestAddressToSet(t *testing.T) {
	a1, a2 := Address{5}, Address{}
	a2.Set(a1)

	if !bytes.Equal(a1.Bytes(), a2.Bytes()) {
		t.Error("Bytes are not the same")
	}
}

func TestAddressToMarshalText(t *testing.T) {
	b := []byte{77, 120, 48, 49, 48, 50, 48, 51, 48, 52, 48, 53, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48}
	a := Address{1, 2, 3, 4, 5}
	hashBytes, err := a.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(hashBytes, b) {
		t.Error("Bytes are not the same")
	}

	a2 := Address{}
	if err := a2.UnmarshalText(hashBytes); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(a2.Bytes(), a.Bytes()) {
		t.Error("Bytes are not the same")
	}
}

func TestHexToPubkey(t *testing.T) {
	p1, p2 := Pubkey{10, 12}, HexToPubkey("Mp0a0c000000000000000000000000000000000000000000000000000000000000")
	if !p1.Equals(p2) {
		t.Error("Pubkeys are not equal")
	}
}

func TestBytesToPubkey(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	p := BytesToPubkey(b)
	if !bytes.Equal(p.Bytes(), b) {
		t.Error("Bytes are not the same")
	}
}

func TestPubkeyToSetBytes(t *testing.T) {
	b, p := []byte{1, 2, 3, 4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Pubkey{}
	p.SetBytes(b)
	if !bytes.Equal(p.Bytes(), b) {
		t.Error("Bytes are not the same")
	}

	b2 := []byte{5, 1, 2, 3, 4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	p.SetBytes(b2)
	if !bytes.Equal(p.Bytes(), b) {
		t.Error("Bytes are not the same")
	}
}

func TestPubkeyToMarshalText(t *testing.T) {
	b := []byte{77, 112, 48, 53, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48}
	p := Pubkey{5}
	hashBytes, err := p.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(hashBytes, b) {
		t.Error("Bytes are not the same")
	}
}
