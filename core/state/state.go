package state

import (
	"github.com/MinterTeam/minter-go-node/core/state/app"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/state/validators"
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
)

type State struct {
	App        *app.App
	Validators *validators.Validators
	Candidates *candidates.Candidates

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

	state := &State{
		Validators: validatorsState,
		App:        appState,
		Candidates: candidatesState,

		height: height,
		db:     db,
	}

	return state, nil
}

func NewCheckState(state *State) *State {
	panic("implement me")
}

func (s *State) Commit() error {
	if err := s.Validators.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *State) Import(state types.AppState) error {
	panic("implement me")
}

func (s *State) CheckForInvariants() error {
	panic("implement me")
}
