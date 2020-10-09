package events

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
	"testing"
)

func TestIEventsDB(t *testing.T) {
	store := NewEventsStore(db.NewMemDB())

	{
		event := &RewardEvent{
			Role:            RoleDevelopers.String(),
			Address:         types.HexToAddress("Mx04bea23efb744dc93b4fda4c20bf4a21c6e195f1"),
			Amount:          "111497225000000000000",
			ValidatorPubKey: types.HexToPubkey("Mp9e13f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f58c6"),
		}
		store.AddEvent(12, event)
	}
	{
		event := &StakeKickEvent{
			Coin:            1,
			Address:         types.HexToAddress("Mx18467bbb64a8edf890201d526c35957d82be3d95"),
			Amount:          "891977800000000000000",
			ValidatorPubKey: types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd1c"),
		}
		store.AddEvent(12, event)
	}
	err := store.CommitEvents()
	if err != nil {
		t.Fatal(err)
	}

	{
		event := &UnbondEvent{
			Coin:            1,
			Address:         types.HexToAddress("Mx18467bbb64a8edf890201d526c35957d82be3d91"),
			Amount:          "891977800000000000001",
			ValidatorPubKey: types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd11"),
		}
		store.AddEvent(14, event)
	}
	{
		event := &UnbondEvent{
			Coin:            2,
			Address:         types.HexToAddress("Mx18467bbb64a8edf890201d526c35957d82be3d92"),
			Amount:          "891977800000000000002",
			ValidatorPubKey: types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd12"),
		}
		store.AddEvent(14, event)
	}
	err = store.CommitEvents()
	if err != nil {
		t.Fatal(err)
	}

	{
		event := &SlashEvent{
			Coin:            10,
			Address:         types.HexToAddress("Mx18467bbb64a8edf890201d526c35957d82be3d10"),
			Amount:          "891977800000000000010",
			ValidatorPubKey: types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd10"),
		}
		store.AddEvent(11, event)
	}
	err = store.CommitEvents()
	if err != nil {
		t.Fatal(err)
	}

	loadEvents := store.LoadEvents(12)

	if len(loadEvents) != 2 {
		t.Fatalf("count of events not equal 2, got %d", len(loadEvents))
	}

	if loadEvents[0].Type() != TypeRewardEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[0].(*RewardEvent).Amount != "111497225000000000000" {
		t.Fatal("invalid Amount")
	}
	if loadEvents[0].(*RewardEvent).Address.String() != "Mx04bea23efb744dc93b4fda4c20bf4a21c6e195f1" {
		t.Fatal("invalid Address")
	}
	if loadEvents[0].(*RewardEvent).ValidatorPubKey.String() != "Mp9e13f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f58c6" {
		t.Fatal("invalid PubKey")
	}
	if loadEvents[0].(*RewardEvent).Role != RoleDevelopers.String() {
		t.Fatal("invalid Role")
	}

	if loadEvents[1].Type() != TypeStakeKickEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[1].(*StakeKickEvent).Amount != "891977800000000000000" {
		t.Fatal("invalid Amount")
	}
	if loadEvents[1].(*StakeKickEvent).Address.String() != "Mx18467bbb64a8edf890201d526c35957d82be3d95" {
		t.Fatal("invalid Address")
	}
	if loadEvents[1].(*StakeKickEvent).ValidatorPubKey.String() != "Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd1c" {
		t.Fatal("invalid PubKey")
	}
	if loadEvents[1].(*StakeKickEvent).Coin.Uint32() != 1 {
		t.Fatal("invalid Coin")
	}

	loadEvents = store.LoadEvents(14)

	if len(loadEvents) != 2 {
		t.Fatal("count of events not equal 2")
	}

	if loadEvents[0].Type() != TypeUnbondEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[0].(*UnbondEvent).Amount != "891977800000000000001" {
		t.Fatal("invalid Amount")
	}
	if loadEvents[0].(*UnbondEvent).Address.String() != "Mx18467bbb64a8edf890201d526c35957d82be3d91" {
		t.Fatal("invalid Address")
	}
	if loadEvents[0].(*UnbondEvent).ValidatorPubKey.String() != "Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd11" {
		t.Fatal("invalid PubKey")
	}
	if loadEvents[0].(*UnbondEvent).Coin.Uint32() != 1 {
		t.Fatal("invalid Coin")
	}

	if loadEvents[1].Type() != TypeUnbondEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[1].(*UnbondEvent).Amount != "891977800000000000002" {
		t.Fatal("invalid Amount")
	}
	if loadEvents[1].(*UnbondEvent).Address.String() != "Mx18467bbb64a8edf890201d526c35957d82be3d92" {
		t.Fatal("invalid Address")
	}
	if loadEvents[1].(*UnbondEvent).ValidatorPubKey.String() != "Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd12" {
		t.Fatal("invalid PubKey")
	}
	if loadEvents[1].(*UnbondEvent).Coin.Uint32() != 2 {
		t.Fatal("invalid Coin")
	}

	loadEvents = store.LoadEvents(11)

	if len(loadEvents) != 1 {
		t.Fatal("count of events not equal 1")
	}

	if loadEvents[0].Type() != TypeSlashEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[0].(*SlashEvent).Amount != "891977800000000000010" {
		t.Fatal("invalid Amount")
	}
	if loadEvents[0].(*SlashEvent).Address.String() != "Mx18467bbb64a8edf890201d526c35957d82be3d10" {
		t.Fatal("invalid Address")
	}
	if loadEvents[0].(*SlashEvent).ValidatorPubKey.String() != "Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd10" {
		t.Fatal("invalid PubKey")
	}
	if loadEvents[0].(*SlashEvent).Coin.Uint32() != 10 {
		t.Fatal("invalid Coin")
	}
}
