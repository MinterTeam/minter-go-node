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
	"github.com/MinterTeam/minter-go-node/core/state/exchange"
	"github.com/MinterTeam/minter-go-node/core/state/frozenfunds"
	"github.com/MinterTeam/minter-go-node/core/state/halts"
	"github.com/MinterTeam/minter-go-node/core/state/validators"
	"github.com/MinterTeam/minter-go-node/core/state/waitlist"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/tendermint/iavl"
	db "github.com/tendermint/tm-db"
	"gopkg.in/errgo.v2/fmt/errors"
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

func (cs *CheckState) Export() types.AppState {
	appState := new(types.AppState)
	cs.App().Export(appState, uint64(cs.state.height))
	cs.Validators().Export(appState)
	cs.Candidates().Export(appState)
	cs.WaitList().Export(appState)
	cs.FrozenFunds().Export(appState, uint64(cs.state.height))
	cs.Accounts().Export(appState)
	cs.Coins().Export(appState)
	cs.Checks().Export(appState)
	cs.Halts().Export(appState)
	cs.Swap().Export(appState)

	return *appState
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
func (cs *CheckState) Halts() halts.RHalts {
	return cs.state.Halts
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
func (cs *CheckState) WaitList() waitlist.RWaitList {
	return cs.state.Waitlist
}

func (cs *CheckState) Swap() exchange.RSwap {
	return cs.state.Swap
}

type State struct {
	App            *app.App
	Validators     *validators.Validators
	Candidates     *candidates.Candidates
	FrozenFunds    *frozenfunds.FrozenFunds
	Halts          *halts.HaltBlocks
	Accounts       *accounts.Accounts
	Coins          *coins.Coins
	Checks         *checks.Checks
	Checker        *checker.Checker
	Waitlist       *waitlist.WaitList
	Swap           *exchange.Swap
	db             db.DB
	events         eventsdb.IEventsDB
	tree           tree.MTree
	keepLastStates int64
	bus            *bus.Bus

	lock   sync.RWMutex
	height int64
}

func (s *State) isValue_State() {}

func NewState(height uint64, db db.DB, events eventsdb.IEventsDB, cacheSize int, keepLastStates int64) (*State, error) {
	iavlTree, err := tree.NewMutableTree(height, db, cacheSize)
	if err != nil {
		return nil, err
	}

	state, err := newStateForTree(iavlTree.GetLastImmutable(), events, db, keepLastStates)
	if err != nil {
		return nil, err
	}

	state.tree = iavlTree

	state.Candidates.LoadCandidatesDeliver()
	state.Candidates.LoadStakes()
	state.Validators.LoadValidators()

	return state, nil
}

func NewCheckStateAtHeight(height uint64, db db.DB) (*CheckState, error) {
	iavlTree, err := tree.NewImmutableTree(height, db)
	if err != nil {
		return nil, err
	}
	return newCheckStateForTree(iavlTree, nil, db, 0)
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
			return fmt.Errorf("invariants error on coin %s: %s", coin.String(), big.NewInt(0).Sub(volume, delta).String())
		}
	}

	return nil
}

func (s *State) Commit() ([]byte, error) {
	s.Checker.Reset()

	hash, version, err := s.tree.Commit(
		s.Accounts,
		s.App,
		s.Coins,
		s.Candidates,
		s.Validators,
		s.Checks,
		s.FrozenFunds,
		s.Halts,
		s.Waitlist,
		s.Swap,
	)
	if err != nil {
		return hash, err
	}

	versionToDelete := version - s.keepLastStates - 1
	if versionToDelete < 1 {
		return hash, nil
	}

	if err := s.tree.DeleteVersion(versionToDelete); err != nil {
		log.Printf("DeleteVersion %d error: %s\n", versionToDelete, err)
	}

	s.height = version

	return hash, nil
}

