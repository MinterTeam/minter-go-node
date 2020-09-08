package events

import (
	"encoding/hex"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestIEventsDB(t *testing.T) {
	store := NewEventsStore(db.NewMemDB())

	{
		amount, _ := big.NewInt(0).SetString("111497225000000000000", 10)
		event := &RewardEvent{
			Role:            RoleDevelopers.String(),
			Address:         [20]byte{},
			Amount:          amount.String(),
			ValidatorPubKey: [32]byte{},
		}
		bytesAddress, err := hex.DecodeString("Mx04bea23efb744dc93b4fda4c20bf4a21c6e195f1"[2:])
		if err != nil {
			t.Fatal(err)
		}
		copy(event.Address[:], bytesAddress)
		hexPubKey, err := hex.DecodeString("Mp9e13f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f58c6"[2:])
		if err != nil {
			t.Fatal(err)
		}
		copy(event.ValidatorPubKey[:], hexPubKey)
		store.AddEvent(12, event)
	}
	{
		amount, _ := big.NewInt(0).SetString("891977800000000000000", 10)
		event := &RewardEvent{
			Role:            RoleValidator.String(),
			Address:         [20]byte{},
			Amount:          amount.String(),
			ValidatorPubKey: [32]byte{},
		}
		bytesAddress, err := hex.DecodeString("Mx18467bbb64a8edf890201d526c35957d82be3d95"[2:])
		if err != nil {
			t.Fatal(err)
		}
		copy(event.Address[:], bytesAddress)
		hexPubKey, err := hex.DecodeString("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd1c"[2:])
		if err != nil {
			t.Fatal(err)
		}
		copy(event.ValidatorPubKey[:], hexPubKey)
		store.AddEvent(12, event)
	}
	err := store.CommitEvents()
	if err != nil {
		t.Fatal(err)
	}

	loadEvents := store.LoadEvents(12)

	if len(loadEvents) != 2 {
		t.Fatal("count of events not equal 2")
	}

	if loadEvents[0].(*RewardEvent).Amount != "111497225000000000000" {
		t.Fatal("invalid Amount")
	}

	if loadEvents[0].(*RewardEvent).Address.String() != "Mx04bea23efb744dc93b4fda4c20bf4a21c6e195f1" {
		t.Fatal("invalid Address")
	}

	if loadEvents[1].(*RewardEvent).Amount != "891977800000000000000" {
		t.Fatal("invalid Amount")
	}

	if loadEvents[1].(*RewardEvent).Address.String() != "Mx18467bbb64a8edf890201d526c35957d82be3d95" {
		t.Fatal("invalid Address")
	}

}
