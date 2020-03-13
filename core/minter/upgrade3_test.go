package minter

import (
	eventsdb "github.com/MinterTeam/events-db"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	db "github.com/tendermint/tm-db"
	"testing"
)

func TestApplyUpgrade3(t *testing.T) {
	cState := getState()

	ApplyUpgrade3(cState, emptyEvents{})

	address := types.HexToAddress("Mx8f16fe070b065b958fa6865bd549193827abc0f8")

	{
		targetBalance := helpers.StringToBigInt("5000000000000000000000")
		balance := cState.Accounts.GetBalance(address, types.StrToCoinSymbol("FUFELL14"))
		if balance.Cmp(targetBalance) != 0 {
			t.Fatalf("Balance of %s is not correct", address)
		}
	}

	{
		targetBalance := helpers.StringToBigInt("85000000000000000000000")
		balance := cState.Accounts.GetBalance(address, types.StrToCoinSymbol("VEXILIFERR"))
		if balance.Cmp(targetBalance) != 0 {
			t.Fatalf("Balance of %s is not correct", address)
		}
	}

	if err := cState.Check(); err != nil {
		t.Fatal(err)
	}
}

func getState() *state.State {
	s, err := state.NewState(0, db.NewMemDB(), emptyEvents{}, 1, 1)

	if err != nil {
		panic(err)
	}

	return s
}

type emptyEvents struct{}

func (e emptyEvents) AddEvent(height uint32, event eventsdb.Event) {}
func (e emptyEvents) LoadEvents(height uint32) eventsdb.Events     { return eventsdb.Events{} }
func (e emptyEvents) CommitEvents() error                          { return nil }
