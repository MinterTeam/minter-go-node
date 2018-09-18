// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/danil-lashin/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
	"math/big"
	"sync"

	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"

	"bytes"
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/dao"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"sort"
)

const UnbondPeriod = 518400

var (
	ValidatorMaxAbsentTimes = 12

	addressPrefix     = []byte("a")
	coinPrefix        = []byte("c")
	frozenFundsPrefix = []byte("f")
	usedCheckPrefix   = []byte("u")
	candidatesKey     = []byte("t")
	validatorsKey     = []byte("v")

	cfg = config.GetConfig()
)

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Coins
// * Accounts
type StateDB struct {
	db   dbm.DB
	iavl *iavl.MutableTree

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects      map[types.Address]*stateObject
	stateObjectsDirty map[types.Address]struct{}

	stateCoins      map[types.CoinSymbol]*stateCoin
	stateCoinsDirty map[types.CoinSymbol]struct{}

	stateFrozenFunds      map[uint64]*stateFrozenFund
	stateFrozenFundsDirty map[uint64]struct{}

	stateCandidates      *stateCandidates
	stateCandidatesDirty bool

	stateValidators      *stateValidators
	stateValidatorsDirty bool

	stakeCache map[types.CoinSymbol]StakeCache

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	thash, bhash types.Hash

	lock sync.Mutex
}

type StakeCache struct {
	TotalValue *big.Int
	BipValue   *big.Int
}

// Create a new state from a given trie
func New(height int64, db dbm.DB) (*StateDB, error) {

	tree := iavl.NewMutableTree(db, 128)

	_, err := tree.LoadVersion(height)

	if err != nil {
		return nil, err
	}

	return &StateDB{
		db:                    db,
		iavl:                  tree,
		stateObjects:          make(map[types.Address]*stateObject),
		stateObjectsDirty:     make(map[types.Address]struct{}),
		stateCoins:            make(map[types.CoinSymbol]*stateCoin),
		stateCoinsDirty:       make(map[types.CoinSymbol]struct{}),
		stateFrozenFunds:      make(map[uint64]*stateFrozenFund),
		stateFrozenFundsDirty: make(map[uint64]struct{}),
		stateCandidates:       nil,
		stateCandidatesDirty:  false,
		stakeCache:            make(map[types.CoinSymbol]StakeCache),
	}, nil
}

// setError remembers the first non-nil error it is called with.
func (s *StateDB) setError(err error) {

	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *StateDB) Error() error {
	return s.dbErr
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *StateDB) Empty(addr types.Address) bool {
	so := s.getStateObject(addr)
	return so == nil || so.empty()
}

// Retrieve the balance from the given address or 0 if object not found
func (s *StateDB) GetBalance(addr types.Address, coinSymbol types.CoinSymbol) *big.Int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance(coinSymbol)
	}
	return types.Big0
}

func (s *StateDB) GetBalances(addr types.Address) Balances {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balances()
	}

	def := make(map[types.CoinSymbol]*big.Int)

	return Balances{
		Data: def,
	}
}