func (s *State) Import(state types.AppState) error {
	s.App.SetMaxGas(state.MaxGas)
	s.App.SetTotalSlashed(helpers.StringToBigInt(state.TotalSlashed))
	s.App.SetCoinsCount(uint32(len(state.Coins)))

	for _, a := range state.Accounts {
		if a.MultisigData != nil {
			var weights []uint32
			for _, weight := range a.MultisigData.Weights {
				weights = append(weights, uint32(weight))
			}
			s.Accounts.CreateMultisig(weights, a.MultisigData.Addresses, uint32(a.MultisigData.Threshold), a.Address)
		}

		s.Accounts.SetNonce(a.Address, a.Nonce)

		for _, b := range a.Balance {
			s.Accounts.SetBalance(a.Address, types.CoinID(b.Coin), helpers.StringToBigInt(b.Value))
		}
	}

	for _, c := range state.Coins {
		var reserve *big.Int
		if c.Reserve != nil {
			reserve = helpers.StringToBigInt(*c.Reserve)
		}
		s.Coins.Create(types.CoinID(c.ID), c.Symbol, c.Name, helpers.StringToBigInt(c.Volume),
			uint32(c.Crr), reserve, helpers.StringToBigInt(c.MaxSupply), c.OwnerAddress)
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

	for _, pubkey := range state.BlockListCandidates {
		s.Candidates.AddToBlockPubKey(pubkey)
	}

	for _, c := range state.Candidates {
		s.Candidates.CreateWithID(c.OwnerAddress, c.RewardAddress, c.ControlAddress, c.PubKey, uint32(c.Commission), uint32(c.ID))
		if c.Status == candidates.CandidateStatusOnline {
			s.Candidates.SetOnline(c.PubKey)
		}

		s.Candidates.SetTotalStake(c.PubKey, helpers.StringToBigInt(c.TotalBipStake))
		s.Candidates.SetStakes(c.PubKey, c.Stakes, c.Updates)
	}
	s.Candidates.RecalculateStakes(state.StartHeight)

	for _, w := range state.Waitlist {
		value, ok := big.NewInt(0).SetString(w.Value, 10)
		if !ok {
			return errors.Newf("Cannot decode %s into big.Int", w.Value)
		}
		s.Waitlist.AddWaitList(w.Owner, s.Candidates.PubKey(uint32(w.CandidateID)), types.CoinID(w.Coin), value)
	}

	for _, hashString := range state.UsedChecks {
		bytes, _ := hex.DecodeString(string(hashString))
		var hash types.Hash
		copy(hash[:], bytes)
		s.Checks.UseCheckHash(hash)
	}

	for _, ff := range state.FrozenFunds {
		s.FrozenFunds.AddFund(ff.Height, ff.Address, *ff.CandidateKey, uint32(ff.CandidateID), types.CoinID(ff.Coin), helpers.StringToBigInt(ff.Value))
	}

	s.Swap.Import(&state)

	return nil
}

func (s *State) Export() types.AppState {
	state, err := NewCheckStateAtHeight(uint64(s.tree.Version()), s.db)
	if err != nil {
		log.Panicf("Create new state at height %d failed: %s", s.tree.Version(), err)
	}

	return state.Export()
}

func newCheckStateForTree(immutableTree *iavl.ImmutableTree, events eventsdb.IEventsDB, db db.DB, keepLastStates int64) (*CheckState, error) {
	stateForTree, err := newStateForTree(immutableTree, events, db, keepLastStates)
	if err != nil {
		return nil, err
	}

	return NewCheckState(stateForTree), nil
}

func newStateForTree(immutableTree *iavl.ImmutableTree, events eventsdb.IEventsDB, db db.DB, keepLastStates int64) (*State, error) {
	stateBus := bus.NewBus()
	stateBus.SetEvents(events)

	stateChecker := checker.NewChecker(stateBus)

	candidatesState := candidates.NewCandidates(stateBus, immutableTree)

	validatorsState := validators.NewValidators(stateBus, immutableTree)

	appState := app.NewApp(stateBus, immutableTree)

	frozenFundsState := frozenfunds.NewFrozenFunds(stateBus, immutableTree)

	accountsState := accounts.NewAccounts(stateBus, immutableTree)

	coinsState := coins.NewCoins(stateBus, immutableTree)

	checksState := checks.NewChecks(immutableTree)

	haltsState := halts.NewHalts(stateBus, immutableTree)

	waitlistState := waitlist.NewWaitList(stateBus, immutableTree)

	swap := exchange.New(stateBus, immutableTree)

	state := &State{
		Validators:  validatorsState,
		App:         appState,
		Candidates:  candidatesState,
		FrozenFunds: frozenFundsState,
		Accounts:    accountsState,
		Coins:       coinsState,
		Checks:      checksState,
		Checker:     stateChecker,
		Halts:       haltsState,
		Waitlist:    waitlistState,
		Swap:        swap,

		height:         immutableTree.Version(),
		bus:            stateBus,
		db:             db,
		events:         events,
		keepLastStates: keepLastStates,
	}

	return state, nil
}
