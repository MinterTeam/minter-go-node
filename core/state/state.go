package state

import (
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/state/app"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/state/checks"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/state/frozenfunds"
	"github.com/MinterTeam/minter-go-node/core/state/validators"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	compact "github.com/klim0v/compact-db"
	db "github.com/tendermint/tm-db"
	"github.com/xujiajun/nutsdb"
)

type State struct {
	App         *app.App
	Validators  *validators.Validators
	Candidates  *candidates.Candidates
	FrozenFunds *frozenfunds.FrozenFunds
	Accounts    *accounts.Accounts
	Coins       *coins.Coins
	Checks      *checks.Checks

	nuts   *nutsdb.DB
	events compact.IEventsDB
	tree   tree.Tree
}

func NewState(height uint64, db db.DB, nuts *nutsdb.DB, events compact.IEventsDB) (*State, error) {
	iavlTree := tree.NewMutableTree(db)
	_, err := iavlTree.LazyLoadVersion(int64(height))
	if err != nil {
		return nil, err
	}

	return newStateForTree(iavlTree, nuts, events)
}

func NewCheckState(state *State) *State {
	s, err := newStateForTree(state.tree, state.nuts, state.events)
	if err != nil {
		panic(err)
	}
	return s
}

func NewCheckStateAtHeight(height uint64, db db.DB) (*State, error) {
	iavlTree := tree.NewImmutableTree(db)
	_, err := iavlTree.LazyLoadVersion(int64(height))
	if err != nil {
		return nil, err
	}

	return newStateForTree(iavlTree, nil, nil)
}

func (s *State) Commit() ([]byte, error) {
	if err := s.Accounts.Commit(); err != nil {
		return nil, err
	}

	if err := s.App.Commit(); err != nil {
		return nil, err
	}

	if err := s.Coins.Commit(); err != nil {
		return nil, err
	}

	if err := s.Candidates.Commit(); err != nil {
		return nil, err
	}

	if err := s.Validators.Commit(); err != nil {
		return nil, err
	}

	if err := s.Checks.Commit(); err != nil {
		return nil, err
	}

	if err := s.FrozenFunds.Commit(); err != nil {
		return nil, err
	}

	hash, _, err := s.tree.SaveVersion()

	return hash, err
}

func (s *State) CheckForInvariants() error {
	panic("implement me")
}

func (s *State) Import(state types.AppState) error {
	s.App.SetMaxGas(state.MaxGas)
	s.App.SetTotalSlashed(state.TotalSlashed)

	for _, a := range state.Accounts {
		s.Accounts.SetNonce(a.Address, a.Nonce)

		//if a.MultisigData != nil {
		//	account.data.MultisigData.Addresses = a.MultisigData.Addresses
		//	account.data.MultisigData.Threshold = a.MultisigData.Threshold
		//	account.data.MultisigData.Weights = a.MultisigData.Weights
		//}

		for _, b := range a.Balance {
			s.Accounts.SetBalance(a.Address, b.Coin, b.Value)
		}
	}

	for _, c := range state.Coins {
		s.Coins.Create(c.Symbol, c.Name, c.Volume, c.Crr, c.ReserveBalance)
	}

	var vals []*validators.Validator
	for _, v := range state.Validators {
		vals = append(vals, validators.NewValidator(
			v.RewardAddress,
			v.PubKey,
			v.Commission,
			v.AbsentTimes,
			v.TotalBipStake,
			v.AccumReward,
			true,
			true,
			true))
	}
	s.Validators.SetValidators(vals)

	for _, c := range state.Candidates {
		s.Candidates.Create(c.OwnerAddress, c.RewardAddress, c.PubKey, c.Commission)
		if c.Status == candidates.CandidateStatusOnline {
			s.Candidates.SetOnline(c.PubKey)
		}

		s.Candidates.SetStakes(c.PubKey, c.Stakes)
	}

	for _, hashString := range state.UsedChecks {
		bytes, _ := hex.DecodeString(string(hashString))
		var hash types.Hash
		copy(hash[:], bytes)
		s.Checks.UseCheckHash(hash)
	}

	for _, ff := range state.FrozenFunds {
		s.FrozenFunds.AddFund(ff.Height, ff.Address, *ff.CandidateKey, ff.Coin, ff.Value)
	}

	return nil
}

func (s *State) Export(height uint64) types.AppState {
	panic("implement me")
}

func newStateForTree(iavlTree tree.Tree, nuts *nutsdb.DB, events compact.IEventsDB) (*State, error) {
	stateBus := bus.NewBus()
	stateBus.SetEvents(events)

	candidatesState, err := candidates.NewCandidates(stateBus, iavlTree)
	if err != nil {
		return nil, err
	}

	validatorsState, err := validators.NewValidators(stateBus, iavlTree)
	if err != nil {
		return nil, err
	}

	appState, err := app.NewApp(stateBus, iavlTree)
	if err != nil {
		return nil, err
	}

	frozenFundsState, err := frozenfunds.NewFrozenFunds(stateBus, iavlTree)
	if err != nil {
		return nil, err
	}

	accountsState, err := accounts.NewAccounts(stateBus, iavlTree)
	if err != nil {
		return nil, err
	}

	coinsState, err := coins.NewCoins(stateBus, iavlTree, nuts)
	if err != nil {
		return nil, err
	}

	checksState, err := checks.NewChecks(iavlTree)
	if err != nil {
		return nil, err
	}

	state := &State{
		Validators:  validatorsState,
		App:         appState,
		Candidates:  candidatesState,
		FrozenFunds: frozenFundsState,
		Accounts:    accountsState,
		Coins:       coinsState,
		Checks:      checksState,

		nuts:   nuts,
		events: events,
		tree:   iavlTree,
	}

	return state, nil
}
