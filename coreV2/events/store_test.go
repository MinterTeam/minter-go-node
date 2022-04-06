package events

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	db "github.com/tendermint/tm-db"
	"testing"
)

func TestIEventsDB(t *testing.T) {
	store := NewEventsStore(db.NewMemDB())

	{
		event := &StakeMoveEvent{
			Address:           types.HexToAddress("Mx04bea23efb744dc93b4fda4c20bf4a21c6e195f1"),
			Amount:            "111497225000000000000",
			Coin:              123,
			CandidatePubKey:   types.HexToPubkey("Mp9e13f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f58c6"),
			ToCandidatePubKey: types.HexToPubkey("Mp0003f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f5000"),
		}
		store.AddEvent(event)
	}
	err := store.CommitEvents(123)
	if err != nil {
		t.Fatal(err)
	}
	{
		event := &RewardEvent{
			Role:            RoleDevelopers.String(),
			Address:         types.HexToAddress("Mx04bea23efb744dc93b4fda4c20bf4a21c6e195f1"),
			Amount:          "111497225000000000000",
			ValidatorPubKey: types.HexToPubkey("Mp9e13f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f58c6"),
		}
		store.AddEvent(event)
	}
	{
		event := &StakeKickEvent{
			Coin:            1,
			Address:         types.HexToAddress("Mx18467bbb64a8edf890201d526c35957d82be3d95"),
			Amount:          "891977800000000000000",
			ValidatorPubKey: types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd1c"),
		}
		store.AddEvent(event)
	}
	err = store.CommitEvents(12)
	if err != nil {
		t.Fatal(err)
	}

	{
		pubkey := types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd11")
		event := &UnbondEvent{
			Coin:            1,
			Address:         types.HexToAddress("Mx18467bbb64a8edf890201d526c35957d82be3d91"),
			Amount:          "891977800000000000001",
			ValidatorPubKey: &pubkey,
		}
		store.AddEvent(event)
	}
	{
		pubkey := types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd12")
		event := &UnbondEvent{
			Coin:            2,
			Address:         types.HexToAddress("Mx18467bbb64a8edf890201d526c35957d82be3d92"),
			Amount:          "891977800000000000002",
			ValidatorPubKey: &pubkey,
		}
		store.AddEvent(event)
	}
	err = store.CommitEvents(14)
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
		store.AddEvent(event)
	}
	err = store.CommitEvents(11)
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
	if loadEvents[1].(*StakeKickEvent).Coin != 1 {
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
	if loadEvents[0].(*UnbondEvent).Coin != 1 {
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
	if loadEvents[1].(*UnbondEvent).Coin != 2 {
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
	if loadEvents[0].(*SlashEvent).Coin != 10 {
		t.Fatal("invalid Coin")
	}

	loadEvents = store.LoadEvents(123)

	if len(loadEvents) != 1 {
		t.Fatal("count of events not equal 1")
	}

	if loadEvents[0].Type() != TypeStakeMoveEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[0].(*StakeMoveEvent).Amount != "111497225000000000000" {
		t.Fatal("invalid Amount")
	}
	if loadEvents[0].(*StakeMoveEvent).Address.String() != "Mx04bea23efb744dc93b4fda4c20bf4a21c6e195f1" {
		t.Fatal("invalid Address")
	}
	if loadEvents[0].(*StakeMoveEvent).CandidatePubKey.String() != "Mp9e13f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f58c6" {
		t.Fatal("invalid PubKey")
	}
	if loadEvents[0].(*StakeMoveEvent).ToCandidatePubKey.String() != "Mp0003f2f5468dd782b316444fbd66595e13dba7d7bd3efa1becd50b42045f5000" {
		t.Fatal("invalid ToPubKey")
	}
}

func TestIEventsDBm2(t *testing.T) {
	store := NewEventsStore(db.NewMemDB())

	{
		event := &UpdateCommissionsEvent{
			Send: "1000000000",
		}
		store.AddEvent(event)
	}
	{
		event := &UpdateNetworkEvent{
			Version: "m2",
		}
		store.AddEvent(event)
	}
	err := store.CommitEvents(12)
	if err != nil {
		t.Fatal(err)
	}

	loadEvents := store.LoadEvents(12)

	if len(loadEvents) != 2 {
		t.Fatalf("count of events not equal 2, got %d", len(loadEvents))
	}

	if loadEvents[0].Type() != TypeUpdateCommissionsEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[0].(*UpdateCommissionsEvent).Send != "1000000000" {
		t.Fatal("invalid Amount")
	}

}

func TestIEventsJail(t *testing.T) {
	store := NewEventsStore(db.NewMemDB())
	{
		event := &JailEvent{
			ValidatorPubKey: types.HexToPubkey("Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd10"),
			JailedUntil:     1234,
		}
		store.AddEvent(event)
	}
	err := store.CommitEvents(12)
	if err != nil {
		t.Fatal(err)
	}

	loadEvents := store.LoadEvents(12)

	if len(loadEvents) != 1 {
		t.Fatalf("count of events not equal 1, got %d", len(loadEvents))
	}

	if loadEvents[0].Type() != TypeJailEvent {
		t.Fatal("invalid event type")
	}
	if loadEvents[0].(*JailEvent).ValidatorPubKeyString() != "Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd10" {
		t.Fatal("invalid public key")
	}
	if loadEvents[0].(*JailEvent).JailedUntil != 1234 {
		t.Fatal("invalid height")
	}

	marshalJSON, err := tmjson.Marshal(loadEvents[0])
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", marshalJSON)
}

func TestIEventsNil(t *testing.T) {
	store := NewEventsStore(db.NewMemDB())
	err := store.CommitEvents(12)
	if err != nil {
		t.Fatal(err)
	}

	if store.LoadEvents(12) == nil {
		t.Fatalf("nil")
	}

	if store.LoadEvents(13) != nil {
		t.Fatalf("not nil")
	}
}
