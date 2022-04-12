package state

import (
	"encoding/hex"
	"log"
	"math/big"
	"sync"

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
	"github.com/MinterTeam/minter-go-node/coreV2/state/update"
	"github.com/MinterTeam/minter-go-node/coreV2/state/validators"
	"github.com/MinterTeam/minter-go-node/coreV2/state/waitlist"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/cosmos/iavl"
	db "github.com/tendermint/tm-db"
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
	cs.Updates().Export(appState)

	return *appState
}

func (cs *CheckState) Updates() update.RUpdate {
	return cs.state.Updates
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
func (cs *CheckState) InitialHeight() int64 {
	return cs.state.InitialVersion
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
	if cs.state.SwapV2 != nil {
		return cs.state.SwapV2
	}
	return cs.state.Swap
}

func (cs *CheckState) Commission() commission.RCommission {
	return cs.state.Commission
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
	Swap        *swap.Swap
	SwapV2      *swap.SwapV2
	Commission  *commission.Commission
	Updates     *update.Update

	db     db.DB
	events eventsdb.IEventsDB
	tree   tree.MTree

	keepLastStates int64
	bus            *bus.Bus
	lock           sync.RWMutex
	height         int64
	InitialVersion int64
}

func (s *State) Bus() *bus.Bus {
	return s.bus
}

func (s *State) isValue_State() {}

func NewState(height uint64, db db.DB, events eventsdb.IEventsDB, cacheSize int, keepLastStates int64, initialVersion uint64) (*State, error) {
	iavlTree, err := tree.NewMutableTree(height, db, cacheSize, initialVersion)
	if err != nil {
		return nil, err
	}

	state, err := newStateForTree(iavlTree.GetLastImmutable(), events, db, keepLastStates) // todo
	if err != nil {
		return nil, err
	}

	state.tree = iavlTree
	state.height = int64(height)
	state.InitialVersion = int64(initialVersion)

	state.Candidates.LoadCandidatesDeliver()
	state.Candidates.LoadStakes()
	state.Validators.LoadValidators()

	return state, nil
}
func NewStateV3(height uint64, db db.DB, events eventsdb.IEventsDB, cacheSize int, keepLastStates int64, initialVersion uint64) (*State, error) {
	iavlTree, err := tree.NewMutableTree(height, db, cacheSize, initialVersion)
	if err != nil {
		return nil, err
	}

	state, err := newStateForTreeV2(iavlTree.GetLastImmutable(), events, db, keepLastStates) // todo
	if err != nil {
		return nil, err
	}

	state.tree = iavlTree
	state.height = int64(height)
	state.InitialVersion = int64(initialVersion)

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
func NewCheckStateAtHeightV3(height uint64, db db.DB) (*CheckState, error) {
	iavlTree, err := tree.NewImmutableTree(height, db)
	if err != nil {
		return nil, err
	}
	return newCheckStateForTreeV2(iavlTree, nil, db, 0)
}

func (s *State) Tree() tree.MTree {
	return s.tree
}

func (s *State) Lock() {
	s.lock.Lock()
}

func (s *State) GetSwap() interface {
	Commit(db *iavl.MutableTree, version int64) error
	SetImmutableTree(immutableTree *iavl.ImmutableTree)
} {
	if s.SwapV2 != nil {
		return s.SwapV2
	}
	return s.Swap
}

func (s *State) Swapper() interface {
	PairSellWithOrders(coin0, coin1 types.CoinID, amount0In, minAmount1Out *big.Int) (*big.Int, *big.Int, uint32, *swap.ChangeDetailsWithOrders, []*swap.OrderDetail)
	PairBuyWithOrders(coin0, coin1 types.CoinID, maxAmount0In, amount1Out *big.Int) (*big.Int, *big.Int, uint32, *swap.ChangeDetailsWithOrders, []*swap.OrderDetail)
	PairAddOrder(coinWantBuy, coinWantSell types.CoinID, wantBuyAmount, wantSellAmount *big.Int, sender types.Address, block uint64) (uint32, uint32)
	PairBuy(coin0, coin1 types.CoinID, maxAmount0In, amount1Out *big.Int) (*big.Int, *big.Int, uint32)
	PairSell(coin0, coin1 types.CoinID, amount0In, minAmount1Out *big.Int) (*big.Int, *big.Int, uint32)
	PairMint(coin0, coin1 types.CoinID, amount0, maxAmount1, totalSupply *big.Int) (*big.Int, *big.Int, *big.Int)
	PairCreate(coin0, coin1 types.CoinID, amount0, amount1 *big.Int) (*big.Int, *big.Int, *big.Int, uint32)
	PairBurn(coin0, coin1 types.CoinID, liquidity, minAmount0, minAmount1, totalSupply *big.Int) (*big.Int, *big.Int)
	PairRemoveLimitOrder(id uint32) (types.CoinID, *big.Int)
	ExpireOrders(beforeHeight uint64)
	Import(state *types.AppState)
	GetSwapper(coinA, coinB types.CoinID) swap.EditableChecker
	SwapPool(coinA, coinB types.CoinID) (reserve0, reserve1 *big.Int, id uint32)
} {
	if s.SwapV2 != nil {
		return s.SwapV2
	}
	return s.Swap
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
		s.GetSwap(),
		s.Commission,
		s.Updates,
	)
	if err != nil {
		return hash, err
	}

	s.height = version

	versionToDelete := version - s.keepLastStates - 1
	if versionToDelete < s.InitialVersion {
		return hash, nil
	}

	if err := s.tree.DeleteVersion(versionToDelete); err != nil {
		log.Printf("DeleteVersion %d error: %s\n", versionToDelete, err)
	}

	return hash, nil
}

func (s *State) Import(state types.AppState, version string) error {
	defer s.Checker.RemoveBaseCoin()

	s.App.SetReward(helpers.StringToBigInt(state.PrevReward.Reward), helpers.StringToBigInt(state.PrevReward.Reward))
	s.App.SetMaxGas(state.MaxGas)
	s.App.SetCoinsCount(uint32(len(state.Coins)))

	totalSlash := helpers.StringToBigInt(state.TotalSlashed)
	s.App.SetTotalSlashed(totalSlash)
	s.Checker.AddCoin(types.GetBaseCoinID(), totalSlash)

	for _, a := range state.Accounts {
		if a.MultisigData != nil {
			var weights []uint32
			for _, weight := range a.MultisigData.Weights {
				weights = append(weights, uint32(weight))
			}
			s.Accounts.CreateMultisig(weights, a.MultisigData.Addresses, uint32(a.MultisigData.Threshold), a.Address)
		}

		s.Accounts.SetNonce(a.Address, a.Nonce)
		if a.LockStakeUntilBlock > 0 {
			s.Accounts.SetLockStakeUntilBlock(a.Address, a.LockStakeUntilBlock)
		}
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
			s.Coins.ImportToken(coinID, c.Symbol, c.Name, c.Mintable, c.Burnable, volume, maxSupply, c.OwnerAddress, c.Version)
		} else {
			reserve := helpers.StringToBigInt(c.Reserve)
			s.Coins.ImportCoin(coinID, c.Symbol, c.Name, volume, uint32(c.Crr), reserve, maxSupply, c.OwnerAddress, c.Version)
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
		s.Candidates.CreateWithID(c.OwnerAddress, c.RewardAddress, c.ControlAddress, c.PubKey, uint32(c.Commission), uint32(c.ID), c.LastEditCommissionHeight, c.JailedUntil)
		if c.Status == candidates.CandidateStatusOnline {
			s.Candidates.SetOnline(c.PubKey)
		}

		s.Candidates.SetTotalStake(c.PubKey, helpers.StringToBigInt(c.TotalBipStake))
		s.Candidates.SetStakes(c.PubKey, c.Stakes, c.Updates)
	}

	if len(state.DeletedCandidates) > 0 {
		s.Candidates.SetDeletedCandidates(state.DeletedCandidates)
	}

	if version == "" {
		s.Candidates.RecalculateStakes(uint64(s.height)) // RecalculateStakesV2
	} else {
		s.Candidates.RecalculateStakesV2(uint64(s.height))
	}

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
		s.FrozenFunds.AddFund(ff.Height, ff.Address, ff.CandidateKey, uint32(ff.CandidateID), coinID, value, uint32(ff.MoveToCandidateID))
	}

	s.Swapper().Import(&state)

	c := state.Commission
	com := &commission.Price{
		Coin:                    types.CoinID(c.Coin),
		PayloadByte:             helpers.StringToBigInt(c.PayloadByte),
		Send:                    helpers.StringToBigInt(c.Send),
		BuyBancor:               helpers.StringToBigInt(c.BuyBancor),
		SellBancor:              helpers.StringToBigInt(c.SellBancor),
		SellAllBancor:           helpers.StringToBigInt(c.SellAllBancor),
		BuyPoolBase:             helpers.StringToBigInt(c.BuyPoolBase),
		BuyPoolDelta:            helpers.StringToBigInt(c.BuyPoolDelta),
		SellPoolBase:            helpers.StringToBigInt(c.SellPoolBase),
		SellPoolDelta:           helpers.StringToBigInt(c.SellPoolDelta),
		SellAllPoolBase:         helpers.StringToBigInt(c.SellAllPoolBase),
		SellAllPoolDelta:        helpers.StringToBigInt(c.SellAllPoolDelta),
		CreateTicker3:           helpers.StringToBigInt(c.CreateTicker3),
		CreateTicker4:           helpers.StringToBigInt(c.CreateTicker4),
		CreateTicker5:           helpers.StringToBigInt(c.CreateTicker5),
		CreateTicker6:           helpers.StringToBigInt(c.CreateTicker6),
		CreateTicker7to10:       helpers.StringToBigInt(c.CreateTicker7_10),
		CreateCoin:              helpers.StringToBigInt(c.CreateCoin),
		CreateToken:             helpers.StringToBigInt(c.CreateToken),
		RecreateCoin:            helpers.StringToBigInt(c.RecreateCoin),
		RecreateToken:           helpers.StringToBigInt(c.RecreateToken),
		DeclareCandidacy:        helpers.StringToBigInt(c.DeclareCandidacy),
		Delegate:                helpers.StringToBigInt(c.Delegate),
		Unbond:                  helpers.StringToBigInt(c.Unbond),
		RedeemCheck:             helpers.StringToBigInt(c.RedeemCheck),
		SetCandidateOn:          helpers.StringToBigInt(c.SetCandidateOn),
		SetCandidateOff:         helpers.StringToBigInt(c.SetCandidateOff),
		CreateMultisig:          helpers.StringToBigInt(c.CreateMultisig),
		MultisendBase:           helpers.StringToBigInt(c.MultisendBase),
		MultisendDelta:          helpers.StringToBigInt(c.MultisendDelta),
		EditCandidate:           helpers.StringToBigInt(c.EditCandidate),
		SetHaltBlock:            helpers.StringToBigInt(c.SetHaltBlock),
		EditTickerOwner:         helpers.StringToBigInt(c.EditTickerOwner),
		EditMultisig:            helpers.StringToBigInt(c.EditMultisig),
		EditCandidatePublicKey:  helpers.StringToBigInt(c.EditCandidatePublicKey),
		CreateSwapPool:          helpers.StringToBigInt(c.CreateSwapPool),
		AddLiquidity:            helpers.StringToBigInt(c.AddLiquidity),
		RemoveLiquidity:         helpers.StringToBigInt(c.RemoveLiquidity),
		EditCandidateCommission: helpers.StringToBigInt(c.EditCandidateCommission),
		BurnToken:               helpers.StringToBigInt(c.BurnToken),
		MintToken:               helpers.StringToBigInt(c.MintToken),
		VoteCommission:          helpers.StringToBigInt(c.VoteCommission),
		VoteUpdate:              helpers.StringToBigInt(c.VoteUpdate),
		More:                    nil,
	}

	if c.FailedTx != "" &&
		c.AddLimitOrder != "" &&
		c.RemoveLimitOrder != "" {
		com.More = append(com.More,
			helpers.StringToBigInt(c.FailedTx),
			helpers.StringToBigInt(c.AddLimitOrder),
			helpers.StringToBigInt(c.RemoveLimitOrder))
	}

	s.Commission.SetNewCommissions(com.Encode())

	for _, vote := range state.CommissionVotes {
		vc := vote.Commission
		voteCom := &commission.Price{
			Coin:                    types.CoinID(vc.Coin),
			PayloadByte:             helpers.StringToBigInt(vc.PayloadByte),
			Send:                    helpers.StringToBigInt(vc.Send),
			BuyBancor:               helpers.StringToBigInt(vc.BuyBancor),
			SellBancor:              helpers.StringToBigInt(vc.SellBancor),
			SellAllBancor:           helpers.StringToBigInt(vc.SellAllBancor),
			BuyPoolBase:             helpers.StringToBigInt(vc.BuyPoolBase),
			BuyPoolDelta:            helpers.StringToBigInt(vc.BuyPoolDelta),
			SellPoolBase:            helpers.StringToBigInt(vc.SellPoolBase),
			SellPoolDelta:           helpers.StringToBigInt(vc.SellPoolDelta),
			SellAllPoolBase:         helpers.StringToBigInt(vc.SellAllPoolBase),
			SellAllPoolDelta:        helpers.StringToBigInt(vc.SellAllPoolDelta),
			CreateTicker3:           helpers.StringToBigInt(vc.CreateTicker3),
			CreateTicker4:           helpers.StringToBigInt(vc.CreateTicker4),
			CreateTicker5:           helpers.StringToBigInt(vc.CreateTicker5),
			CreateTicker6:           helpers.StringToBigInt(vc.CreateTicker6),
			CreateTicker7to10:       helpers.StringToBigInt(vc.CreateTicker7_10),
			CreateCoin:              helpers.StringToBigInt(vc.CreateCoin),
			CreateToken:             helpers.StringToBigInt(vc.CreateToken),
			RecreateCoin:            helpers.StringToBigInt(vc.RecreateCoin),
			RecreateToken:           helpers.StringToBigInt(vc.RecreateToken),
			DeclareCandidacy:        helpers.StringToBigInt(vc.DeclareCandidacy),
			Delegate:                helpers.StringToBigInt(vc.Delegate),
			Unbond:                  helpers.StringToBigInt(vc.Unbond),
			RedeemCheck:             helpers.StringToBigInt(vc.RedeemCheck),
			SetCandidateOn:          helpers.StringToBigInt(vc.SetCandidateOn),
			SetCandidateOff:         helpers.StringToBigInt(vc.SetCandidateOff),
			CreateMultisig:          helpers.StringToBigInt(vc.CreateMultisig),
			MultisendBase:           helpers.StringToBigInt(vc.MultisendBase),
			MultisendDelta:          helpers.StringToBigInt(vc.MultisendDelta),
			EditCandidate:           helpers.StringToBigInt(vc.EditCandidate),
			SetHaltBlock:            helpers.StringToBigInt(vc.SetHaltBlock),
			EditTickerOwner:         helpers.StringToBigInt(vc.EditTickerOwner),
			EditMultisig:            helpers.StringToBigInt(vc.EditMultisig),
			EditCandidatePublicKey:  helpers.StringToBigInt(vc.EditCandidatePublicKey),
			CreateSwapPool:          helpers.StringToBigInt(vc.CreateSwapPool),
			AddLiquidity:            helpers.StringToBigInt(vc.AddLiquidity),
			RemoveLiquidity:         helpers.StringToBigInt(vc.RemoveLiquidity),
			EditCandidateCommission: helpers.StringToBigInt(vc.EditCandidateCommission),
			BurnToken:               helpers.StringToBigInt(vc.BurnToken),
			MintToken:               helpers.StringToBigInt(vc.MintToken),
			VoteCommission:          helpers.StringToBigInt(vc.VoteCommission),
			VoteUpdate:              helpers.StringToBigInt(vc.VoteUpdate),
			More:                    nil,
		}
		if vc.FailedTx != "" &&
			vc.AddLimitOrder != "" &&
			vc.RemoveLimitOrder != "" {
			voteCom.More = append(com.More,
				helpers.StringToBigInt(vc.FailedTx),
				helpers.StringToBigInt(vc.AddLimitOrder),
				helpers.StringToBigInt(vc.RemoveLimitOrder))
		}
		for _, pubkey := range vote.Votes {
			s.Commission.AddVote(vote.Height, pubkey, voteCom.Encode())
		}
	}

	for _, vote := range state.UpdateVotes {
		for _, pubkey := range vote.Votes {
			s.Updates.AddVote(vote.Height, pubkey, vote.Version)
		}
	}

	return nil
}

