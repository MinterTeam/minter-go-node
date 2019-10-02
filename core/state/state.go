package state

import (
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/state/app"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/state/frozen_funds"
	"github.com/MinterTeam/minter-go-node/core/state/validators"
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
)

type State struct {
	App         *app.App
	Validators  *validators.Validators
	Candidates  *candidates.Candidates
	FrozenFunds *frozen_funds.FrozenFunds
	Accounts    *accounts.Accounts

	height uint64
	db     db.DB
}

func NewState(height uint64, db db.DB) (*State, error) {
	validatorsState, err := validators.NewValidators(db)
	if err != nil {
		return nil, err
	}

	candidatesState, err := candidates.NewCandidates(db)
	if err != nil {
		return nil, err
	}

	appState, err := app.NewApp(db)
	if err != nil {
		return nil, err
	}

	frozenFundsState, err := frozen_funds.NewFrozenFunds(db)
	if err != nil {
		return nil, err
	}

	accountsState, err := accounts.NewAccounts(db)
	if err != nil {
		return nil, err
	}

	state := &State{
		Validators:  validatorsState,
		App:         appState,
		Candidates:  candidatesState,
		FrozenFunds: frozenFundsState,
		Accounts:    accountsState,

		height: height,
		db:     db,
	}

	return state, nil
}

func NewCheckState(state *State) *State {
	panic("implement me")
}

func NewCheckStateAtHeight(height, state *State) (*State, error) {
	panic("implement me")
}

func (s *State) Commit() ([]byte, error) {
	if err := s.Validators.Commit(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *State) Import(state types.AppState) error {
	panic("implement me")
}

func (s *State) CheckForInvariants() error {
	panic("implement me")
}
