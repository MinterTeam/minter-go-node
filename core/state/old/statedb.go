package old

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/eventsdb/events"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/upgrades"
	tmTypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
	"math/big"
	"os"
	"sync"

	"bytes"
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/dao"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"sort"
)

const UnbondPeriod = 518400
const MaxDelegatorsPerCandidate = 1000

var (
	ValidatorMaxAbsentWindow = 24
	ValidatorMaxAbsentTimes  = 12

	addressPrefix     = []byte("a")
	coinPrefix        = []byte("c")
	frozenFundsPrefix = []byte("f")
	usedCheckPrefix   = []byte("u")
	candidatesKey     = []byte("t")
	validatorsKey     = []byte("v")
	maxGasKey         = []byte("g")
	totalSlashedKey   = []byte("s")
)

type StateDB struct {
	db   dbm.DB
	iavl Tree

	height uint64

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateAccounts      map[types.Address]*stateAccount
	stateAccountsDirty map[types.Address]struct{}

	stateCoins      map[types.CoinSymbol]*stateCoin
	stateCoinsDirty map[types.CoinSymbol]struct{}

	stateFrozenFunds      map[uint64]*stateFrozenFund
	stateFrozenFundsDirty map[uint64]struct{}

	stateCandidates      *stateCandidates
	stateCandidatesDirty bool

	stateValidators      *stateValidators
	stateValidatorsDirty bool

	totalSlashed      *big.Int
	totalSlashedDirty bool

	stakeCache map[types.CoinSymbol]StakeCache

	lock             sync.Mutex
	keepStateHistory bool
}

type StakeCache struct {
	TotalValue *big.Int
	BipValue   *big.Int
}

func NewForCheck(height uint64, db dbm.DB) (*StateDB, error) {
	tree := NewMutableTree(db)

	t, err := tree.GetImmutableAtHeight(int64(height))
	if err != nil {
		return nil, err
	}

	return &StateDB{
		db:                    db,
		iavl:                  t,
		height:                height,
		stateAccounts:         make(map[types.Address]*stateAccount),
		stateAccountsDirty:    make(map[types.Address]struct{}),
		stateCoins:            make(map[types.CoinSymbol]*stateCoin),
		stateCoinsDirty:       make(map[types.CoinSymbol]struct{}),
		stateFrozenFunds:      make(map[uint64]*stateFrozenFund),
		stateFrozenFundsDirty: make(map[uint64]struct{}),
		stateCandidates:       nil,
		stateCandidatesDirty:  false,
		stateValidators:       nil,
		stateValidatorsDirty:  false,
		totalSlashed:          nil,
		totalSlashedDirty:     false,
		stakeCache:            make(map[types.CoinSymbol]StakeCache),
	}, nil
}

func NewForCheckFromDeliver(s *StateDB) *StateDB {
	return &StateDB{
		db:                    s.db,
		iavl:                  s.iavl.GetImmutable(),
		height:                s.height,
		stateAccounts:         make(map[types.Address]*stateAccount),
		stateAccountsDirty:    make(map[types.Address]struct{}),
		stateCoins:            make(map[types.CoinSymbol]*stateCoin),
		stateCoinsDirty:       make(map[types.CoinSymbol]struct{}),
		stateFrozenFunds:      make(map[uint64]*stateFrozenFund),
		stateFrozenFundsDirty: make(map[uint64]struct{}),
		stateCandidates:       nil,
		stateCandidatesDirty:  false,
		stateValidators:       nil,
		stateValidatorsDirty:  false,
		totalSlashed:          nil,
		totalSlashedDirty:     false,
		stakeCache:            make(map[types.CoinSymbol]StakeCache),
	}
}

func New(height uint64, db dbm.DB, keepState bool) (*StateDB, error) {
	tree := NewMutableTree(db)

	_, err := tree.LazyLoadVersion(int64(height))

	if err != nil {
		return nil, err
	}

	return &StateDB{
		db:                    db,
		height:                height + 1,
		iavl:                  tree,
		stateAccounts:         make(map[types.Address]*stateAccount),
		stateAccountsDirty:    make(map[types.Address]struct{}),
		stateCoins:            make(map[types.CoinSymbol]*stateCoin),
		stateCoinsDirty:       make(map[types.CoinSymbol]struct{}),
		stateFrozenFunds:      make(map[uint64]*stateFrozenFund),
		stateFrozenFundsDirty: make(map[uint64]struct{}),
		stateCandidates:       nil,
		stateCandidatesDirty:  false,
		stateValidators:       nil,
		stateValidatorsDirty:  false,
		totalSlashed:          nil,
		totalSlashedDirty:     false,
		stakeCache:            make(map[types.CoinSymbol]StakeCache),
		keepStateHistory:      keepState,
	}, nil
}

func (s *StateDB) Clear() {
	s.stateAccounts = make(map[types.Address]*stateAccount)
	s.stateAccountsDirty = make(map[types.Address]struct{})
	s.stateCoins = make(map[types.CoinSymbol]*stateCoin)
	s.stateCoinsDirty = make(map[types.CoinSymbol]struct{})
	s.stateFrozenFunds = make(map[uint64]*stateFrozenFund)
	s.stateFrozenFundsDirty = make(map[uint64]struct{})
	s.stateCandidates = nil
	s.stateCandidatesDirty = false
	s.stateValidators = nil
	s.stateValidatorsDirty = false
	s.totalSlashed = nil
	s.totalSlashedDirty = false
	s.stakeCache = make(map[types.CoinSymbol]StakeCache)
	s.lock = sync.Mutex{}
}

func (s *StateDB) GetTotalSlashed() *big.Int {
	// Prefer 'live' object.
	if s.totalSlashed != nil {
		return s.totalSlashed
	}

	// Load the object from the database.
	_, enc := s.iavl.Get(totalSlashedKey)
	if len(enc) == 0 {
		return big.NewInt(0)
	}
	var data *big.Int
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode total slashed", "err", err)
		return nil
	}

	s.setTotalSlashed(data)
	return data
}

// Retrieve the balance from the given address or 0 if object not found
func (s *StateDB) GetBalance(addr types.Address, coinSymbol types.CoinSymbol) *big.Int {
	stateObject := s.getStateAccount(addr)
	if stateObject != nil {
		return stateObject.Balance(coinSymbol)
	}
	return types.Big0
}

func (s *StateDB) GetBalances(addr types.Address) Balances {
	stateObject := s.getStateAccount(addr)
	if stateObject != nil {
		return stateObject.Balances()
	}

	return Balances{
		Data: make(map[types.CoinSymbol]*big.Int),
	}
}

func (s *StateDB) GetNonce(addr types.Address) uint64 {
	stateObject := s.getStateAccount(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr
func (s *StateDB) AddBalance(addr types.Address, coinSymbol types.CoinSymbol, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(coinSymbol, amount)
	}
}

// SubBalance subtracts amount from the account associated with addr
func (s *StateDB) SubBalance(addr types.Address, coinSymbol types.CoinSymbol, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(coinSymbol, amount)
	}
}

func (s *StateDB) SetBalance(addr types.Address, coinSymbol types.CoinSymbol, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(coinSymbol, amount)
	}
}

func (s *StateDB) SetNonce(addr types.Address, nonce uint64) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

//
// Setting, updating & deleting state object methods
//

// updateStateObject writes the given object to the trie.
func (s *StateDB) updateStateObject(stateObject *stateAccount) {
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}

	s.iavl.Set(append(addressPrefix, addr[:]...), data)
}