func (s *StateDB) GetNonce(addr types.Address) uint64 {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

// Database retrieves the low level database supporting the lower level trie ops.
func (s *StateDB) Database() dbm.DB {
	return s.db
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
func (s *StateDB) updateStateObject(stateObject *stateObject) {
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
	binary.BigEndian.PutUint64(height, stateFrozenFund.blockHeight)

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

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteStateObject(stateObject *stateObject) {
	stateObject.deleted = true
	addr := stateObject.Address()

	s.iavl.Remove(append(addressPrefix, addr[:]...))
}

// deleteStateCoin removes the given object from the state trie.
func (s *StateDB) deleteStateCoin(stateCoin *stateCoin) {
	symbol := stateCoin.Symbol()
	s.iavl.Remove(append(coinPrefix, symbol[:]...))
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteFrozenFunds(stateFrozenFund *stateFrozenFund) {
	stateFrozenFund.deleted = true
	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, stateFrozenFund.blockHeight)
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
	binary.BigEndian.PutUint64(height, blockHeight)
	key := append(frozenFundsPrefix, height...)

	// Load the object from the database.
	_, enc := s.iavl.Get(key)
	if len(enc) == 0 {
		return nil
	}
	var data FrozenFunds
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		// log.Error("Failed to decode state coin", "symbol", symbol, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newFrozenFund(s, blockHeight, data, s.MarkStateFrozenFundsDirty)
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
		// log.Error("Failed to decode state coin", "symbol", symbol, "err", err)
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
		return nil
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
		return nil
	}
	// Insert into the live set.
	obj := newValidator(s, data, s.MarkStateValidatorsDirty)
	s.setStateValidators(obj)
	return obj
}

// Retrieve a state object given my the address. Returns nil if not found.
func (s *StateDB) getStateObject(addr types.Address) (stateObject *stateObject) {
	// Prefer 'live' objects.
	if obj := s.stateObjects[addr]; obj != nil {
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
		// log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newObject(s, addr, data, s.MarkStateObjectDirty)
	s.setStateObject(obj)
	return obj
}

func (s *StateDB) setStateObject(object *stateObject) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateObjects[object.Address()] = object
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
func (s *StateDB) GetOrNewStateObject(addr types.Address) *stateObject {
	stateObject := s.getStateObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = s.createObject(addr)
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

	s.stateObjectsDirty[addr] = struct{}{}
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

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (s *StateDB) createObject(addr types.Address) (newobj, prev *stateObject) {
	prev = s.getStateObject(addr)
	newobj = newObject(s, addr, Account{}, s.MarkStateObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	s.setStateObject(newobj)
	return newobj, prev
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
	reserve *big.Int,
	creator types.Address) *stateCoin {

	newC := newCoin(s, symbol, Coin{
		Name:           name,
		Symbol:         symbol,
		Volume:         volume,
		Crr:            crr,
		ReserveBalance: reserve,
		Creator:        creator,
	}, s.MarkStateCoinDirty)
	s.setStateCoin(newC)
	return newC
}

func (s *StateDB) CreateValidator(
	address types.Address,
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
		CandidateAddress: address,
		TotalBipStake:    initialStake,
		PubKey:           pubkey,
		Commission:       commission,
		AccumReward:      big.NewInt(0),
		AbsentTimes:      NewBitArray(24),
	})

	s.MarkStateValidatorsDirty()
	s.setStateValidators(vals)
	return vals
}

func (s *StateDB) CreateCandidate(
	address types.Address,
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
		CandidateAddress: address,
		PubKey:           pubkey,
		Commission:       commission,
		Stakes: []Stake{
			{
				Owner: address,
				Coin:  coin,
				Value: initialStake,
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
func (s *StateDB) Commit(deleteEmptyObjects bool) (root []byte, version int64, err error) {

	// Commit objects to the trie.
	for _, addr := range getOrderedObjectsKeys(s.stateObjects) {
		stateObject := s.stateObjects[addr]
		_, isDirty := s.stateObjectsDirty[addr]
		switch {
		case stateObject.suicided || (isDirty && deleteEmptyObjects && stateObject.empty()):
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.deleteStateObject(stateObject)
		case isDirty:
			// Update the object in the main account trie.
			s.updateStateObject(stateObject)
		}
		delete(s.stateObjectsDirty, addr)
	}

	// Commit coins to the trie.
	for _, symbol := range getOrderedCoinsKeys(s.stateCoins) {
		stateCoin := s.stateCoins[symbol]
		_, isDirty := s.stateCoinsDirty[symbol]
		if isDirty {
			if stateCoin.data.Volume.Cmp(types.Big0) == 0 {
				s.deleteStateCoin(stateCoin)
			} else {
				s.updateStateCoin(stateCoin)
			}
		}

		delete(s.stateCoinsDirty, symbol)
	}

	// Commit frozen funds to the trie.
	for _, block := range getOrderedFrozenFundsKeys(s.stateFrozenFunds) {
		frozenFund := s.stateFrozenFunds[block]
		_, isDirty := s.stateFrozenFundsDirty[block]
		switch {
		case frozenFund.deleted:
			s.deleteFrozenFunds(frozenFund)
		case isDirty:
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

	hash, version, err := s.iavl.SaveVersion()

	if cfg.ValidatorMode && version > 1 {
		err = s.iavl.DeleteVersion(version - 1)

		if err != nil {
			panic(err)
		}
	}

	return hash, version, err
}

func getOrderedObjectsKeys(objects map[types.Address]*stateObject) []types.Address {
	keys := make([]types.Address, 0, len(objects))
	for k := range objects {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func getOrderedCoinsKeys(objects map[types.CoinSymbol]*stateCoin) []types.CoinSymbol {
	keys := make([]types.CoinSymbol, 0, len(objects))
	for k := range objects {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func getOrderedFrozenFundsKeys(objects map[uint64]*stateFrozenFund) []uint64 {
	keys := make([]uint64, 0, len(objects))
	for k := range objects {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	return keys
}

func (s *StateDB) CoinExists(symbol types.CoinSymbol) bool {

	if symbol == types.GetBaseCoin() {
		return true
	}

	stateCoin := s.getStateCoin(symbol)

	return stateCoin != nil
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

func (s *StateDB) GetStateCoin(symbol types.CoinSymbol) *stateCoin {
	return s.getStateCoin(symbol)
}

func (s *StateDB) AddCoinVolume(symbol types.CoinSymbol, value *big.Int) {
	stateCoin := s.GetStateCoin(symbol)
	if stateCoin != nil {
		stateCoin.AddVolume(value)
	}
}

func (s *StateDB) SubCoinVolume(symbol types.CoinSymbol, value *big.Int) {
	stateCoin := s.GetStateCoin(symbol)
	if stateCoin != nil {
		stateCoin.SubVolume(value)
	}
}

func (s *StateDB) AddCoinReserve(symbol types.CoinSymbol, value *big.Int) {
	stateCoin := s.GetStateCoin(symbol)
	if stateCoin != nil {
		stateCoin.AddReserve(value)
	}
}

func (s *StateDB) SubCoinReserve(symbol types.CoinSymbol, value *big.Int) {
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

	sort.Slice(activeCandidates, func(i, j int) bool {
		return activeCandidates[i].TotalBipStake.Cmp(candidates[j].TotalBipStake) == -1
	})

	if len(activeCandidates) < count {
		count = len(activeCandidates)
	}

	sort.Slice(activeCandidates, func(i, j int) bool {
		return activeCandidates[i].TotalBipStake.Cmp(activeCandidates[j].TotalBipStake) == 1
	})

	return activeCandidates[:count]
}

func (s *StateDB) AddAccumReward(pubkey types.Pubkey, reward *big.Int) {
	validators := s.getStateValidators()

	for i := range validators.data {
		if bytes.Equal(validators.data[i].PubKey, pubkey) {
			validators.data[i].AccumReward.Add(validators.data[i].AccumReward, reward)
			s.setStateValidators(validators)
			s.MarkStateValidatorsDirty()
			return
		}
	}
}

func (s *StateDB) PayRewards(height int64) {
	edb := eventsdb.GetCurrent()

	validators := s.getStateValidators()

	for i := range validators.data {
		validator := validators.data[i]

		if validator.AccumReward.Cmp(types.Big0) == 1 {

			totalReward := big.NewInt(0).Set(validator.AccumReward)

			// pay commission to DAO
			DAOReward := big.NewInt(0).Set(totalReward)
			DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
			DAOReward.Div(DAOReward, big.NewInt(100))
			s.AddBalance(dao.Address, types.GetBaseCoin(), DAOReward)
			edb.AddEvent(height, eventsdb.RewardEvent{
				Role:            eventsdb.RoleDAO,
				Address:         dao.Address,
				Amount:          DAOReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			// pay commission to Developers
			DevelopersReward := big.NewInt(0).Set(totalReward)
			DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
			DevelopersReward.Div(DevelopersReward, big.NewInt(100))
			s.AddBalance(developers.Address, types.GetBaseCoin(), DevelopersReward)
			edb.AddEvent(height, eventsdb.RewardEvent{
				Role:            eventsdb.RoleDevelopers,
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
			s.AddBalance(validator.CandidateAddress, types.GetBaseCoin(), validatorReward)
			edb.AddEvent(height, eventsdb.RewardEvent{
				Role:            eventsdb.RoleValidator,
				Address:         validator.CandidateAddress,
				Amount:          validatorReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			candidate := s.GetStateCandidate(validator.PubKey)

			// pay rewards
			for j := range candidate.Stakes {
				stake := candidate.Stakes[j]

				if stake.BipValue == nil {
					continue
				}

				reward := big.NewInt(0).Set(totalReward)
				reward.Mul(reward, stake.BipValue)
				reward.Div(reward, validator.TotalBipStake)

				if reward.Cmp(types.Big0) < 1 {
					continue
				}

				s.AddBalance(stake.Owner, types.GetBaseCoin(), reward)

				edb.AddEvent(height, eventsdb.RewardEvent{
					Role:            eventsdb.RoleDelegator,
					Address:         stake.Owner,
					Amount:          reward.Bytes(),
					ValidatorPubKey: candidate.PubKey,
				})
			}

			validator.AccumReward.SetInt64(0)
		}
	}

	s.setStateValidators(validators)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) RecalculateTotalStakeValues() {
	stateCandidates := s.getStateCandidates()
	validators := s.getStateValidators()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]

		totalBipStake := big.NewInt(0)

		for j := range candidate.Stakes {
			stake := &candidate.Stakes[j]
			stake.BipValue = stake.CalcBipValue(s)
			totalBipStake.Add(totalBipStake, stake.BipValue)
		}

		candidate.TotalBipStake = totalBipStake

		for j := range validators.data {
			if bytes.Equal(validators.data[j].PubKey, candidate.PubKey) {
				validators.data[j].TotalBipStake = totalBipStake
				break
			}
		}
	}

	s.setStateValidators(validators)
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
					Owner: sender,
					Coin:  coin,
					Value: value,
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
			// todo: remove if stake == 0
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
	trieHash := append(usedCheckPrefix, checkHash...)

	s.iavl.Set(trieHash, []byte{0x1})
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
}

func (s *StateDB) SetValidatorPresent(height int64, address [20]byte) {
	validators := s.getStateValidators()

	for i := range validators.data {
		validator := &validators.data[i]
		if validator.GetAddress() == address {
			validator.AbsentTimes.SetIndex(uint(height%24), false)
		}
	}

	s.setStateValidators(validators)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) SetValidatorAbsent(height int64, address [20]byte) {
	edb := eventsdb.GetCurrent()

	validators := s.getStateValidators()

	for i := range validators.data {
		validator := &validators.data[i]
		if validator.GetAddress() == address {

			candidates := s.getStateCandidates()

			var candidate *Candidate

			for i := range candidates.data {
				if candidates.data[i].GetAddress() == address {
					candidate = &candidates.data[i]
				}
			}

			if candidate.Status == CandidateStatusOffline {
				return
			}

			validator.AbsentTimes.SetIndex(uint(height%24), true)

			if validator.CountAbsentTimes() > ValidatorMaxAbsentTimes {
				candidate.Status = CandidateStatusOffline
				validator.AbsentTimes = NewBitArray(24)
				validator.toDrop = true

				totalStake := big.NewInt(0)

				for j, stake := range candidate.Stakes {
					// TODO: sub custom's coin volume and reserve
					newValue := big.NewInt(0).Set(stake.Value)
					newValue.Mul(newValue, big.NewInt(99))
					newValue.Div(newValue, big.NewInt(100))

					slashed := big.NewInt(0).Set(stake.Value)
					slashed.Sub(slashed, newValue)

					edb.AddEvent(height, eventsdb.SlashEvent{
						Address:         stake.Owner,
						Amount:          slashed.Bytes(),
						Coin:            stake.Coin,
						ValidatorPubKey: candidate.PubKey,
					})

					candidate.Stakes[j] = Stake{
						Owner: stake.Owner,
						Coin:  stake.Coin,
						Value: newValue,
					}
					totalStake.Add(totalStake, newValue)
				}

				// TODO: recalc total stake in bips
				validator.TotalBipStake = totalStake
			}

			s.setStateCandidates(candidates)
			s.MarkStateCandidateDirty()
		}
	}

	s.setStateValidators(validators)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) PunishByzantineValidator(currentBlock uint64, address [20]byte) {

	edb := eventsdb.GetCurrent()

	validators := s.getStateValidators()

	for i := range validators.data {
		validator := &validators.data[i]
		if validator.GetAddress() == address {
			candidates := s.getStateCandidates()

			var candidate *Candidate

			for i := range candidates.data {
				if candidates.data[i].GetAddress() == address {
					candidate = &candidates.data[i]
				}
			}

			for _, stake := range candidate.Stakes {
				// TODO: sub custom's coin volume and reserve
				newValue := big.NewInt(0).Set(stake.Value)
				newValue.Mul(newValue, big.NewInt(95))
				newValue.Div(newValue, big.NewInt(100))

				slashed := big.NewInt(0).Set(stake.Value)
				slashed.Sub(slashed, newValue)

				edb.AddEvent(int64(currentBlock), eventsdb.SlashEvent{
					Address:         stake.Owner,
					Amount:          slashed.Bytes(),
					Coin:            stake.Coin,
					ValidatorPubKey: candidate.PubKey,
				})

				s.GetOrNewStateFrozenFunds(currentBlock+UnbondPeriod).AddFund(stake.Owner, candidate.PubKey, stake.Coin, newValue)
			}

			candidate.Stakes = []Stake{}
			candidate.Status = CandidateStatusOffline
			validator.AccumReward = big.NewInt(0)
			validator.TotalBipStake = big.NewInt(0)

			s.setStateCandidates(candidates)
			s.MarkStateCandidateDirty()
		}
	}

	s.setStateValidators(validators)
	s.MarkStateValidatorsDirty()
}

func (s *StateDB) PunishFrozenFundsWithAddress(fromBlock uint64, toBlock uint64, address [20]byte) {
	for i := fromBlock; i <= toBlock; i++ {
		frozenFund := s.getStateFrozenFunds(i)

		if frozenFund == nil {
			continue
		}

		frozenFund.PunishFund(address)
	}
}

func (s *StateDB) SetNewValidators(candidates []Candidate) {
	oldVals := s.getStateValidators()

	var newVals Validators

	for _, candidate := range candidates {
		accumReward := big.NewInt(0)
		absentTimes := NewBitArray(24)

		for _, oldVal := range oldVals.data {
			if oldVal.GetAddress() == candidate.GetAddress() {
				accumReward = oldVal.AccumReward
				absentTimes = oldVal.AbsentTimes
			}
		}

		newVals = append(newVals, Validator{
			CandidateAddress: candidate.CandidateAddress,
			TotalBipStake:    candidate.TotalBipStake,
			PubKey:           candidate.PubKey,
			Commission:       candidate.Commission,
			AccumReward:      accumReward,
			AbsentTimes:      absentTimes,
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

		for j, stake := range candidate.Stakes {
			if stake.Value.Cmp(types.Big0) == 0 {
				candidate.Stakes = append(candidate.Stakes[:j], candidate.Stakes[j+1:]...)
			}
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}
