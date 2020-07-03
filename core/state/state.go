package state

import (
	"encoding/hex"
	"fmt"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/state/app"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/state/checker"
	"github.com/MinterTeam/minter-go-node/core/state/checks"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/state/frozenfunds"
	"github.com/MinterTeam/minter-go-node/core/state/validators"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"log"
	"math/big"
	"sync"
)

type Interface interface {
	isValue_State()
}

type CheckState struct {
	state *State
}

func NewCheckState(state *State) *CheckState {
	return &CheckState{state: state}
}

func (cs *CheckState) isValue_State() {}

func (cs *CheckState) Lock() {
	cs.state.lock.Lock()
}

func (cs *CheckState) Unlock() {
	cs.state.lock.Unlock()
}

func (cs *CheckState) RLock() {
	cs.state.lock.RLock()
}

func (cs *CheckState) RUnlock() {
	cs.state.lock.RUnlock()
}

func (cs *CheckState) Validators() validators.RValidators {
	return cs.state.Validators
}
func (cs *CheckState) App() app.RApp {
	return cs.state.App
}
func (cs *CheckState) Candidates() candidates.RCandidates {
	return cs.state.Candidates
}
func (cs *CheckState) FrozenFunds() frozenfunds.RFrozenFunds {
	return cs.state.FrozenFunds
}
func (cs *CheckState) Accounts() accounts.RAccounts {
	return cs.state.Accounts
}
func (cs *CheckState) Coins() coins.RCoins {
	return cs.state.Coins
}
func (cs *CheckState) Checks() checks.RChecks {
	return cs.state.Checks
}
func (cs *CheckState) Tree() tree.ReadOnlyTree {
	return cs.state.Tree()
}

type State struct {
	App         *app.App
	Validators  *validators.Validators
	Candidates  *candidates.Candidates
	FrozenFunds *frozenfunds.FrozenFunds
	Accounts    *accounts.Accounts
	Coins       *coins.Coins
	Checks      *checks.Checks
	Checker     *checker.Checker

	db             db.DB
	events         eventsdb.IEventsDB
	tree           tree.MTree
	keepLastStates int64
	bus            *bus.Bus

	lock sync.RWMutex
}

func (s *State) isValue_State() {}

func NewState(height uint64, db db.DB, events eventsdb.IEventsDB, cacheSize int, keepLastStates int64) (*State, error) {
	iavlTree := tree.NewMutableTree(height, db, cacheSize)

	state, err := newStateForTree(iavlTree, events, db, keepLastStates)
	if err != nil {
		return nil, err
	}

	state.Candidates.LoadCandidates()
	state.Candidates.LoadStakes()
	state.Validators.LoadValidators()

	return state, nil
}

func NewCheckStateAtHeight(height uint64, db db.DB) (*CheckState, error) {
	iavlTree := tree.NewMutableTree(0, db, 1024)
	_, err := iavlTree.LazyLoadVersion(int64(height))
	if err != nil {
		return nil, err
	}

	return newCheckStateForTree(iavlTree, nil, nil, 0)
}

func (s *State) Tree() tree.MTree {
	return s.tree
}

func (s *State) Lock() {
	s.lock.Lock()
}

func (s *State) Unlock() {
	s.lock.Unlock()
}

func (s *State) RLock() {
	s.lock.RLock()
}

func (s *State) RUnlock() {
	s.lock.RUnlock()
}

func (s *State) Check() error {
	volumeDeltas := s.Checker.VolumeDeltas()
	for coin, delta := range s.Checker.Deltas() {
		volume := volumeDeltas[coin]
		if volume == nil {
			volume = big.NewInt(0)
		}

		if delta.Cmp(volume) != 0 {
			return fmt.Errorf("invariants error on coin %s: %s", coin.String(), big.NewInt(0).Sub(volumeDeltas[coin], delta).String())
		}
	}

	return nil
}

