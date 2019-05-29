package state

import (
	"crypto/rand"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tendermint/libs/db"
	"math/big"
	"testing"
)

func TestStake_CalcSimulatedBipValue(t *testing.T) {
	s, err := New(0, db.NewMemDB(), false)
	if err != nil {
		panic(err)
	}

	createTestCandidate(s)

	coin := types.StrToCoinSymbol("ABC")
	value := helpers.BipToPip(big.NewInt(100))
	reserve := helpers.BipToPip(big.NewInt(201))

	s.CreateCoin(coin, "COIN", value, 30, reserve)

	bipValue := (&Stake{
		Coin:     coin,
		Value:    helpers.BipToPip(big.NewInt(52)),
		BipValue: big.NewInt(0),
	}).CalcSimulatedBipValue(s)

	target := "183595287704679693988"
	if bipValue.String() != target {
		t.Fatalf("Bip value is not equals to target. Got %s, expected %s", bipValue, target)
	}
}

func createTestCandidate(stateDB *StateDB) []byte {
	address := types.Address{}
	pubkey := make([]byte, 32)
	rand.Read(pubkey)

	stateDB.CreateCandidate(address, address, pubkey, 10, 0, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1)))

	return pubkey
}