func (s *StateDB) updateStateFrozenFund(stateFrozenFund *stateFrozenFund) {
	blockHeight := stateFrozenFund.BlockHeight()
	data, err := rlp.EncodeToBytes(stateFrozenFund)
	if err != nil {
		panic(fmt.Errorf("can't encode frozen fund at %d: %v", blockHeight, err))
	}
	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, uint64(stateFrozenFund.blockHeight))

	s.iavl.Set(append(frozenFundsPrefix, height...), data)
}

func (s *StateDB) updateStateCoin(stateCoin *stateCoin) {
	symbol := stateCoin.Symbol()

	data, err := rlp.EncodeToBytes(stateCoin)
	if err != nil {
		panic(fmt.Errorf("can't encode coin at %x: %v", symbol[:], err))
	}

	s.iavl.Set(append(coinPrefix, symbol[:]...), data)
}

func (s *StateDB) updateStateCandidates(stateCandidates *stateCandidates) {
	data, err := rlp.EncodeToBytes(stateCandidates)
	if err != nil {
		panic(fmt.Errorf("can't encode candidates: %v", err))
	}

	s.iavl.Set(candidatesKey, data)
}

func (s *StateDB) updateStateValidators(validators *stateValidators) {
	data, err := rlp.EncodeToBytes(validators)
	if err != nil {
		panic(fmt.Errorf("can't encode validators: %v", err))
	}

	s.iavl.Set(validatorsKey, data)
}

func (s *StateDB) updateTotalSlashed(value *big.Int) {
	data, err := rlp.EncodeToBytes(value)
	if err != nil {
		panic(fmt.Errorf("can't encode total slashed: %v", err))
	}

	s.iavl.Set(totalSlashedKey, data)
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteStateObject(stateObject *stateAccount) {
	stateObject.deleted = true
	addr := stateObject.Address()

	s.iavl.Remove(append(addressPrefix, addr[:]...))
}

// deleteStateCoin removes the given object from the state trie.
func (s *StateDB) deleteStateCoin(stateCoin *stateCoin) {
	symbol := stateCoin.Symbol()
	eventsdb.GetCurrent().AddEvent(s.height, events.CoinLiquidationEvent{
		Coin: symbol,
	})
	s.iavl.Remove(append(coinPrefix, symbol[:]...))
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteFrozenFunds(stateFrozenFund *stateFrozenFund) {
	stateFrozenFund.deleted = true
	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, uint64(stateFrozenFund.blockHeight))
	key := append(frozenFundsPrefix, height...)
	s.iavl.Remove(key)
}

// Retrieve a state frozen funds by block height. Returns nil if not found.
func (s *StateDB) getStateFrozenFunds(blockHeight uint64) (stateFrozenFund *stateFrozenFund) {
	// Prefer 'live' objects.
	if obj := s.stateFrozenFunds[blockHeight]; obj != nil {
		return obj
	}

	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, uint64(blockHeight))
	key := append(frozenFundsPrefix, height...)

	// Load the object from the database.
	_, enc := s.iavl.Get(key)
	if len(enc) == 0 {
		return nil
	}
	var data FrozenFunds
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		return nil
	}
	// Insert into the live set.
	obj := newFrozenFund(s, uint64(blockHeight), data, s.MarkStateFrozenFundsDirty)
	s.setStateFrozenFunds(obj)
	return obj
}

// Retrieve a state coin by symbol. Returns nil if not found.
func (s *StateDB) getStateCoin(symbol types.CoinSymbol) (stateCoin *stateCoin) {
	// Prefer 'live' objects.
	if obj := s.stateCoins[symbol]; obj != nil {
		return obj
	}

	// Load the object from the database.
	_, enc := s.iavl.Get(append(coinPrefix, symbol[:]...))
	if len(enc) == 0 {
		return nil
	}
	var data Coin
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state coin", "symbol", symbol, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newCoin(s, symbol, data, s.MarkStateCoinDirty)
	s.setStateCoin(obj)
	return obj
}

func (s *StateDB) GetStateCandidates() (stateCandidates *stateCandidates) {
	return s.getStateCandidates()
}

// Retrieve a state candidates. Returns nil if not found.
func (s *StateDB) getStateCandidates() (stateCandidates *stateCandidates) {
	// Prefer 'live' objects.
	if s.stateCandidates != nil {
		return s.stateCandidates
	}

	// Load the object from the database.
	_, enc := s.iavl.Get(candidatesKey)
	if len(enc) == 0 {
		return nil
	}
	var data Candidates
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		panic(err)
	}
	// Insert into the live set.
	obj := newCandidate(s, data, s.MarkStateCandidateDirty)
	s.setStateCandidates(obj)
	return obj
}

func (s *StateDB) GetStateValidators() (stateValidators *stateValidators) {
	return s.getStateValidators()
}

// Retrieve a state candidates. Returns nil if not found.
func (s *StateDB) getStateValidators() (stateValidators *stateValidators) {
	// Prefer 'live' objects.
	if s.stateValidators != nil {
		return s.stateValidators
	}

	// Load the object from the database.
	_, enc := s.iavl.Get(validatorsKey)
	if len(enc) == 0 {
		return nil
	}
	var data Validators
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		panic(err)
	}
	// Insert into the live set.
	obj := newValidator(s, data, s.MarkStateValidatorsDirty)
	s.setStateValidators(obj)
	return obj
}

// Retrieve a state account given my the address. Returns nil if not found.
func (s *StateDB) getStateAccount(addr types.Address) (stateObject *stateAccount) {
	// Prefer 'live' objects.
	if obj := s.stateAccounts[addr]; obj != nil {
		if obj.deleted {
			return nil
		}
		return obj
	}

	// Load the object from the database.
	_, enc := s.iavl.Get(append(addressPrefix, addr[:]...))
	if len(enc) == 0 {
		return nil
	}
	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newObject(addr, data, s.MarkStateObjectDirty)
	s.setStateObject(obj)
	return obj
}

func (s *StateDB) setStateObject(object *stateAccount) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateAccounts[object.Address()] = object
}

func (s *StateDB) setTotalSlashed(object *big.Int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.totalSlashed = object
}

func (s *StateDB) AddTotalSlashed(value *big.Int) {
	current := s.GetTotalSlashed()
	current.Add(current, value)

	s.setTotalSlashed(current)
	s.totalSlashedDirty = true
}

func (s *StateDB) setStateCoin(coin *stateCoin) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateCoins[coin.Symbol()] = coin
}

func (s *StateDB) setStateFrozenFunds(frozenFund *stateFrozenFund) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateFrozenFunds[frozenFund.BlockHeight()] = frozenFund
}

func (s *StateDB) setStateCandidates(candidates *stateCandidates) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateCandidates = candidates
}

func (s *StateDB) SetStateValidators(validators *stateValidators) {
	s.setStateValidators(validators)
}

func (s *StateDB) setStateValidators(validators *stateValidators) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateValidators = validators
}

// Retrieve a state object or create a new state object if nil
func (s *StateDB) GetOrNewStateObject(addr types.Address) *stateAccount {
	stateObject := s.getStateAccount(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = s.createAccount(addr)
	}
	return stateObject
}

func (s *StateDB) GetStateFrozenFunds(blockHeight uint64) *stateFrozenFund {
	return s.getStateFrozenFunds(blockHeight)
}

func (s *StateDB) GetOrNewStateFrozenFunds(blockHeight uint64) *stateFrozenFund {
	frozenFund := s.getStateFrozenFunds(blockHeight)
	if frozenFund == nil {
		frozenFund, _ = s.createFrozenFunds(blockHeight)
	}
	return frozenFund
}

// MarkStateObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (s *StateDB) MarkStateObjectDirty(addr types.Address) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateAccountsDirty[addr] = struct{}{}
}

func (s *StateDB) MarkStateCandidateDirty() {
	s.stateCandidatesDirty = true
}

func (s *StateDB) MarkStateValidatorsDirty() {
	s.stateValidatorsDirty = true
}

func (s *StateDB) MarkStateCoinDirty(symbol types.CoinSymbol) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateCoinsDirty[symbol] = struct{}{}
}

func (s *StateDB) MarkStateFrozenFundsDirty(blockHeight uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateFrozenFundsDirty[blockHeight] = struct{}{}
}

func (s *StateDB) createAccount(addr types.Address) (newobj, prev *stateAccount) {
	prev = s.getStateAccount(addr)
	newobj = newObject(addr, Account{}, s.MarkStateObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	s.setStateObject(newobj)
	return newobj, prev
}

func (s *StateDB) createMultisigAccount(addr types.Address, multisig Multisig) (newobj *stateAccount) {
	newobj = newObject(addr, Account{
		MultisigData: multisig,
	}, s.MarkStateObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	s.setStateObject(newobj)
	return newobj
}

func (s *StateDB) createFrozenFunds(blockHeight uint64) (newobj, prev *stateFrozenFund) {
	prev = s.getStateFrozenFunds(blockHeight)
	newobj = newFrozenFund(s, blockHeight, FrozenFunds{}, s.MarkStateFrozenFundsDirty)
	s.MarkStateFrozenFundsDirty(blockHeight)
	s.setStateFrozenFunds(newobj)
	return newobj, prev
}

func (s *StateDB) CreateCoin(
	symbol types.CoinSymbol,
	name string,
	volume *big.Int,
	crr uint,
	reserve *big.Int) *stateCoin {

	newC := newCoin(s, symbol, Coin{
		Name:           name,
		Symbol:         symbol,
		Volume:         volume,
		Crr:            crr,
		ReserveBalance: reserve,
	}, s.MarkStateCoinDirty)
	s.setStateCoin(newC)
	return newC
}

func (s *StateDB) CreateValidator(
	rewardAddress types.Address,
	pubkey types.Pubkey,
	commission uint,
	currentBlock uint,
	coin types.CoinSymbol,
	initialStake *big.Int) *stateValidators {

	vals := s.getStateValidators()

	if vals == nil {
		vals = newValidator(s, Validators{}, s.MarkStateValidatorsDirty)
	}

	vals.data = append(vals.data, Validator{
		RewardAddress: rewardAddress,
		TotalBipStake: initialStake,
		PubKey:        pubkey,
		Commission:    commission,
		AccumReward:   big.NewInt(0),
		AbsentTimes:   types.NewBitArray(ValidatorMaxAbsentWindow),
	})

	s.MarkStateValidatorsDirty()
	s.setStateValidators(vals)
	return vals
}

func (s *StateDB) CreateCandidate(
	rewardAddress types.Address,
	ownerAddress types.Address,
	pubkey types.Pubkey,
	commission uint,
	currentBlock uint,
	coin types.CoinSymbol,
	initialStake *big.Int) *stateCandidates {

	candidates := s.getStateCandidates()

	if candidates == nil {
		candidates = newCandidate(s, Candidates{}, s.MarkStateCandidateDirty)
	}

	candidate := Candidate{
		RewardAddress: rewardAddress,
		OwnerAddress:  ownerAddress,
		PubKey:        pubkey,
		Commission:    commission,
		Stakes: []Stake{
			{
				Owner:    rewardAddress,
				Coin:     coin,
				Value:    initialStake,
				BipValue: big.NewInt(0),
			},
		},
		CreatedAtBlock: currentBlock,
		Status:         CandidateStatusOffline,
	}

	candidate.Stakes[0].BipValue = candidate.Stakes[0].CalcBipValue(s)

	candidates.data = append(candidates.data, candidate)

	s.MarkStateCandidateDirty()
	s.setStateCandidates(candidates)
	return candidates
}

// Commit writes the state to the underlying in-memory trie database.
func (s *StateDB) Commit() (root []byte, version int64, err error) {
	// Commit objects to the trie.
	for _, addr := range getOrderedObjectsKeys(s.stateAccountsDirty) {
		stateObject := s.stateAccounts[addr]
		if stateObject.empty() {
			s.deleteStateObject(stateObject)
		} else {
			s.updateStateObject(stateObject)
		}
		delete(s.stateAccountsDirty, addr)
	}

	// Commit coins to the trie.
	for _, symbol := range getOrderedCoinsKeys(s.stateCoinsDirty) {
		stateCoin := s.stateCoins[symbol]

		if stateCoin.data.Volume.Cmp(types.Big0) == 0 {
			s.deleteStateCoin(stateCoin)
		} else {
			s.updateStateCoin(stateCoin)
		}

		delete(s.stateCoinsDirty, symbol)
	}

	// Commit frozen funds to the trie.
	for _, block := range getOrderedFrozenFundsKeys(s.stateFrozenFundsDirty) {
		frozenFund := s.stateFrozenFunds[block]
		if frozenFund.deleted {
			s.deleteFrozenFunds(frozenFund)
		} else {
			s.updateStateFrozenFund(frozenFund)
		}

		delete(s.stateFrozenFundsDirty, block)
	}

	if s.stateCandidatesDirty {
		s.clearStateCandidates()
		s.updateStateCandidates(s.stateCandidates)
		s.stateCandidatesDirty = false
	}

	if s.stateValidatorsDirty {
		s.updateStateValidators(s.stateValidators)
		s.stateValidatorsDirty = false
	}

	if s.totalSlashedDirty {
		s.updateTotalSlashed(s.totalSlashed)
		s.totalSlashedDirty = false
	}

	hash, version, err := s.iavl.SaveVersion()

	if !s.keepStateHistory && version > 1 {
		err = s.iavl.DeleteVersion(version - 1)

		if err != nil {
			panic(err)
		}
	}

	s.Clear()
	s.height++

	return hash, version, err
}

func getOrderedObjectsKeys(objects map[types.Address]struct{}) []types.Address {
	keys := make([]types.Address, 0, len(objects))
	for k := range objects {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func getOrderedCoinsKeys(objects map[types.CoinSymbol]struct{}) []types.CoinSymbol {
	keys := make([]types.CoinSymbol, 0, len(objects))
	for k := range objects {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func getOrderedFrozenFundsKeys(objects map[uint64]struct{}) []uint64 {
	keys := make([]uint64, 0, len(objects))
	for k := range objects {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	return keys
}

func (s *StateDB) CoinExists(symbol types.CoinSymbol) bool {
	if symbol == types.GetBaseCoin() {
		return true
	}

	stateCoin := s.getStateCoin(symbol)

	return stateCoin != nil && !stateCoin.isDeleted
}

func (s *StateDB) CandidateExists(key types.Pubkey) bool {

	stateCandidates := s.getStateCandidates()

	if stateCandidates == nil {
		return false
	}

	for _, candidate := range stateCandidates.data {
		if bytes.Equal(candidate.PubKey, key) {
			return true
		}
	}

	return false
}

func (s *StateDB) GetStateCandidate(key types.Pubkey) *Candidate {
	stateCandidates := s.getStateCandidates()
	if stateCandidates == nil {
		return nil
	}

	for i, candidate := range stateCandidates.data {
		if bytes.Equal(candidate.PubKey, key) {
			return &(stateCandidates.data[i])
		}
	}

	return nil
}

func (s *StateDB) GetStateCandidateByTmAddress(address [20]byte) *Candidate {
	stateCandidates := s.getStateCandidates()
	if stateCandidates == nil {
		return nil
	}

	for i, candidate := range stateCandidates.data {
		if candidate.GetAddress() == address {
			return &(stateCandidates.data[i])
		}
	}

	return nil
}

func (s *StateDB) GetStateCoin(symbol types.CoinSymbol) *stateCoin {
	return s.getStateCoin(symbol)
}

func (s *StateDB) AddCoinVolume(symbol types.CoinSymbol, value *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}

	stateCoin := s.GetStateCoin(symbol)
	if stateCoin != nil {
		stateCoin.AddVolume(value)
	}
}

func (s *StateDB) SubCoinVolume(symbol types.CoinSymbol, value *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}

	stateCoin := s.GetStateCoin(symbol)
	if stateCoin != nil {
		stateCoin.SubVolume(value)
	}
}

func (s *StateDB) AddCoinReserve(symbol types.CoinSymbol, value *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}

	stateCoin := s.GetStateCoin(symbol)
	if stateCoin != nil {
		stateCoin.AddReserve(value)
	}
}

func (s *StateDB) SubCoinReserve(symbol types.CoinSymbol, value *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}

	stateCoin := s.GetStateCoin(symbol)
	if stateCoin != nil {
		stateCoin.SubReserve(value)
	}
}

func (s *StateDB) GetCandidates(count int, block int64) []Candidate {
	stateCandidates := s.getStateCandidates()

	if stateCandidates == nil {
		return nil
	}

	candidates := stateCandidates.data

	var activeCandidates Candidates

	// get only active candidates
	for _, v := range candidates {
		if v.Status == CandidateStatusOnline && len(v.PubKey) == 32 {
			activeCandidates = append(activeCandidates, v)
		}
	}

	sort.SliceStable(activeCandidates, func(i, j int) bool {
		return activeCandidates[i].TotalBipStake.Cmp(candidates[j].TotalBipStake) == -1
	})

	if len(activeCandidates) < count {
		count = len(activeCandidates)
	}

	sort.SliceStable(activeCandidates, func(i, j int) bool {
		return activeCandidates[i].TotalBipStake.Cmp(activeCandidates[j].TotalBipStake) == 1
	})

	return activeCandidates[:count]
}

func (s *StateDB) AddAccumReward(pubkey types.Pubkey, reward *big.Int) {
	vals := s.getStateValidators()
	for i := range vals.data {
		if bytes.Equal(vals.data[i].PubKey, pubkey) {
			vals.data[i].AccumReward.Add(vals.data[i].AccumReward, reward)
			s.setStateValidators(vals)
			s.MarkStateValidatorsDirty()
			return
		}
	}
}

func (s *StateDB) PayRewards() {
	edb := eventsdb.GetCurrent()

	vals := s.getStateValidators()
	for i := range vals.data {
		validator := vals.data[i]

		if validator.AccumReward.Cmp(types.Big0) == 1 {
			totalReward := big.NewInt(0).Set(validator.AccumReward)
			remainder := big.NewInt(0).Set(validator.AccumReward)

			// pay commission to DAO
			DAOReward := big.NewInt(0).Set(totalReward)
			DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
			DAOReward.Div(DAOReward, big.NewInt(100))
			s.AddBalance(dao.Address, types.GetBaseCoin(), DAOReward)
			remainder.Sub(remainder, DAOReward)
			edb.AddEvent(s.height, events.RewardEvent{
				Role:            events.RoleDAO,
				Address:         dao.Address,
				Amount:          DAOReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			// pay commission to Developers
			DevelopersReward := big.NewInt(0).Set(totalReward)
			DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
			DevelopersReward.Div(DevelopersReward, big.NewInt(100))
			s.AddBalance(developers.Address, types.GetBaseCoin(), DevelopersReward)
			remainder.Sub(remainder, DevelopersReward)
			edb.AddEvent(s.height, events.RewardEvent{
				Role:            events.RoleDevelopers,
				Address:         developers.Address,
				Amount:          DevelopersReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			totalReward.Sub(totalReward, DevelopersReward)
			totalReward.Sub(totalReward, DAOReward)

			// pay commission to validator
			validatorReward := big.NewInt(0).Set(totalReward)
			validatorReward.Mul(validatorReward, big.NewInt(int64(validator.Commission)))
			validatorReward.Div(validatorReward, big.NewInt(100))
			totalReward.Sub(totalReward, validatorReward)
			s.AddBalance(validator.RewardAddress, types.GetBaseCoin(), validatorReward)
			remainder.Sub(remainder, validatorReward)
			edb.AddEvent(s.height, events.RewardEvent{
				Role:            events.RoleValidator,
				Address:         validator.RewardAddress,
				Amount:          validatorReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			candidate := s.GetStateCandidate(validator.PubKey)

			// pay rewards
			for j := range candidate.Stakes {
				stake := candidate.Stakes[j]

				if stake.BipValue.Cmp(big.NewInt(0)) == 0 {
					continue
				}

				reward := big.NewInt(0).Set(totalReward)
				reward.Mul(reward, stake.BipValue)

				reward.Div(reward, validator.TotalBipStake)

				if reward.Cmp(types.Big0) < 1 {
					continue
				}

				s.AddBalance(stake.Owner, types.GetBaseCoin(), reward)
				remainder.Sub(remainder, reward)

				edb.AddEvent(s.height, events.RewardEvent{
					Role:            events.RoleDelegator,
					Address:         stake.Owner,
					Amount:          reward.Bytes(),
					ValidatorPubKey: candidate.PubKey,
				})
			}

			vals.data[i].AccumReward = big.NewInt(0)

			if remainder.Cmp(big.NewInt(0)) > -1 {
				s.AddTotalSlashed(remainder)
			} else {
				log.With("module", "main").Error("Negative remainder", "remainder", remainder.String())
			}
		}
	}

	s.setStateValidators(vals)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) RecalculateTotalStakeValues() {
	stateCandidates := s.getStateCandidates()
	vals := s.getStateValidators()
	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]

		totalBipStake := big.NewInt(0)

		for j := range candidate.Stakes {
			stake := &candidate.Stakes[j]
			stake.BipValue = stake.CalcBipValue(s)
			totalBipStake.Add(totalBipStake, stake.BipValue)
		}

		candidate.TotalBipStake = totalBipStake

		for j := range vals.data {
			if bytes.Equal(vals.data[j].PubKey, candidate.PubKey) {
				vals.data[j].TotalBipStake = totalBipStake
				break
			}
		}
	}

	s.setStateValidators(vals)
	s.MarkStateValidatorsDirty()

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) Delegate(sender types.Address, pubkey []byte, coin types.CoinSymbol, value *big.Int) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if candidate.PubKey.Compare(pubkey) == 0 {
			exists := false

			for j := range candidate.Stakes {
				stake := &candidate.Stakes[j]
				if sender.Compare(stake.Owner) == 0 && stake.Coin.Compare(coin) == 0 {
					stake.Value.Add(stake.Value, value)
					exists = true
					break
				}
			}

			if !exists {
				candidate.Stakes = append(candidate.Stakes, Stake{
					Owner:    sender,
					Coin:     coin,
					Value:    value,
					BipValue: big.NewInt(0),
				})
			}
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) SubStake(sender types.Address, pubkey []byte, coin types.CoinSymbol, value *big.Int) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if candidate.PubKey.Compare(pubkey) == 0 {
			currentStakeValue := candidate.GetStakeOfAddress(sender, coin).Value
			currentStakeValue.Sub(currentStakeValue, value)
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) IsCheckUsed(check *check.Check) bool {
	checkHash := check.Hash().Bytes()
	_, data := s.iavl.Get(append(usedCheckPrefix, checkHash...))

	return len(data) != 0
}

func (s *StateDB) UseCheck(check *check.Check) {
	checkHash := check.Hash().Bytes()
	s.useCheckHash(checkHash)
}

func (s *StateDB) useCheckHash(hash []byte) {
	trieHash := append(usedCheckPrefix, hash...)

	s.iavl.Set(trieHash, []byte{0x1})
}

func (s *StateDB) EditCandidate(pubkey []byte, newRewardAddress types.Address, newOwnerAddress types.Address) {
	stateCandidates := s.getStateCandidates()
	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Equal(candidate.PubKey, pubkey) {
			candidate.RewardAddress = newRewardAddress
			candidate.OwnerAddress = newOwnerAddress
			break
		}
	}
	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()

	vals := s.getStateValidators()
	for i := range vals.data {
		validator := &vals.data[i]
		if bytes.Equal(validator.PubKey, pubkey) {
			validator.RewardAddress = newRewardAddress
			break
		}
	}
	s.setStateValidators(vals)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) SetCandidateOnline(pubkey []byte) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Equal(candidate.PubKey, pubkey) {
			candidate.Status = CandidateStatusOnline
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) SetCandidateOffline(pubkey []byte) {
	stateCandidates := s.getStateCandidates()
	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Equal(candidate.PubKey, pubkey) {
			candidate.Status = CandidateStatusOffline
		}
	}
	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()

	vals := s.getStateValidators()
	for i := range vals.data {
		validator := &vals.data[i]
		if bytes.Equal(validator.PubKey, pubkey) {
			validator.toDrop = true
		}
	}
	s.setStateValidators(vals)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) SetValidatorPresent(address [20]byte) {
	vals := s.getStateValidators()
	for i := range vals.data {
		validator := &vals.data[i]
		if validator.GetAddress() == address {
			validator.AbsentTimes.SetIndex(int(s.height)%ValidatorMaxAbsentWindow, false)
		}
	}
	s.setStateValidators(vals)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) SetValidatorAbsent(address [20]byte) {
	edb := eventsdb.GetCurrent()

	vals := s.getStateValidators()
	for i := range vals.data {
		validator := &vals.data[i]
		if validator.GetAddress() == address {

			candidates := s.getStateCandidates()

			var candidate *Candidate
			for i := range candidates.data {
				if candidates.data[i].GetAddress() == address {
					candidate = &candidates.data[i]
				}
			}

			if candidate == nil {
				log.Error("Candidate not found", "address", fmt.Sprintf("%x", address))
				return
			}

			if candidate.Status == CandidateStatusOffline {
				return
			}

			validator.AbsentTimes.SetIndex(int(s.height)%ValidatorMaxAbsentWindow, true)

			if validator.CountAbsentTimes() > ValidatorMaxAbsentTimes {
				candidate.Status = CandidateStatusOffline
				validator.AbsentTimes = types.NewBitArray(ValidatorMaxAbsentWindow)
				validator.toDrop = true

				totalStake := big.NewInt(0)

				for j, stake := range candidate.Stakes {
					newValue := big.NewInt(0).Set(stake.Value)
					newValue.Mul(newValue, big.NewInt(99))
					newValue.Div(newValue, big.NewInt(100))

					slashed := big.NewInt(0).Set(stake.Value)
					slashed.Sub(slashed, newValue)

					if !stake.Coin.IsBaseCoin() {
						coin := s.GetStateCoin(stake.Coin).Data()
						ret := formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, slashed)

						s.SubCoinVolume(coin.Symbol, slashed)
						s.SubCoinReserve(coin.Symbol, ret)
						s.SanitizeCoin(stake.Coin)

						s.AddTotalSlashed(ret)
					} else {
						s.AddTotalSlashed(slashed)
					}

					edb.AddEvent(s.height, events.SlashEvent{
						Address:         stake.Owner,
						Amount:          slashed.Bytes(),
						Coin:            stake.Coin,
						ValidatorPubKey: candidate.PubKey,
					})

					candidate.Stakes[j] = Stake{
						Owner:    stake.Owner,
						Coin:     stake.Coin,
						Value:    newValue,
						BipValue: big.NewInt(0),
					}
					totalStake.Add(totalStake, newValue)
				}

				validator.TotalBipStake = totalStake
			}

			s.setStateCandidates(candidates)
			s.MarkStateCandidateDirty()
		}
	}

	s.setStateValidators(vals)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) PunishByzantineValidator(address [20]byte) {
	edb := eventsdb.GetCurrent()
	vals := s.getStateValidators()

	for i := range vals.data {
		validator := &vals.data[i]
		if validator.GetAddress() == address {
			candidates := s.getStateCandidates()

			var candidate *Candidate
			for i := range candidates.data {
				if candidates.data[i].GetAddress() == address {
					candidate = &candidates.data[i]
				}
			}

			if candidate == nil {
				return
			}

			for _, stake := range candidate.Stakes {
				newValue := big.NewInt(0).Set(stake.Value)
				newValue.Mul(newValue, big.NewInt(95))
				newValue.Div(newValue, big.NewInt(100))

				slashed := big.NewInt(0).Set(stake.Value)
				slashed.Sub(slashed, newValue)

				if !stake.Coin.IsBaseCoin() {
					coin := s.GetStateCoin(stake.Coin).Data()
					ret := formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, slashed)

					s.SubCoinVolume(coin.Symbol, slashed)
					s.SubCoinReserve(coin.Symbol, ret)

					s.AddTotalSlashed(ret)
				} else {
					s.AddTotalSlashed(slashed)
				}

				edb.AddEvent(s.height, events.SlashEvent{
					Address:         stake.Owner,
					Amount:          slashed.Bytes(),
					Coin:            stake.Coin,
					ValidatorPubKey: candidate.PubKey,
				})

				s.GetOrNewStateFrozenFunds(s.height+UnbondPeriod).AddFund(stake.Owner, candidate.PubKey,
					stake.Coin, newValue)
				s.SanitizeCoin(stake.Coin)
			}

			candidate.Stakes = []Stake{}
			candidate.Status = CandidateStatusOffline
			validator.TotalBipStake = big.NewInt(0)
			validator.toDrop = true

			s.setStateCandidates(candidates)
			s.MarkStateCandidateDirty()
		}
	}

	s.setStateValidators(vals)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) PunishFrozenFundsWithAddress(fromBlock uint64, toBlock uint64, address [20]byte) {
	for i := fromBlock; i <= toBlock; i++ {
		frozenFund := s.getStateFrozenFunds(i)

		if frozenFund == nil {
			continue
		}

		frozenFund.PunishFund(s, address, fromBlock)
	}
}

func (s *StateDB) SetNewValidators(candidates []Candidate) {
	oldVals := s.getStateValidators()

	var newVals Validators

	for _, candidate := range candidates {
		accumReward := big.NewInt(0)
		absentTimes := types.NewBitArray(ValidatorMaxAbsentWindow)

		for _, oldVal := range oldVals.data {
			if oldVal.GetAddress() == candidate.GetAddress() {
				accumReward = oldVal.AccumReward
				absentTimes = oldVal.AbsentTimes
			}
		}

		newVals = append(newVals, Validator{
			RewardAddress: candidate.RewardAddress,
			TotalBipStake: candidate.TotalBipStake,
			PubKey:        candidate.PubKey,
			Commission:    candidate.Commission,
			AccumReward:   accumReward,
			AbsentTimes:   absentTimes,
		})
	}

	oldVals.data = newVals
	s.setStateValidators(oldVals)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) RemoveCurrentValidator(pubkey types.Pubkey) {
	oldVals := s.getStateValidators()

	var newVals Validators

	for _, val := range oldVals.data {
		if val.PubKey.Compare(pubkey) == 0 {
			continue
		}

		newVals = append(newVals, val)
	}

	oldVals.data = newVals
	s.setStateValidators(oldVals)
	s.MarkStateValidatorsDirty()
}

// remove 0-valued stakes
func (s *StateDB) clearStateCandidates() {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]

		var newStakes []Stake
		for _, stake := range candidate.Stakes {
			if stake.Value.Cmp(types.Big0) == 0 {
				continue
			}
			newStakes = append(newStakes, stake)
		}

		candidate.Stakes = newStakes
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) CreateMultisig(weights []uint, addresses []types.Address, threshold uint) types.Address {
	msig := Multisig{
		Weights:   weights,
		Threshold: threshold,
		Addresses: addresses,
	}

	msigAddress := msig.Address()
	s.createMultisigAccount(msigAddress, msig).touch()

	return msigAddress
}

func (s *StateDB) AccountExists(address types.Address) bool {
	return s.getStateAccount(address) != nil
}

func (s *StateDB) MultisigAccountExists(address types.Address) bool {
	acc := s.getStateAccount(address)

	return acc != nil && acc.IsMultisig()
}

func (s *StateDB) IsNewCandidateStakeSufficient(coinSymbol types.CoinSymbol, stake *big.Int) bool {
	bipValue := (&Stake{
		Coin:     coinSymbol,
		Value:    stake,
		BipValue: big.NewInt(0),
	}).CalcBipValue(s)

	candidates := s.getStateCandidates()

	for _, candidate := range candidates.data {
		if candidate.TotalBipStake.Cmp(bipValue) == -1 {
			return true
		}
	}

	return false
}

func (s *StateDB) CandidatesCount() int {
	candidates := s.getStateCandidates()

	if candidates == nil {
		return 0
	}

	return len(candidates.data)
}

func (s *StateDB) ClearCandidates() {
	maxCandidates := validators.GetCandidatesCountForBlock(s.height)

	candidates := s.getStateCandidates()

	// Check for candidates count overflow and unbond smallest ones
	if len(candidates.data) > maxCandidates {
		sort.SliceStable(candidates.data, func(i, j int) bool {
			return candidates.data[i].TotalBipStake.Cmp(candidates.data[j].TotalBipStake) == 1
		})

		dropped := candidates.data[maxCandidates:]
		candidates.data = candidates.data[:maxCandidates]

		unbondAtBlock := s.height + UnbondPeriod
		for _, candidate := range dropped {
			for _, stake := range candidate.Stakes {
				s.GetOrNewStateFrozenFunds(unbondAtBlock).AddFund(stake.Owner, candidate.PubKey, stake.Coin, stake.Value)
			}
		}
	}

	s.setStateCandidates(candidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) ClearStakes() {
	candidates := s.getStateCandidates()

	for i := range candidates.data {
		candidate := &candidates.data[i]
		// Check for delegators count overflow and unbond smallest ones
		if len(candidate.Stakes) > MaxDelegatorsPerCandidate {
			sort.SliceStable(candidate.Stakes, func(t, d int) bool {
				return candidates.data[i].Stakes[t].BipValue.Cmp(candidates.data[i].Stakes[d].BipValue) == 1
			})

			dropped := candidates.data[i].Stakes[MaxDelegatorsPerCandidate:]
			candidates.data[i].Stakes = candidates.data[i].Stakes[:MaxDelegatorsPerCandidate]

			for _, stake := range dropped {
				eventsdb.GetCurrent().AddEvent(s.height, events.UnbondEvent{
					Address:         stake.Owner,
					Amount:          stake.Value.Bytes(),
					Coin:            stake.Coin,
					ValidatorPubKey: candidate.PubKey,
				})
				s.AddBalance(stake.Owner, stake.Coin, stake.Value)
			}
		}
	}

	s.setStateCandidates(candidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) IsDelegatorStakeSufficient(sender types.Address, pubKey []byte, coinSymbol types.CoinSymbol,
	value *big.Int) bool {
	if s.StakeExists(sender, pubKey, coinSymbol) {
		return true
	}

	stake := Stake{
		Coin:     coinSymbol,
		Value:    value,
		BipValue: big.NewInt(0),
	}

	bipValue := big.NewInt(0)
	if s.Height() > upgrades.UpgradeBlock1 {
		bipValue = stake.CalcSimulatedBipValue(s)
	} else {
		bipValue = stake.CalcBipValue(s)
	}

	candidates := s.getStateCandidates()

	for _, candidate := range candidates.data {
		if bytes.Equal(candidate.PubKey, pubKey) {
			for _, stake := range candidate.Stakes[:MaxDelegatorsPerCandidate] {
				if stake.BipValue.Cmp(bipValue) == -1 {
					return true
				}
			}

			return false
		}
	}

	return false
}

func (s *StateDB) StakeExists(owner types.Address, pubKey []byte, coinSymbol types.CoinSymbol) bool {
	candidates := s.getStateCandidates().data

	for _, c := range candidates {
		if !bytes.Equal(pubKey, c.PubKey) {
			continue
		}

		for _, s := range c.Stakes {
			if s.Owner == owner && s.Coin == coinSymbol {
				return true
			}
		}
	}

	return false
}

func (s *StateDB) GetCurrentMaxGas() uint64 {
	_, maxGasBytes := s.iavl.Get(maxGasKey)

	return binary.BigEndian.Uint64(maxGasBytes)
}

func (s *StateDB) SetMaxGas(maxGas uint64) {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, maxGas)

	s.iavl.Set(maxGasKey, bs)
}

func (s *StateDB) GetMaxGas() uint64 {
	_, b := s.iavl.Get(maxGasKey)

	return binary.BigEndian.Uint64(b)
}

func (s *StateDB) SanitizeCoin(symbol types.CoinSymbol) {
	if symbol.IsBaseCoin() {
		return
	}

	coin := s.GetStateCoin(symbol)
	if coin.IsToDelete() {
		s.deleteCoin(coin.symbol)
	}
}

func (s *StateDB) deleteCoin(symbol types.CoinSymbol) {
	coinToDelete := s.getStateCoin(symbol)
	if coinToDelete.isDeleted {
		return
	}
	coinToDelete.isDeleted = true

	var addresses []types.Address
	for _, account := range s.stateAccounts {
		addresses = append(addresses, account.address)
	}

	sort.SliceStable(addresses, func(i, j int) bool {
		return addresses[i].Compare(addresses[j]) == -1
	})

	var frozenFundsHeights []uint64
	for height := range s.stateFrozenFunds {
		frozenFundsHeights = append(frozenFundsHeights, height)
	}

	sort.SliceStable(frozenFundsHeights, func(i, j int) bool {
		return frozenFundsHeights[i] < frozenFundsHeights[j]
	})

	s.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == addressPrefix[0] {
			account := s.GetOrNewStateObject(types.BytesToAddress(key[1:]))
			for _, coin := range account.Balances().getCoins() {
				if coin == symbol {
					addresses = append(addresses, account.address)
				}
			}
		}

		if key[0] == frozenFundsPrefix[0] {
			frozenFunds := s.GetStateFrozenFunds(binary.BigEndian.Uint64(key[1:]))
			for _, ff := range frozenFunds.data.List {
				if ff.Coin == symbol {
					frozenFundsHeights = append(frozenFundsHeights, frozenFunds.BlockHeight())
				}
			}
		}

		return false
	})

	// remove coin from accounts
	for _, address := range addresses {
		account := s.getStateAccount(address)
		for _, coin := range account.Balances().getCoins() {
			if coin == symbol {
				amount := account.Balance(symbol)
				ret := formula.CalculateSaleReturn(coinToDelete.Volume(), coinToDelete.ReserveBalance(), 100, amount)

				coinToDelete.SubReserve(ret)
				coinToDelete.SubVolume(amount)

				account.AddBalance(types.GetBaseCoin(), ret)
				account.SetBalance(symbol, big.NewInt(0))
				s.MarkStateObjectDirty(account.address)
			}
		}
	}

	// remove coin from frozen funds
	for _, height := range frozenFundsHeights {
		frozenFunds := s.GetStateFrozenFunds(height)
		if frozenFunds != nil {
			var newFrozenFundsList []FrozenFund

			for _, ff := range frozenFunds.data.List {
				if ff.Coin == symbol {
					ret := formula.CalculateSaleReturn(coinToDelete.Volume(), coinToDelete.ReserveBalance(), 100, ff.Value)

					coinToDelete.SubReserve(ret)
					coinToDelete.SubVolume(ff.Value)

					newFrozenFundsList = append(newFrozenFundsList, FrozenFund{
						Address:      ff.Address,
						CandidateKey: ff.CandidateKey,
						Coin:         types.GetBaseCoin(),
						Value:        ret,
					})

					continue
				}

				newFrozenFundsList = append(newFrozenFundsList, ff)
			}

			frozenFunds.data.List = newFrozenFundsList
			s.setStateFrozenFunds(frozenFunds)
			s.MarkStateFrozenFundsDirty(frozenFunds.blockHeight)
		}
	}

	// remove coin from stakes
	candidates := s.getStateCandidates()
	if candidates != nil {
		for i := range candidates.data {
			candidate := &candidates.data[i]
			for j, stake := range candidate.Stakes {
				if stake.Coin == symbol {
					ret := formula.CalculateSaleReturn(coinToDelete.Volume(), coinToDelete.ReserveBalance(), 100, stake.Value)

					coinToDelete.SubReserve(ret)
					coinToDelete.SubVolume(stake.Value)

					stake := candidate.GetStakeOfAddress(stake.Owner, types.GetBaseCoin())
					if stake == nil {
						candidate.Stakes[j].Value = ret
						candidate.Stakes[j].Coin = types.GetBaseCoin()
					} else {
						candidate.Stakes[j].Value = big.NewInt(0)
						stake.Value.Add(stake.Value, ret)
					}
				}
			}
		}
		s.setStateCandidates(candidates)
		s.MarkStateCandidateDirty()
	}

	// set coin volume to 0
	s.SubCoinVolume(symbol, s.GetStateCoin(symbol).Volume())
	s.MarkStateCoinDirty(symbol)
}

func (s *StateDB) Export(currentHeight uint64) types.AppState {
	appState := types.AppState{}

	s.iavl.Iterate(func(key []byte, value []byte) bool {
		// export accounts
		if key[0] == addressPrefix[0] {
			account := s.GetOrNewStateObject(types.BytesToAddress(key[1:]))

			balance := make([]types.Balance, len(account.Balances().Data))
			i := 0
			for coin, value := range account.Balances().Data {
				balance[i] = types.Balance{
					Coin:  coin,
					Value: value,
				}
				i++
			}

			acc := types.Account{
				Address: account.address,
				Balance: balance,
				Nonce:   account.data.Nonce,
			}

			if account.IsMultisig() {
				acc.MultisigData = &types.Multisig{
					Weights:   account.data.MultisigData.Weights,
					Threshold: account.data.MultisigData.Threshold,
					Addresses: account.data.MultisigData.Addresses,
				}
			}

			appState.Accounts = append(appState.Accounts, acc)
		}

		// export coins
		if key[0] == coinPrefix[0] {
			coin := s.GetStateCoin(types.StrToCoinSymbol(string(key[1:])))

			appState.Coins = append(appState.Coins, types.Coin{
				Name:           coin.Name(),
				Symbol:         coin.Symbol(),
				Volume:         coin.Volume(),
				Crr:            coin.Crr(),
				ReserveBalance: coin.ReserveBalance(),
			})
		}

		// export used checks
		if key[0] == usedCheckPrefix[0] {
			appState.UsedChecks = append(appState.UsedChecks, types.UsedCheck(fmt.Sprintf("%x", key[1:])))
		}

		// export frozen funds
		if key[0] == frozenFundsPrefix[0] {
			height := binary.BigEndian.Uint64(key[1:])
			frozenFunds := s.GetStateFrozenFunds(height)

			for _, frozenFund := range frozenFunds.List() {
				appState.FrozenFunds = append(appState.FrozenFunds, types.FrozenFund{
					Height:       height - uint64(currentHeight),
					Address:      frozenFund.Address,
					CandidateKey: frozenFund.CandidateKey,
					Coin:         frozenFund.Coin,
					Value:        frozenFund.Value,
				})
			}
		}

		return false
	})

	candidates := s.getStateCandidates()
	for _, candidate := range candidates.data {
		var stakes []types.Stake
		for _, s := range candidate.Stakes {
			stakes = append(stakes, types.Stake{
				Owner:    s.Owner,
				Coin:     s.Coin,
				Value:    s.Value,
				BipValue: s.BipValue,
			})
		}

		appState.Candidates = append(appState.Candidates, types.Candidate{
			RewardAddress:  candidate.RewardAddress,
			OwnerAddress:   candidate.OwnerAddress,
			TotalBipStake:  candidate.TotalBipStake,
			PubKey:         candidate.PubKey,
			Commission:     candidate.Commission,
			Stakes:         stakes,
			CreatedAtBlock: candidate.CreatedAtBlock,
			Status:         candidate.Status,
		})
	}

	vals := s.getStateValidators()
	for _, val := range vals.data {
		appState.Validators = append(appState.Validators, types.Validator{
			RewardAddress: val.RewardAddress,
			TotalBipStake: val.TotalBipStake,
			PubKey:        val.PubKey,
			Commission:    val.Commission,
			AccumReward:   val.AccumReward,
			AbsentTimes:   val.AbsentTimes,
		})
	}

	appState.MaxGas = s.GetMaxGas()
	appState.StartHeight = s.height
	appState.TotalSlashed = s.GetTotalSlashed()

	return appState
}

func (s *StateDB) Import(appState types.AppState) {
	s.SetMaxGas(appState.MaxGas)
	s.setTotalSlashed(appState.TotalSlashed)

	for _, a := range appState.Accounts {
		account := s.GetOrNewStateObject(a.Address)

		account.data.Nonce = a.Nonce

		if a.MultisigData != nil {
			account.data.MultisigData.Addresses = a.MultisigData.Addresses
			account.data.MultisigData.Threshold = a.MultisigData.Threshold
			account.data.MultisigData.Weights = a.MultisigData.Weights
		}

		for _, b := range a.Balance {
			account.SetBalance(b.Coin, b.Value)
		}

		s.setStateObject(account)
		s.MarkStateObjectDirty(a.Address)
	}

	for _, c := range appState.Coins {
		s.CreateCoin(c.Symbol, c.Name, c.Volume, c.Crr, c.ReserveBalance)
	}

	vals := &stateValidators{}
	for _, v := range appState.Validators {
		vals.data = append(vals.data, Validator{
			RewardAddress: v.RewardAddress,
			TotalBipStake: v.TotalBipStake,
			PubKey:        v.PubKey,
			Commission:    v.Commission,
			AccumReward:   v.AccumReward,
			AbsentTimes:   v.AbsentTimes,
		})
	}
	s.SetStateValidators(vals)
	s.MarkStateValidatorsDirty()

	cands := &stateCandidates{}
	for _, c := range appState.Candidates {
		stakes := make([]Stake, len(c.Stakes))
		for i, stake := range c.Stakes {
			stakes[i] = Stake{
				Owner:    stake.Owner,
				Coin:     stake.Coin,
				Value:    stake.Value,
				BipValue: stake.BipValue,
			}
		}
		cands.data = append(cands.data, Candidate{
			RewardAddress:  c.RewardAddress,
			OwnerAddress:   c.OwnerAddress,
			TotalBipStake:  c.TotalBipStake,
			PubKey:         c.PubKey,
			Commission:     c.Commission,
			Stakes:         stakes,
			CreatedAtBlock: 1,
			Status:         c.Status,
		})
	}
	s.setStateCandidates(cands)
	s.MarkStateCandidateDirty()

	for _, hashString := range appState.UsedChecks {
		hash, _ := hex.DecodeString(string(hashString))
		s.useCheckHash(hash)
	}

	for _, ff := range appState.FrozenFunds {
		frozenFunds := s.GetOrNewStateFrozenFunds(ff.Height)
		frozenFunds.AddFund(ff.Address, ff.CandidateKey, ff.Coin, ff.Value)
		s.setStateFrozenFunds(frozenFunds)
	}
}

func (s *StateDB) CheckForInvariants() error {
	height := s.height
	if height <= 1 {
		return nil
	}

	genesisFile := utils.GetMinterHome() + "/config/genesis.json"
	genesis, err := tmTypes.GenesisDocFromFile(genesisFile)
	if err != nil {
		panic(err)
	}

	var genesisState types.AppState
	_ = amino.UnmarshalJSON(genesis.AppState, &genesisState)

	GenesisAlloc := big.NewInt(0)
	for _, account := range genesisState.Accounts {
		for _, bal := range account.Balance {
			if bal.Coin.IsBaseCoin() {
				GenesisAlloc.Add(GenesisAlloc, bal.Value)
			}
		}
	}

	for _, candidate := range genesisState.Candidates {
		for _, stake := range candidate.Stakes {
			if stake.Coin.IsBaseCoin() {
				GenesisAlloc.Add(GenesisAlloc, stake.Value)
			}
		}
	}

	for _, coin := range genesisState.Coins {
		GenesisAlloc.Add(GenesisAlloc, coin.ReserveBalance)
	}

	for _, ff := range genesisState.FrozenFunds {
		if ff.Coin.IsBaseCoin() {
			GenesisAlloc.Add(GenesisAlloc, ff.Value)
		}
	}

	totalBasecoinVolume := big.NewInt(0)

	coinSupplies := map[types.CoinSymbol]*big.Int{}
	coinTotalOwned := map[types.CoinSymbol]*big.Int{}

	s.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == addressPrefix[0] {
			account := s.GetOrNewStateObject(types.BytesToAddress(key[1:]))

			for coin, value := range account.Balances().Data {
				if coin.IsBaseCoin() {
					totalBasecoinVolume.Add(totalBasecoinVolume, value)
					continue
				}

				if coinTotalOwned[coin] == nil {
					coinTotalOwned[coin] = big.NewInt(0)
				}
				coinTotalOwned[coin].Add(coinTotalOwned[coin], value)
			}

		}

		if key[0] == coinPrefix[0] {
			coin := s.GetStateCoin(types.StrToCoinSymbol(string(key[1:])))

			totalBasecoinVolume.Add(totalBasecoinVolume, coin.ReserveBalance())
			coinSupplies[coin.symbol] = coin.Volume()
		}

		if key[0] == frozenFundsPrefix[0] {
			height := binary.BigEndian.Uint64(key[1:])
			frozenFunds := s.GetStateFrozenFunds(height)

			for _, frozenFund := range frozenFunds.List() {
				if frozenFund.Coin.IsBaseCoin() {
					totalBasecoinVolume.Add(totalBasecoinVolume, frozenFund.Value)
					continue
				}

				if coinTotalOwned[frozenFund.Coin] == nil {
					coinTotalOwned[frozenFund.Coin] = big.NewInt(0)
				}
				coinTotalOwned[frozenFund.Coin].Add(coinTotalOwned[frozenFund.Coin], frozenFund.Value)
			}
		}

		return false
	})

	candidates := s.getStateCandidates()
	if candsCount := len(candidates.data); candsCount > validators.GetCandidatesCountForBlock(height) {
		return fmt.Errorf("too many candidates in blockchain. Expected %d, got %d",
			validators.GetCandidatesCountForBlock(height), candsCount)
	}

	for _, candidate := range candidates.data {
		for _, stake := range candidate.Stakes {
			if stake.Coin.IsBaseCoin() {
				totalBasecoinVolume.Add(totalBasecoinVolume, stake.Value)
				continue
			}

			if coinTotalOwned[stake.Coin] == nil {
				coinTotalOwned[stake.Coin] = big.NewInt(0)
			}
			coinTotalOwned[stake.Coin].Add(coinTotalOwned[stake.Coin], stake.Value)
		}
	}

	vals := s.getStateValidators()
	if valsCount := len(vals.data); valsCount > validators.GetValidatorsCountForBlock(height) {
		return fmt.Errorf("too many validators in blockchain. Expected %d, got %d",
			validators.GetValidatorsCountForBlock(height), valsCount)
	}

	for _, val := range vals.data {
		totalBasecoinVolume.Add(totalBasecoinVolume, val.AccumReward)
	}

	predictedBasecoinVolume := big.NewInt(0)
	predictedBasecoinVolume.Add(predictedBasecoinVolume, rewards.BeforeGenesis)
	for i := uint64(1); i < height; i++ {
		predictedBasecoinVolume.Add(predictedBasecoinVolume, rewards.GetRewardForBlock(i))
	}
	predictedBasecoinVolume.Sub(predictedBasecoinVolume, s.GetTotalSlashed())
	predictedBasecoinVolume.Add(predictedBasecoinVolume, GenesisAlloc)

	if height >= upgrades.UpgradeBlock0 {
		d, _ := big.NewInt(0).SetString("35703071844419651412692", 10)
		predictedBasecoinVolume.Sub(predictedBasecoinVolume, d)
	}

	delta := big.NewInt(0).Sub(predictedBasecoinVolume, totalBasecoinVolume)

	if delta.Cmp(big.NewInt(0)) != 0 {
		e := fmt.Errorf("smth wrong with total base coins in blockchain. Expected total supply to be %s, got %s",
			predictedBasecoinVolume, totalBasecoinVolume)

		if delta.Cmp(helpers.BipToPip(big.NewInt(1000))) == 1 {
			println(fmt.Sprintf("CRITICAL INVARIANTS FAILURE (H:%d): %s", height, e))
			os.Exit(1)
		}

		return e
	}

	for coin, volume := range coinSupplies {
		if volume.Cmp(coinTotalOwned[coin]) != 0 {
			return fmt.Errorf("smth wrong with %s coin in blockchain. Total supply (%s) does not match total owned (%s)",
				coin, volume, coinTotalOwned[coin])
		}
	}

	return nil
}

func (s *StateDB) Height() uint64 {
	return s.height
}

func (s *StateDB) DB() dbm.DB {
	return s.db
}