func (s *State) Commit() ([]byte, error) {
	s.Checker.Reset()

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

	hash, version, err := s.tree.SaveVersion()
	if err != nil {
		return hash, err
	}

	if s.keepLastStates < version-1 {
		_ = s.tree.DeleteVersion(version - s.keepLastStates)
	}

	return hash, nil
}

func (s *State) Import(state types.AppState) error {
	s.App.SetMaxGas(state.MaxGas)
	s.App.SetTotalSlashed(helpers.StringToBigInt(state.TotalSlashed))

	for _, a := range state.Accounts {
		if a.MultisigData != nil {
			s.Accounts.CreateMultisig(a.MultisigData.Weights, a.MultisigData.Addresses, a.MultisigData.Threshold, 1)
		}

		s.Accounts.SetNonce(a.Address, a.Nonce)

		for _, b := range a.Balance {
			s.Accounts.SetBalance(a.Address, b.Coin, helpers.StringToBigInt(b.Value))
		}
	}

	for _, c := range state.Coins {
		s.Coins.Create(c.Symbol, c.Name, helpers.StringToBigInt(c.Volume), c.Crr, helpers.StringToBigInt(c.Reserve), helpers.StringToBigInt(c.MaxSupply))
	}

	var vals []*validators.Validator
	for _, v := range state.Validators {
		vals = append(vals, validators.NewValidator(
			v.PubKey,
			v.AbsentTimes,
			helpers.StringToBigInt(v.TotalBipStake),
			helpers.StringToBigInt(v.AccumReward),
			true,
			true,
			true,
			s.bus))
	}
	s.Validators.SetValidators(vals)

	for _, c := range state.Candidates {
		s.Candidates.Create(c.OwnerAddress, c.RewardAddress, c.PubKey, c.Commission)
		if c.Status == candidates.CandidateStatusOnline {
			s.Candidates.SetOnline(c.PubKey)
		}

		s.Candidates.SetTotalStake(c.PubKey, helpers.StringToBigInt(c.TotalBipStake))
		s.Candidates.SetStakes(c.PubKey, c.Stakes, c.Updates)
	}

	for _, hashString := range state.UsedChecks {
		bytes, _ := hex.DecodeString(string(hashString))
		var hash types.Hash
		copy(hash[:], bytes)
		s.Checks.UseCheckHash(hash)
	}

	for _, ff := range state.FrozenFunds {
		s.FrozenFunds.AddFund(ff.Height, ff.Address, *ff.CandidateKey, ff.Coin, helpers.StringToBigInt(ff.Value))
	}

	return nil
}

func (s *State) Export(height uint64) types.AppState {
	state, err := NewCheckStateAtHeight(height, s.db)
	if err != nil {
		log.Panicf("Create new state at height %d failed: %s", height, err)
	}

	appState := new(types.AppState)
	state.App().Export(appState, height)
	state.Validators().Export(appState)
	state.Candidates().Export(appState)
	state.FrozenFunds().Export(appState, height)
	state.Accounts().Export(appState)
	state.Coins().Export(appState)
	state.Checks().Export(appState)

	return *appState
}

func newCheckStateForTree(iavlTree tree.MTree, events eventsdb.IEventsDB, db db.DB, keepLastStates int64) (*CheckState, error) {
	stateForTree, err := newStateForTree(iavlTree, events, db, keepLastStates)
	if err != nil {
		return nil, err
	}

	return NewCheckState(stateForTree), nil
}

func newStateForTree(iavlTree tree.MTree, events eventsdb.IEventsDB, db db.DB, keepLastStates int64) (*State, error) {
	stateBus := bus.NewBus()
	stateBus.SetEvents(events)

	stateChecker := checker.NewChecker(stateBus)

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

	coinsState, err := coins.NewCoins(stateBus, iavlTree)
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
		Checker:     stateChecker,
		bus:         stateBus,

		db:             db,
		events:         events,
		tree:           iavlTree,
		keepLastStates: keepLastStates,
	}

	return state, nil
}
