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
	"math/big"
	"sync"

	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"

	"bytes"
	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/dao"
	abci "github.com/tendermint/abci/types"
	"sort"
)

var (
	// emptyState is the known hash of an empty state trie entry.
	emptyState              = crypto.Keccak256Hash(nil)
	candidatesKey           = []byte("candidates")
	CandidateMaxAbsentTimes = uint(12)
)

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Coins
// * Accounts
type StateDB struct {
	db   Database
	trie Trie

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects      map[types.Address]*stateObject
	stateObjectsDirty map[types.Address]struct{}

	stateCoins      map[types.CoinSymbol]*stateCoin
	stateCoinsDirty map[types.CoinSymbol]struct{}

	stateFrozenFunds      map[uint64]*stateFrozenFund
	stateFrozenFundsDirty map[uint64]struct{}

	stateCandidates      *stateCandidates
	stateCandidatesDirty bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	thash, bhash types.Hash

	lock sync.Mutex
}

// Create a new state from a given trie
func New(root types.Hash, db Database) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		db:                    db,
		trie:                  tr,
		stateObjects:          make(map[types.Address]*stateObject),
		stateObjectsDirty:     make(map[types.Address]struct{}),
		stateCoins:            make(map[types.CoinSymbol]*stateCoin),
		stateCoinsDirty:       make(map[types.CoinSymbol]struct{}),
		stateFrozenFunds:      make(map[uint64]*stateFrozenFund),
		stateFrozenFundsDirty: make(map[uint64]struct{}),
		stateCandidates:       nil,
		stateCandidatesDirty:  false,
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
func (s *StateDB) Database() Database {
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
	s.setError(s.trie.TryUpdate(addr[:], data))
}

func (s *StateDB) updateStateFrozenFund(stateFrozenFund *stateFrozenFund) {
	blockHeight := stateFrozenFund.BlockHeight()
	data, err := rlp.EncodeToBytes(stateFrozenFund)
	if err != nil {
		panic(fmt.Errorf("can't encode frozen fund at %d: %v", blockHeight, err))
	}
	key := []byte("frozenFundsForBlock:" + string(blockHeight))
	// TODO: change key generation
	s.setError(s.trie.TryUpdate(key, data))
}

func (s *StateDB) updateStateCoin(stateCoin *stateCoin) {
	symbol := stateCoin.Symbol()
	data, err := rlp.EncodeToBytes(stateCoin)
	if err != nil {
		panic(fmt.Errorf("can't encode coin at %x: %v", symbol[:], err))
	}
	// TODO: change key generation
	s.setError(s.trie.TryUpdate(symbol[:], data))
}