func (s *State) Export() types.AppState {
	state, err := NewCheckStateAtHeightV3(uint64(s.tree.Version()), s.db)
	if err != nil {
		log.Panicf("Create new state at height %d failed: %s", s.tree.Version(), err)
	}

	return state.Export()
}

// Only for tests
func (s *State) ReloadFromDiskAndExport() types.AppState {
	state, err := NewCheckStateAtHeightV3(uint64(s.tree.Version()), s.db)
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

func newCheckStateForTreeV2(immutableTree *iavl.ImmutableTree, events eventsdb.IEventsDB, db db.DB, keepLastStates int64) (*CheckState, error) {
	stateForTree, err := newStateForTreeV2(immutableTree, events, db, keepLastStates)
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

	pool := swap.New(stateBus, immutableTree)

	commission := commission.NewCommission(immutableTree)

	update := update.New(immutableTree)

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
		Swap:        pool,
		Commission:  commission,
		Updates:     update,

		height:         immutableTree.Version(),
		bus:            stateBus,
		db:             db,
		events:         events,
		keepLastStates: keepLastStates,
	}

	return state, nil
}
func newStateForTreeV2(immutableTree *iavl.ImmutableTree, events eventsdb.IEventsDB, db db.DB, keepLastStates int64) (*State, error) {
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

	poolV2 := swap.NewV2(stateBus, immutableTree)

	commission := commission.NewCommission(immutableTree)

	update := update.New(immutableTree)

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
		SwapV2:      poolV2,
		Commission:  commission,
		Updates:     update,

		height:         immutableTree.Version(),
		bus:            stateBus,
		db:             db,
		events:         events,
		keepLastStates: keepLastStates,
	}

	return state, nil
}
