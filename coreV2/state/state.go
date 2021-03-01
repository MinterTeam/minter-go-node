package state

import (
	"encoding/hex"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/accounts"
	"github.com/MinterTeam/minter-go-node/coreV2/state/app"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checks"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/frozenfunds"
	"github.com/MinterTeam/minter-go-node/coreV2/state/halts"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/state/validators"
	"github.com/MinterTeam/minter-go-node/coreV2/state/waitlist"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/cosmos/iavl"
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

func (cs *CheckState) Export() types.AppState {
	appState := new(types.AppState)
	cs.App().Export(appState)
	cs.Validators().Export(appState)
	cs.Candidates().Export(appState)
	cs.WaitList().Export(appState)
	cs.FrozenFunds().Export(appState, uint64(cs.state.height))
	cs.Accounts().Export(appState)
	cs.Coins().Export(appState)
	cs.Checks().Export(appState)
	cs.Halts().Export(appState)
	cs.Swap().Export(appState)
	cs.Commission().Export(appState)

	return *appState
}

func (cs *CheckState) ExportV1toV2(bipRate float64) types.AppState {
	appState := new(types.AppState)
	cs.App().Export(appState)
	cs.Validators().Export(appState)
	cs.Candidates().ExportV1toV2(appState)
	cs.WaitList().Export(appState)
	cs.FrozenFunds().Export(appState, uint64(cs.state.height))
	cs.Checks().Export(appState) // todo: mb refactor store
	cs.Halts().Export(appState)

	totalUSDCValue := helpers.BipToPip(big.NewInt(1000000000))
	poolUSDCValue := helpers.BipToPip(big.NewInt(10000))
	rate := big.NewFloat(bipRate)
	poolBipValue, _ := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(poolUSDCValue), rate).Int(nil)

	subValues := cs.Accounts().ExportV1(appState, poolBipValue)
	usdcCoinID := cs.Coins().ExportV1(appState, subValues)
	poolTokenVolume := cs.Swap().ExportV1(appState, usdcCoinID, poolUSDCValue, poolBipValue)
	cs.Commission().ExportV1(appState, usdcCoinID)

	lpUSDC := uint64(usdcCoinID) + 1
	appState.Coins = append(appState.Coins, types.Coin{
		ID:           lpUSDC,
		Name:         "Liquidity Pool 0:" + usdcCoinID.String(),
		Symbol:       types.StrToCoinSymbol("PL-1"),
		Volume:       poolTokenVolume.String(),
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    coins.MaxCoinSupply().String(),
		Version:      0,
		OwnerAddress: nil,
		Mintable:     true,
		Burnable:     true,
	})

	appState.Accounts = append(appState.Accounts, types.Account{
		Address: types.HexToAddress("Mxffffffffffffffffffffffffffffffffffffffff"),
		Balance: []types.Balance{
			{
				Coin:  uint64(usdcCoinID),
				Value: big.NewInt(0).Sub(totalUSDCValue, poolUSDCValue).String(),
			},
			{
				Coin:  lpUSDC,
				Value: big.NewInt(0).Sub(poolTokenVolume, big.NewInt(1000)).String(),
			},
		},
		Nonce: 0,
		MultisigData: &types.Multisig{
			Weights:   []uint64{1000},
			Threshold: 667,
			Addresses: []types.Address{
				types.StringToAddress("Mx90b704f155b3cd7f998802ff2ce5c39cb2a9caac"),
			},
		},
	})

	if appState.Accounts[0].Address == [20]byte{} {
		appState.Accounts[0].Balance = append(appState.Accounts[0].Balance, types.Balance{
			Coin:  lpUSDC,
			Value: "1000",
		})
	} else {
		appState.Accounts = append(appState.Accounts, types.Account{
			Address: types.Address{},
			Balance: []types.Balance{
				{
					Coin:  lpUSDC,
					Value: "1000"},
			},
			Nonce:        0,
			MultisigData: nil,
		})
	}
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

func (cs *CheckState) Swap() swap.RSwap {
	return cs.state.Swap
}

func (cs *CheckState) Commission() commission.RCommission {
	return cs.state.Commission
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
	Swap           *swap.Swap
	Commission     *commission.Commission
	db             db.DB
	events         eventsdb.IEventsDB
	tree           tree.MTree
	keepLastStates int64

	bus            *bus.Bus
	lock           sync.RWMutex
	height         int64
	initialVersion int64
}

func (s *State) isValue_State() {}

func NewState(height uint64, db db.DB, events eventsdb.IEventsDB, cacheSize int, keepLastStates int64, initialVersion uint64) (*State, error) {
	iavlTree, err := tree.NewMutableTree(height, db, cacheSize, initialVersion)
	if err != nil {
		return nil, err
	}

	state, err := newStateForTree(iavlTree.GetLastImmutable(), events, db, keepLastStates)
	if err != nil {
		return nil, err
	}

	state.tree = iavlTree
	state.height = int64(height)
	state.initialVersion = int64(initialVersion)
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
	return s.Checker.Check()
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
		s.Commission,
	)
	if err != nil {
		return hash, err
	}

	versionToDelete := version - s.keepLastStates - 1
	if versionToDelete < s.initialVersion {
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
		}
	}

	for _, c := range state.Coins {
		coinID := types.CoinID(c.ID)
		volume := helpers.StringToBigInt(c.Volume)
		maxSupply := helpers.StringToBigInt(c.MaxSupply)
		if c.Crr == 0 {
			s.Coins.CreateToken(coinID, c.Symbol, c.Name, c.Mintable, c.Burnable, volume, maxSupply, c.OwnerAddress)
		} else {
			reserve := helpers.StringToBigInt(c.Reserve)
			s.Coins.Create(coinID, c.Symbol, c.Name, volume, uint32(c.Crr), reserve, maxSupply, c.OwnerAddress)
		}
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
	s.Candidates.RecalculateStakes(uint64(s.height))

	for _, w := range state.Waitlist {
		value := helpers.StringToBigInt(w.Value)
		coinID := types.CoinID(w.Coin)
		s.Waitlist.AddWaitList(w.Owner, s.Candidates.PubKey(uint32(w.CandidateID)), coinID, value)
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
		s.FrozenFunds.AddFund(ff.Height, ff.Address, *ff.CandidateKey, uint32(ff.CandidateID), coinID, value, nil)
	}

	s.Swap.Import(&state)

	s.Checker.RemoveBaseCoin()

	return nil
}

func (s *State) Export() types.AppState {
	state, err := NewCheckStateAtHeight(uint64(s.tree.Version()), s.db)
	if err != nil {
		log.Panicf("Create new state at height %d failed: %s", s.tree.Version(), err)
	}

	return state.Export()
}

// Only for tests
func (s *State) ReloadFromDiskAndExport() types.AppState {
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

	swap := swap.New(stateBus, immutableTree)

	commission := commission.NewCommission(immutableTree)

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
		Commission:  commission,

		height:         immutableTree.Version(),
		bus:            stateBus,
		db:             db,
		events:         events,
		keepLastStates: keepLastStates,
	}

	return state, nil
}