func (s *StateDB) updateStateCandidates(stateCandidates *stateCandidates) {
	data, err := rlp.EncodeToBytes(stateCandidates)
	if err != nil {
		panic(fmt.Errorf("can't encode candidates: %v", err))
	}
	// TODO: change key generation (IMPORTANT, possible attack)
	err = s.trie.TryUpdate(candidatesKey, data)
	s.setError(err)
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteStateObject(stateObject *stateObject) {
	stateObject.deleted = true
	addr := stateObject.Address()
	s.setError(s.trie.TryDelete(addr[:]))
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteFrozenFunds(stateFrozenFund *stateFrozenFund) {
	stateFrozenFund.deleted = true
	key := []byte("frozenFundsForBlock:" + string(stateFrozenFund.blockHeight))
	s.setError(s.trie.TryDelete(key))
}

// Retrieve a state frozen funds by block height. Returns nil if not found.
func (s *StateDB) getStateFrozenFunds(blockHeight uint64) (stateFrozenFund *stateFrozenFund) {
	// Prefer 'live' objects.
	if obj := s.stateFrozenFunds[blockHeight]; obj != nil {
		return obj
	}

	// TODO: change key generation
	key := []byte("frozenFundsForBlock:" + string(blockHeight))
	// Load the object from the database.
	enc, err := s.trie.TryGet(key)
	if len(enc) == 0 {
		s.setError(err)
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

	// TODO: change key generation

	// Load the object from the database.
	enc, err := s.trie.TryGet(symbol[:])
	if len(enc) == 0 {
		s.setError(err)
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

// Retrieve a state candidate by public key. Returns nil if not found.
func (s *StateDB) getStateCandidates() (stateCandidates *stateCandidates) {
	// Prefer 'live' objects.
	if s.stateCandidates != nil {
		return s.stateCandidates
	}

	// TODO: change key generation
	// Load the object from the database.
	enc, err := s.trie.TryGet(candidatesKey)
	if len(enc) == 0 {
		s.setError(err)
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
	enc, err := s.trie.TryGet(addr[:])
	if len(enc) == 0 {
		s.setError(err)
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
		ReserveCoin:    types.GetBaseCoin(),
		ReserveBalance: reserve,
		Creator:        creator,
	}, s.MarkStateCoinDirty)
	s.setStateCoin(newC)
	return newC
}

func (s *StateDB) CreateCandidate(
	address types.Address,
	pubkey types.Pubkey,
	commission uint,
	currentBlock uint,
	initialStake *big.Int) *stateCandidates {

	candidates := s.getStateCandidates()

	if candidates == nil {
		candidates = newCandidate(s, Candidates{}, s.MarkStateCandidateDirty)
	}

	candidates.data = append(candidates.data, Candidate{
		CandidateAddress: address,
		TotalStake:       initialStake,
		PubKey:           pubkey,
		Commission:       commission,
		AccumReward:      big.NewInt(0),
		Stakes: []Stake{
			{
				Owner: address,
				Value: initialStake,
			},
		},
		CreatedAtBlock: currentBlock,
		Status:         CandidateStatusOffline,
		AbsentTimes:    0,
	})

	s.MarkStateCandidateDirty()
	s.setStateCandidates(candidates)
	return candidates
}

// Commit writes the state to the underlying in-memory trie database.
func (s *StateDB) Commit(deleteEmptyObjects bool) (root types.Hash, err error) {

	// Commit objects to the trie.
	for addr, stateObject := range s.stateObjects {
		_, isDirty := s.stateObjectsDirty[addr]
		switch {
		case stateObject.suicided || (isDirty && deleteEmptyObjects && stateObject.empty()):
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.deleteStateObject(stateObject)
		case isDirty:
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitTrie(s.db); err != nil {
				return types.Hash{}, err
			}
			// Update the object in the main account trie.
			s.updateStateObject(stateObject)
		}
		delete(s.stateObjectsDirty, addr)
	}

	// Commit coins to the trie.
	for symbol, stateCoin := range s.stateCoins {
		_, isDirty := s.stateCoinsDirty[symbol]
		switch {
		case isDirty:
			s.updateStateCoin(stateCoin)
		}
		delete(s.stateCoinsDirty, symbol)
	}

	// Commit frozen funds to the trie.
	for block, frozenFund := range s.stateFrozenFunds {
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
		s.updateStateCandidates(s.stateCandidates)
		s.stateCandidatesDirty = false
	}

	// Write trie changes.
	root, err = s.trie.Commit(func(leaf []byte, parent types.Hash) error {
		var account Account
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyState {
			s.db.TrieDB().Reference(account.Root, parent)
		}
		return nil
	})
	// log.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return root, err
}

func (s *StateDB) CoinExists(symbol types.CoinSymbol) bool {

	if symbol == types.GetBaseCoin() {
		return true
	}

	stateCoin := s.getStateCoin(symbol)
	if stateCoin != nil {
		return true
	}

	return false
}

func (s *StateDB) CandidateExists(key types.Pubkey) bool {

	stateCandidates := s.getStateCandidates()

	if stateCandidates == nil {
		return false
	}

	for _, candidate := range stateCandidates.data {
		if bytes.Compare(candidate.PubKey, key) == 0 {
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
		if bytes.Compare(candidate.PubKey, key) == 0 {
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

func (s *StateDB) GetValidators(count int) ([]abci.Validator, []Candidate) {
	stateCandidates := s.getStateCandidates()

	if stateCandidates == nil {
		return nil, nil
	}

	candidates := stateCandidates.data

	var activeCandidates Candidates

	// get only active candidates
	for _, v := range candidates {
		if v.Status == CandidateStatusOnline {
			activeCandidates = append(activeCandidates, v)
		}
	}

	sort.Slice(activeCandidates, func(i, j int) bool {
		return activeCandidates[i].TotalStake.Cmp(candidates[j].TotalStake) == -1
	})

	if len(activeCandidates) < count {
		count = len(activeCandidates)
	}

	validators := make([]abci.Validator, count)

	// calculate total power
	totalPower := big.NewInt(0)
	for _, candidate := range activeCandidates[:count] {
		totalPower.Add(totalPower, candidate.TotalStake)
	}

	for i := range activeCandidates[:count] {
		power := big.NewInt(0).Div(big.NewInt(0).Mul(activeCandidates[i].TotalStake, big.NewInt(100)), totalPower)

		validators[i] = abci.Ed25519Validator(activeCandidates[i].PubKey, power.Int64())
	}

	return validators, activeCandidates
}

func (s *StateDB) AddAccumReward(pubkey types.Pubkey, reward *big.Int) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		if bytes.Compare(stateCandidates.data[i].PubKey, pubkey) == 0 {
			stateCandidates.data[i].AccumReward.Add(stateCandidates.data[i].AccumReward, reward)
			s.setStateCandidates(stateCandidates)
			s.MarkStateCandidateDirty()
			return
		}
	}
}

func (s *StateDB) PayRewards() {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := stateCandidates.data[i]

		if candidate.AccumReward.Cmp(types.Big0) == 1 {

			totalReward := big.NewInt(0).Set(candidate.AccumReward)

			// pay commission to DAO
			DAOReward := big.NewInt(0).Set(totalReward)
			DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
			DAOReward.Div(DAOReward, big.NewInt(100))
			totalReward.Sub(totalReward, DAOReward)
			s.AddBalance(dao.Address, types.GetBaseCoin(), DAOReward)

			// pay commission to validator
			validatorReward := big.NewInt(0).Set(totalReward)
			validatorReward.Mul(validatorReward, big.NewInt(int64(candidate.Commission)))
			validatorReward.Div(validatorReward, big.NewInt(100))
			totalReward.Sub(totalReward, validatorReward)
			s.AddBalance(candidate.CandidateAddress, types.GetBaseCoin(), validatorReward)

			// pay rewards
			for j := range candidate.Stakes {
				stake := candidate.Stakes[j]

				reward := big.NewInt(0).Set(totalReward)
				reward.Mul(reward, stake.Value)
				reward.Div(reward, candidate.TotalStake)

				s.AddBalance(stake.Owner, types.GetBaseCoin(), reward)
			}

			candidate.AccumReward.SetInt64(0)
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) Delegate(sender types.Address, pubkey []byte, value *big.Int) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Compare(candidate.PubKey, pubkey) == 0 {

			exists := false

			for j := range candidate.Stakes {
				stake := &candidate.Stakes[j]
				if bytes.Compare(sender.Bytes(), stake.Owner.Bytes()) == 0 {
					stake.Value.Add(stake.Value, value)
					exists = true
					break
				}
			}

			if !exists {
				candidate.Stakes = append(candidate.Stakes, Stake{
					Owner: sender,
					Value: value,
				})
			}

			candidate.TotalStake.Add(candidate.TotalStake, value)
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) SubStake(sender types.Address, pubkey []byte, value *big.Int) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Compare(candidate.PubKey, pubkey) == 0 {
			// todo: remove if stake == 0
			currentStakeValue := candidate.GetStakeOfAddress(sender).Value
			currentStakeValue.Sub(currentStakeValue, value)
			candidate.TotalStake.Sub(candidate.TotalStake, value)
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) IsCheckUsed(check *check.Check) bool {
	checkHash := check.Hash().Bytes()
	// TODO: change key generation
	trieHash := append(checkHash, byte('C'))

	data, _ := s.trie.TryGet(trieHash)

	return len(data) != 0
}

func (s *StateDB) UseCheck(check *check.Check) {
	checkHash := check.Hash().Bytes()
	// TODO: change key generation
	trieHash := append(checkHash, byte('C'))

	s.setError(s.trie.TryUpdate(trieHash, []byte{0x1}))
}

func (s *StateDB) SetCandidateOnline(pubkey []byte) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Compare(candidate.PubKey, pubkey) == 0 {
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
		if bytes.Compare(candidate.PubKey, pubkey) == 0 {
			candidate.Status = CandidateStatusOffline
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) SetValidatorAbsent(pubkey types.Pubkey) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Compare(candidate.PubKey, pubkey) == 0 {

			if candidate.Status == CandidateStatusOffline {
				return
			}

			candidate.AbsentTimes = candidate.AbsentTimes + 1

			if candidate.AbsentTimes > CandidateMaxAbsentTimes {
				candidate.Status = CandidateStatusOffline
				candidate.AbsentTimes = 0

				totalStake := big.NewInt(0)

				for j, stake := range candidate.Stakes {
					newValue := big.NewInt(0).Set(stake.Value)
					newValue.Mul(newValue, big.NewInt(99))
					newValue.Div(newValue, big.NewInt(100))

					candidate.Stakes[j] = Stake{
						Owner: stake.Owner,
						Value: newValue,
					}
					totalStake.Add(totalStake, newValue)
				}

				candidate.TotalStake = totalStake
			}
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) PunishByzantineCandidate(PubKey []byte) {

	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Compare(candidate.PubKey, PubKey) == 0 {
			candidate.AbsentTimes = candidate.AbsentTimes + 1

			candidate.Stakes = []Stake{}
			candidate.TotalStake = big.NewInt(0)
			candidate.Status = CandidateStatusOffline
			candidate.AccumReward = big.NewInt(0)
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}

func (s *StateDB) RemoveFrozenFundsWithPubKey(fromBlock uint64, toBlock uint64, PubKey []byte) {
	for i := fromBlock; i <= toBlock; i++ {
		frozenFund := s.getStateFrozenFunds(i)

		if frozenFund == nil {
			continue
		}

		frozenFund.RemoveFund(PubKey)
	}
}

func (s *StateDB) SetValidatorPresent(pubkey types.Pubkey) {
	stateCandidates := s.getStateCandidates()

	for i := range stateCandidates.data {
		candidate := &stateCandidates.data[i]
		if bytes.Compare(candidate.PubKey, pubkey) == 0 {
			candidate.AbsentTimes = 0
		}
	}

	s.setStateCandidates(stateCandidates)
	s.MarkStateCandidateDirty()
}
