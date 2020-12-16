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
	"github.com/MinterTeam/minter-go-node/core/state/halts"
	"github.com/MinterTeam/minter-go-node/core/state/validators"
	"github.com/MinterTeam/minter-go-node/core/state/waitlist"
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

func (cs *CheckState) Export(height uint64) types.AppState {
	return cs.state.Export(height)
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
func (cs *CheckState) Tree() tree.ReadOnlyTree {
	return cs.state.Tree()
}

type State struct {
	App         *app.App
	Validators  *validators.Validators
	Candidates  *candidates.Candidates
	FrozenFunds *frozenfunds.FrozenFunds
	Halts       *halts.HaltBlocks
	Accounts    *accounts.Accounts
	Coins       *coins.Coins
	Checks      *checks.Checks
	Checker     *checker.Checker
	Waitlist    *waitlist.WaitList

	db             db.DB
	events         eventsdb.IEventsDB
	tree           tree.MTree
	keepLastStates int64
	bus            *bus.Bus

	lock sync.RWMutex
}

func (s *State) isValue_State() {}

func NewState(height uint64, db db.DB, events eventsdb.IEventsDB, cacheSize int, keepLastStates int64) (*State, error) {
	iavlTree, err := tree.NewMutableTree(height, db, cacheSize)
	if err != nil {
		return nil, err
	}

	state, err := newStateForTree(iavlTree, events, db, keepLastStates)
	if err != nil {
		return nil, err
	}

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

const countBatchBlocksDelete = 60

func (s *State) Commit() ([]byte, error) {
	s.Checker.Reset()

	s.tree.GlobalLock()
	defer s.tree.GlobalUnlock()

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

	if err := s.Halts.Commit(); err != nil {
		return nil, err
	}

	if err := s.Waitlist.Commit(); err != nil {
		return nil, err
	}

	hash, version, err := s.tree.SaveVersion()
	if err != nil {
		return hash, err
	}

	versionToDelete := version - s.keepLastStates - 1
	if versionToDelete < 1 {
		return hash, nil
	}

	if err := s.tree.DeleteVersionIfExists(versionToDelete); err != nil {
		log.Printf("DeleteVersion %d error: %s\n", versionToDelete, err)
	}

	return hash, nil
}

func (s *State) Import(state types.AppState) error {
	s.App.SetMaxGas(state.MaxGas)
	totalSlash := helpers.StringToBigInt(state.TotalSlashed)
	s.App.SetTotalSlashed(totalSlash)
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
			balance := helpers.StringToBigInt(b.Value)
			coinID := types.CoinID(b.Coin)
			s.Accounts.SetBalance(a.Address, coinID, balance)
			s.Checker.AddCoin(coinID, new(big.Int).Neg(balance))
		}
	}

	for _, c := range state.Coins {
		coinID := types.CoinID(c.ID)
		volume := helpers.StringToBigInt(c.Volume)
		reserve := helpers.StringToBigInt(c.Reserve)
		s.Coins.Create(coinID, c.Symbol, c.Name, volume, uint32(c.Crr), reserve, helpers.StringToBigInt(c.MaxSupply), c.OwnerAddress)
		s.Checker.AddCoin(types.GetBaseCoinID(), new(big.Int).Neg(reserve))
		s.Checker.AddCoinVolume(coinID, new(big.Int).Neg(volume))
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
		value := helpers.StringToBigInt(w.Value)
		coinID := types.CoinID(w.Coin)
		s.Waitlist.AddWaitList(w.Owner, s.Candidates.PubKey(uint32(w.CandidateID)), coinID, value)
		s.Checker.AddCoin(coinID, new(big.Int).Neg(value))
	}

	for _, hashString := range state.UsedChecks {
		bytes, _ := hex.DecodeString(string(hashString))
		var hash types.Hash
		copy(hash[:], bytes)
		s.Checks.UseCheckHash(hash)
	}

	for _, ff := range state.FrozenFunds {
		coinID := types.CoinID(ff.Coin)
		value := helpers.StringToBigInt(ff.Value)
		s.FrozenFunds.AddFund(ff.Height, ff.Address, *ff.CandidateKey, uint32(ff.CandidateID), coinID, value)
		s.Checker.AddCoin(coinID, new(big.Int).Neg(value))
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
	state.WaitList().Export(appState)
	state.FrozenFunds().Export(appState, height)
	state.Accounts().Export(appState)
	state.Coins().Export(appState)
	state.Checks().Export(appState)
	state.Halts().Export(appState)

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

	haltsState, err := halts.NewHalts(stateBus, iavlTree)
	if err != nil {
		return nil, err
	}

	waitlistState, err := waitlist.NewWaitList(stateBus, iavlTree)
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
		Halts:       haltsState,
		Waitlist:    waitlistState,

		bus: stateBus,

		db:             db,
		events:         events,
		tree:           iavlTree,
		keepLastStates: keepLastStates,
	}

	return state, nil
}
