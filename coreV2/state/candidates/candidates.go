package candidates

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/dao"
	"github.com/MinterTeam/minter-go-node/coreV2/developers"
	"log"
	"sync/atomic"

	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"

	"math/big"
	"sort"
	"sync"
)

// Common constants
const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02

	MaxDelegatorsPerCandidate = 1000
)

const (
	mainPrefix             = 'c'
	pubKeyIDPrefix         = mainPrefix + 'p'
	blockListPrefix        = mainPrefix + 'b'
	maxIDPrefix            = mainPrefix + 'i'
	deleteCandidatesPrefix = mainPrefix + 'd'
	stakesPrefix           = 's'
	totalStakePrefix       = 't'
	updatesPrefix          = 'u'
)

var (
	minValidatorBipStake = helpers.BipToPip(big.NewInt(1000))
)

// RCandidates interface represents Candidates state
type RCandidates interface {
	// Deprecated
	ExportV1(state *types.AppState, height uint64, validators []*types.Candidate) []uint32
	DeletedCandidates() (result []*deletedID)
	Export(state *types.AppState)
	Exists(pubkey types.Pubkey) bool
	IsBlockedPubKey(pubkey types.Pubkey) bool
	PubKey(id uint32) types.Pubkey
	Count() int
	IsNewCandidateStakeSufficient(coin types.CoinID, stake *big.Int, limit int) bool
	IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinID, amount *big.Int) bool
	IsDelegatorStakeAllowed(address types.Address, pubkey types.Pubkey, coin types.CoinID, amount *big.Int) (low, big bool)
	GetStakeValueOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinID) *big.Int
	GetCandidateOwner(pubkey types.Pubkey) types.Address
	GetCandidateControl(pubkey types.Pubkey) types.Address
	GetTotalStake(pubkey types.Pubkey) *big.Int
	LoadCandidates()
	LoadStakesOfCandidate(pubkey types.Pubkey) *big.Int
	GetCandidate(pubkey types.Pubkey) *Candidate
	LoadStakes()
	GetCandidates() []*Candidate
	GetStakes(pubkey types.Pubkey) []*stake
	IsCandidateJailed(pubkey types.Pubkey, block uint64) bool
}

// Candidates struct is a store of Candidates state
type Candidates struct {
	list map[uint32]*Candidate

	isDirty   bool
	blockList map[types.Pubkey]struct{}
	pubKeyIDs map[types.Pubkey]uint32
	maxID     uint32

	db  atomic.Value
	bus *bus.Bus

	lock                sync.RWMutex
	loaded              bool
	isChangedPublicKeys bool

	totalStakes            *big.Int
	deletedCandidates      map[types.Pubkey]*deletedID
	dirtyDeletedCandidates bool
	muDeletedCandidates    sync.RWMutex
}

type deletedID struct {
	ID      uint32
	PubKey  types.Pubkey
	isDirty bool
}

// NewCandidates returns newly created Candidates state with a given bus and iavl
func NewCandidates(bus *bus.Bus, db *iavl.ImmutableTree) *Candidates {
	immutableTree := atomic.Value{}
	loaded := false
	if db != nil {
		immutableTree.Store(db)
	} else {
		loaded = true
	}
	candidates := &Candidates{
		db:                immutableTree,
		loaded:            loaded,
		bus:               bus,
		deletedCandidates: map[types.Pubkey]*deletedID{},
		blockList:         map[types.Pubkey]struct{}{},
		pubKeyIDs:         map[types.Pubkey]uint32{},
		list:              map[uint32]*Candidate{},
		totalStakes:       big.NewInt(0),
	}
	candidates.bus.SetCandidates(NewBus(candidates))

	return candidates
}

func (c *Candidates) immutableTree() *iavl.ImmutableTree {
	db := c.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (c *Candidates) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	if c.immutableTree() == nil && c.loaded {
		c.loaded = false
	}
	c.db.Store(immutableTree)
}

func (c *Candidates) IsChangedPublicKeys() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.isChangedPublicKeys
}

func (c *Candidates) ResetIsChangedPublicKeys() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.isChangedPublicKeys = false
}

func (c *Candidates) SetDeletedCandidates(list []types.DeletedCandidate) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.muDeletedCandidates.Lock()
	defer c.muDeletedCandidates.Unlock()

	c.dirtyDeletedCandidates = true
	for _, deleted := range list {
		id := uint32(deleted.ID)
		//if c.maxID < id { // TODO: FIXME
		//	c.maxID = id
		//}
		c.deletedCandidates[deleted.PubKey] = &deletedID{
			ID:      id,
			PubKey:  deleted.PubKey,
			isDirty: true,
		}
	}
}

func (c *Candidates) IsCandidateJailed(pubkey types.Pubkey, block uint64) bool {
	candidate := c.GetCandidate(pubkey)

	return candidate.JailedUntil >= block
}

// Commit writes changes to iavl, may return an error
func (c *Candidates) Commit(db *iavl.MutableTree, version int64) error {
	keys := c.getOrderedCandidates()

	c.lock.RLock()
	defer c.lock.RUnlock()

	c.muDeletedCandidates.RLock()
	dirtyDeletedCandidates := c.dirtyDeletedCandidates
	c.muDeletedCandidates.RUnlock()

	hasDirty := dirtyDeletedCandidates
	if !hasDirty {
		for _, candidate := range keys {
			candidate.lock.RLock()
			if candidate.isDirty {
				hasDirty = true
				candidate.lock.RUnlock()
				break
			}
			candidate.lock.RUnlock()
		}
	}

	if hasDirty {
		var candidates []*Candidate
		for _, candidate := range keys {
			candidates = append(candidates, candidate)
		}
		data, err := rlp.EncodeToBytes(candidates)
		if err != nil {
			return fmt.Errorf("can't encode candidates: %v", err)
		}

		path := []byte{mainPrefix}
		db.Set(path, data)
	}

	if c.isDirty {
		c.isDirty = false
		var pubIDs []*pubkeyID
		for pk, v := range c.pubKeyIDs {
			pubIDs = append(pubIDs, &pubkeyID{
				PubKey: pk,
				ID:     v,
			})
		}
		sort.SliceStable(pubIDs, func(i, j int) bool {
			return pubIDs[i].ID < pubIDs[j].ID
		})
		pubIDData, err := rlp.EncodeToBytes(pubIDs)
		if err != nil {
			panic(fmt.Sprintf("failed to encode candidates public key with ID: %s", err))
		}

		db.Set([]byte{pubKeyIDPrefix}, pubIDData)

		var blockList []types.Pubkey
		for pubKey := range c.blockList {
			blockList = append(blockList, pubKey)
		}
		sort.SliceStable(blockList, func(i, j int) bool {
			return bytes.Compare(blockList[i].Bytes(), blockList[j].Bytes()) == 1
		})
		blockListData, err := rlp.EncodeToBytes(blockList)
		if err != nil {
			return fmt.Errorf("can't encode block list of candidates: %v", err)
		}
		db.Set([]byte{blockListPrefix}, blockListData)

		db.Set([]byte{maxIDPrefix}, c.maxIDBytes())
	}

	if dirtyDeletedCandidates {
		c.muDeletedCandidates.RLock()

		c.dirtyDeletedCandidates = false
		var deletedCandidates = make([]*deletedID, 0, len(c.deletedCandidates))
		for _, key := range c.deletedCandidates {
			deletedCandidates = append(deletedCandidates, key)
		}

		sort.Slice(deletedCandidates, func(i, j int) bool {
			if deletedCandidates[i].ID == deletedCandidates[j].ID {
				return deletedCandidates[i].PubKey.String() > deletedCandidates[j].PubKey.String()
			}
			return deletedCandidates[i].ID < deletedCandidates[j].ID
		})

		var deletedKeys [][]byte
		for _, id := range deletedCandidates {
			if id.isDirty {
				id.isDirty = false
				db.IterateRange(append([]byte{mainPrefix}, idBytes(id.ID)...), append([]byte{mainPrefix}, idBytes(id.ID+1)...), true, func(key []byte, value []byte) bool {
					if len(key) <= 5 || !(key[5] == stakesPrefix || key[5] == updatesPrefix || key[5] == totalStakePrefix) {
						return false
					}

					deletedKeys = append(deletedKeys, key)
					return false
				})
			}
		}

		for _, key := range deletedKeys {
			db.Remove(key)
		}

		data, err := rlp.EncodeToBytes(deletedCandidates)
		if err != nil {
			return fmt.Errorf("can't encode deletedCandidates: %v", err)
		}

		db.Set([]byte{deleteCandidatesPrefix}, data)

		c.muDeletedCandidates.RUnlock()
	}

	for _, candidate := range keys {
		candidate.lock.Lock()
		candidate.isDirty = false
		dirty := candidate.isTotalStakeDirty
		candidate.lock.Unlock()

		if dirty {
			candidate.lock.Lock()
			candidate.isTotalStakeDirty = false
			totalStakeBytes := candidate.totalBipStake.Bytes()
			candidate.lock.Unlock()

			path := []byte{mainPrefix}
			path = append(path, candidate.idBytes()...)
			path = append(path, totalStakePrefix)
			db.Set(path, totalStakeBytes)
		}

		for index, stake := range candidate.stakes {
			candidate.lock.RLock()
			dirtyStakes := candidate.dirtyStakes[index]
			candidate.lock.RUnlock()
			if !dirtyStakes {
				continue
			}

			candidate.lock.Lock()
			candidate.dirtyStakes[index] = false
			candidate.lock.Unlock()

			path := []byte{mainPrefix}
			path = append(path, candidate.idBytes()...)
			path = append(path, stakesPrefix)
			path = append(path, big.NewInt(int64(index)).Bytes()...)

			isEmpty := stake == nil
			if !isEmpty {
				stake.lock.RLock()
				isEmpty = stake.Value.Sign() == 0
				stake.lock.RUnlock()
			}
			if isEmpty {
				db.Remove(path)

				candidate.lock.Lock()
				candidate.stakes[index] = nil
				candidate.lock.Unlock()
				continue
			}

			data, err := rlp.EncodeToBytes(stake)
			if err != nil {
				return fmt.Errorf("can't encode stake: %v", err)
			}

			db.Set(path, data)
		}

		candidate.lock.RLock()
		updatesDirty := candidate.isUpdatesDirty
		candidate.lock.RUnlock()

		if updatesDirty {
			candidate.lock.Lock()
			candidate.isUpdatesDirty = false
			data, err := rlp.EncodeToBytes(candidate.updates)
			candidate.lock.Unlock()
			if err != nil {
				return fmt.Errorf("can't encode candidates updates: %v", err)
			}

			path := []byte{mainPrefix}
			path = append(path, candidate.idBytes()...)
			path = append(path, updatesPrefix)
			db.Set(path, data)
		}
	}

	return nil
}

// GetNewCandidates returns list of candidates that can be the new validators
// Skips offline candidates and candidates with stake less than minValidatorBipStake
// Result is sorted by candidates stakes and limited to valCount
func (c *Candidates) GetNewCandidates(valCount int) []*Candidate {
	var result []*Candidate

	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		if candidate.GetStatus() != CandidateStatusOnline {
			continue
		}

		if candidate.GetTotalBipStake().Cmp(minValidatorBipStake) == -1 {
			continue
		}

		result = append(result, candidate)
	}

	if len(result) > valCount {
		result = result[:valCount]
	}

	return result
}

// Create creates a new candidate with given params and adds it to state
func (c *Candidates) Create(ownerAddress, rewardAddress, controlAddress types.Address, pubkey types.Pubkey, commission uint32, block uint64, jailedUntil uint64) {
	candidate := &Candidate{
		ID:                       0,
		PubKey:                   pubkey,
		RewardAddress:            rewardAddress,
		OwnerAddress:             ownerAddress,
		ControlAddress:           controlAddress,
		Commission:               commission,
		LastEditCommissionHeight: block,
		JailedUntil:              jailedUntil,
		Status:                   CandidateStatusOffline,
		totalBipStake:            big.NewInt(0),
		stakes:                   [MaxDelegatorsPerCandidate]*stake{},
		isDirty:                  true,
		isTotalStakeDirty:        true,
	}

	candidate.setTmAddress()
	c.setToMap(pubkey, candidate)
}

// CreateWithID creates a new candidate with given params and adds it to state
// CreateWithID uses given ID to be associated with public key of a candidate
func (c *Candidates) CreateWithID(ownerAddress, rewardAddress, controlAddress types.Address, pubkey types.Pubkey, commission, id uint32, commissionEditHeight, jail uint64) {
	c.setPubKeyID(pubkey, id)
	c.Create(ownerAddress, rewardAddress, controlAddress, pubkey, commission, commissionEditHeight, jail)
}

// PunishByzantineCandidate finds candidate with given tmAddress and punishes it:
// 1. Subs 5% of each stake of a candidate
// 2. Unbond each stake of a candidate
func (c *Candidates) PunishByzantineCandidate(height uint64, tmAddress types.TmAddress) {
	candidate := c.GetCandidateByTendermintAddress(tmAddress)
	stakes := c.GetStakes(candidate.PubKey)

	for _, stake := range stakes {
		newValue := big.NewInt(0).Set(stake.Value)
		newValue.Mul(newValue, big.NewInt(95))
		newValue.Div(newValue, big.NewInt(100))

		slashed := big.NewInt(0).Set(stake.Value)
		slashed.Sub(slashed, newValue)

		if !stake.Coin.IsBaseCoin() {
			coin := c.bus.Coins().GetCoin(stake.Coin)
			ret := formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, slashed)

			c.bus.Coins().SubCoinVolume(coin.ID, slashed)
			c.bus.Coins().SubCoinReserve(coin.ID, ret)

			c.bus.App().AddTotalSlashed(ret)
		} else {
			c.bus.App().AddTotalSlashed(slashed)
		}

		c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(slashed))

		c.bus.Events().AddEvent(&eventsdb.SlashEvent{
			Address:         stake.Owner,
			Amount:          slashed.String(),
			Coin:            uint64(stake.Coin),
			ValidatorPubKey: candidate.PubKey,
		})

		c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(newValue))
		c.bus.FrozenFunds().AddFrozenFund(height+types.GetUnbondPeriod(), stake.Owner, &candidate.PubKey, candidate.ID, stake.Coin, newValue)
		stake.setValue(big.NewInt(0))
	}
}

// GetCandidateByTendermintAddress finds and returns candidate with given tendermint-address
func (c *Candidates) GetCandidateByTendermintAddress(address types.TmAddress) *Candidate {
	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		if candidate.GetTmAddress() == address {
			return candidate
		}
	}

	return nil
}

// RecalculateStakes recalculate stakes of all candidates:
// 1. Updates bip-values of each stake
// 2. Applies updates
func (c *Candidates) RecalculateStakes(height uint64) {
	c.recalculateStakes(height)
}

// RecalculateStakesV2 recalculate stakes of all candidates:
// 1. Updates bip-values of each stake
// 2. Applies updates
// 3. Removal of candidates over 100
func (c *Candidates) RecalculateStakesV2(height uint64) {
	c.recalculateStakes(height)
	candidates := c.getOrderedCandidatesLessID()
	if len(candidates) < 100 {
		return
	}

	for _, candidate := range candidates[100:] {
		c.DeleteCandidate(height, candidate)
	}
}

func (c *Candidates) recalculateStakes(height uint64) {
	coinsCache := newCoinsCache()
	c.lock.Lock()
	c.totalStakes.SetInt64(0)
	c.lock.Unlock()
	for _, candidate := range c.getOrderedCandidates() {
		stakes := &candidate.stakes
		for _, stake := range stakes {
			if stake == nil {
				continue
			}
			stake.setBipValue(c.calculateBipValue(stake.Coin, stake.Value, false, true, coinsCache))
		}

		// apply updates for existing stakes
		for _, update := range candidate.updates {
			stake := c.GetStakeOfAddress(candidate.PubKey, update.Owner, update.Coin)
			if stake != nil {
				stake.addValue(update.Value)
				update.setValue(big.NewInt(0))
				stake.setBipValue(c.calculateBipValue(stake.Coin, stake.Value, false, true, coinsCache))
			}
		}

		candidate.filterUpdates()
		for _, update := range candidate.updates {
			update.setBipValue(c.calculateBipValue(update.Coin, update.Value, false, true, coinsCache))
		}

		for _, update := range candidate.updates {
			// find and replace smallest stake
			index := -1
			smallestStake := big.NewInt(0)

			for i, stake := range stakes {
				if stake == nil {
					index = i
					smallestStake = big.NewInt(0)
					break
				}

				if index == -1 || smallestStake.Cmp(stake.BipValue) == 1 {
					smallestStake = big.NewInt(0).Set(stake.BipValue)
					index = i
				}
			}

			if smallestStake.Cmp(update.BipValue) == 1 {
				c.stakeKick(update.Owner, update.Value, update.Coin, candidate.PubKey, height)
				update.setValue(big.NewInt(0))
				continue
			}

			if stakes[index] != nil {
				c.stakeKick(stakes[index].Owner, stakes[index].Value, stakes[index].Coin, candidate.PubKey, height)
			}

			candidate.setStakeAtIndex(index, update, true)
		}

		candidate.clearUpdates()

		totalBipValue := big.NewInt(0)
		for _, stake := range stakes {
			if stake == nil {
				continue
			}
			totalBipValue.Add(totalBipValue, stake.BipValue)
		}

		candidate.setTotalBipStake(totalBipValue)

		c.lock.Lock()
		c.totalStakes.Add(c.totalStakes, totalBipValue)
		c.lock.Unlock()
	}
}

func (c *Candidates) stakeKick(owner types.Address, value *big.Int, coin types.CoinID, pubKey types.Pubkey, height uint64) {
	c.bus.WaitList().AddToWaitList(owner, pubKey, coin, value)
	c.bus.Events().AddEvent(&eventsdb.StakeKickEvent{
		Address:         owner,
		Amount:          value.String(),
		Coin:            uint64(coin),
		ValidatorPubKey: pubKey,
	})
	c.bus.Checker().AddCoin(coin, big.NewInt(0).Neg(value))
}

// Exists returns wherever a candidate with given public key exists
func (c *Candidates) Exists(pubkey types.Pubkey) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.existPubKey(pubkey)
}

func (c *Candidates) existPubKey(pubKey types.Pubkey) bool {
	_, exists := c.pubKeyIDs[pubKey]
	return exists
}

// IsBlockedPubKey returns if given public key is blacklisted
func (c *Candidates) IsBlockedPubKey(pubkey types.Pubkey) bool {
	return c.isBlocked(pubkey)
}

func (c *Candidates) isBlocked(pubKey types.Pubkey) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, exists := c.blockList[pubKey]
	return exists
}

// Count returns current amount of candidates
func (c *Candidates) Count() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.list)
}

// IsNewCandidateStakeSufficient determines if given stake is sufficient to create new candidate
func (c *Candidates) IsNewCandidateStakeSufficient(coin types.CoinID, stake *big.Int, limit int) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	bipValue := c.calculateBipValue(coin, stake, true, true, nil)
	var stakes []*big.Int

	for _, candidate := range c.list {
		stakes = append(stakes, big.NewInt(0).Set(candidate.totalBipStake))
	}

	sort.SliceStable(stakes, func(i, j int) bool {
		return stakes[i].Cmp(stakes[j]) == 1
	})

	for _, stake := range stakes[:limit] {
		if stake.Cmp(bipValue) == -1 {
			return true
		}
	}

	return false
}

// GetCandidate returns candidate by a public key
func (c *Candidates) GetCandidate(pubkey types.Pubkey) *Candidate {
	return c.getFromMap(pubkey)
}

// IsDelegatorStakeSufficient determines if given stake is sufficient to add it to a candidate
func (c *Candidates) IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinID, amount *big.Int) bool {
	stakes := c.GetStakes(pubkey)
	if len(stakes) < MaxDelegatorsPerCandidate {
		return true
	}

	stakeValue := c.calculateBipValue(coin, amount, true, true, nil)
	for _, stake := range stakes {
		if stakeValue.Cmp(stake.BipValue) == 1 || (stake.Owner == address && stake.Coin == coin) {
			return true
		}
	}

	return false
}

// IsDelegatorStakeAllowed determines if given stake is sufficient to add it to a candidate
func (c *Candidates) IsDelegatorStakeAllowed(address types.Address, pubkey types.Pubkey, coin types.CoinID, amount *big.Int) (low, b bool) {
	low = true
	old := big.NewInt(0)
	stakeValue := c.calculateBipValue(coin, amount, true, true, nil)

	stakes := c.GetStakes(pubkey)
	if len(stakes) < MaxDelegatorsPerCandidate {
		low = false
	} else {
		for _, stake := range stakes {
			if stakeValue.Cmp(stake.BipValue) == 1 || (stake.Owner == address && stake.Coin == coin) {
				old = stake.BipValue
				low = false
				break
			}
		}
	}

	if low {
		return true, false
	}

	diff := big.NewInt(0).Sub(stakeValue, old)
	newTotalStake := big.NewInt(0).Add(c.GetCandidate(pubkey).GetTotalBipStake(), diff)

	c.lock.RLock()
	defer c.lock.RUnlock()

	//if len(c.pubKeyIDs) < 4 {
	//	return false, false
	//}

	if c.bus.Validators().Count() < 4 {
		return false, false
	}

	newTotalStakes := big.NewInt(0).Add(c.totalStakes, diff)

	if big.NewInt(0).Div(newTotalStakes, newTotalStake).Cmp(big.NewInt(5)) == -1 {
		return false, true
	}

	return false, false
}

// Delegate adds a stake to a candidate
func (c *Candidates) Delegate(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int, bipValue *big.Int) {
	candidate := c.GetCandidate(pubkey)
	candidate.addUpdate(&stake{
		Owner:    address,
		Coin:     coin,
		Value:    big.NewInt(0).Set(value),
		BipValue: big.NewInt(0).Set(bipValue),
	})

	c.bus.Checker().AddCoin(coin, value)
}

// Edit edits a candidate
func (c *Candidates) Edit(pubkey types.Pubkey, rewardAddress types.Address, ownerAddress types.Address, controlAddress types.Address) {
	candidate := c.getFromMap(pubkey)
	candidate.setOwner(ownerAddress)
	candidate.setReward(rewardAddress)
	candidate.setControl(controlAddress)
}

// EditCommission edits a candidate commission
func (c *Candidates) EditCommission(pubkey types.Pubkey, commission uint32, height uint64) {
	candidate := c.getFromMap(pubkey)
	candidate.setCommission(commission, height)
}

// SetOnline sets candidate status to CandidateStatusOnline
func (c *Candidates) SetOnline(pubkey types.Pubkey) {
	c.getFromMap(pubkey).setStatus(CandidateStatusOnline)
}

// SetOffline sets candidate status to CandidateStatusOffline
func (c *Candidates) SetOffline(pubkey types.Pubkey) {
	c.getFromMap(pubkey).setStatus(CandidateStatusOffline)
}

// SubStake subs given value from delegator's stake
func (c *Candidates) SubStake(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int) {
	c.GetStakeOfAddress(pubkey, address, coin).subValue(value)
	c.bus.Checker().AddCoin(coin, big.NewInt(0).Neg(value))
}

// GetCandidates returns a list of all candidates
func (c *Candidates) GetCandidates() []*Candidate {
	return c.getOrderedCandidates()
}

// GetTotalStake calculates and returns total stake of a candidate
func (c *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	candidate := c.getFromMap(pubkey)
	candidate.lock.RLock()
	notLoaded := candidate.totalBipStake == nil
	candidate.lock.RUnlock()
	if notLoaded {
		path := []byte{mainPrefix}
		path = append(path, candidate.idBytes()...)
		path = append(path, totalStakePrefix)
		_, enc := c.immutableTree().Get(path)

		candidate.lock.Lock()
		candidate.totalBipStake = big.NewInt(0)
		if len(enc) != 0 {
			candidate.totalBipStake.SetBytes(enc)
		}
		candidate.lock.Unlock()
	}

	candidate.lock.RLock()
	defer candidate.lock.RUnlock()

	return big.NewInt(0).Set(candidate.totalBipStake)
}

// GetStakes returns list of stakes of candidate with given public key
func (c *Candidates) GetStakes(pubkey types.Pubkey) []*stake {
	candidate := c.GetCandidate(pubkey)

	var stakes []*stake
	for i := 0; i < MaxDelegatorsPerCandidate; i++ {
		stake := candidate.stakes[i]
		if stake == nil {
			continue
		}
		stakes = append(stakes, stake)
	}

	return stakes
}

// GetStakeOfAddress returns stake of address in given candidate and in given coin
func (c *Candidates) GetStakeOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinID) *stake {
	candidate := c.GetCandidate(pubkey)
	if candidate == nil {
		return nil
	}

	for _, stake := range candidate.stakes {
		if stake == nil {
			continue
		}

		if stake.Owner == address && stake.Coin == coin {
			return stake
		}
	}

	return nil
}

// GetStakeValueOfAddress returns stake VALUE of address in given candidate and in given coin
func (c *Candidates) GetStakeValueOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinID) *big.Int {
	stake := c.GetStakeOfAddress(pubkey, address, coin)
	if stake == nil {
		return nil
	}

	return stake.Value
}

// GetCandidateOwner returns candidate's owner address
func (c *Candidates) GetCandidateOwner(pubkey types.Pubkey) types.Address {
	return c.getFromMap(pubkey).OwnerAddress
}

// GetCandidateControl returns candidate's control address
func (c *Candidates) GetCandidateControl(pubkey types.Pubkey) types.Address {
	return c.getFromMap(pubkey).ControlAddress
}

// LoadCandidates loads only list of candidates (for read)
func (c *Candidates) LoadCandidates() {
	if c.checkAndSetLoaded() {
		return
	}

	_ = c.loadCandidatesList()
}

// LoadCandidatesDeliver loads full info about candidates (for edit)
func (c *Candidates) LoadCandidatesDeliver() {
	if c.checkAndSetLoaded() {
		return
	}

	c.maxID = c.loadCandidatesList()

	_, blockListEnc := c.immutableTree().Get([]byte{blockListPrefix})
	if len(blockListEnc) != 0 {
		var blockList []types.Pubkey
		if err := rlp.DecodeBytes(blockListEnc, &blockList); err != nil {
			panic(fmt.Sprintf("failed to decode candidates block list: %s", err))
		}

		blockListMap := map[types.Pubkey]struct{}{}
		for _, pubkey := range blockList {
			blockListMap[pubkey] = struct{}{}
		}
		c.setBlockList(blockListMap)
	}

	_, valueMaxID := c.immutableTree().Get([]byte{maxIDPrefix})
	if len(valueMaxID) != 0 {
		c.maxID = binary.LittleEndian.Uint32(valueMaxID)
	}

}

func (c *Candidates) loadCandidatesList() (maxID uint32) {
	_, pubIDenc := c.immutableTree().Get([]byte{pubKeyIDPrefix})
	if len(pubIDenc) != 0 {
		var pubIDs []pubkeyID
		if err := rlp.DecodeBytes(pubIDenc, &pubIDs); err != nil {
			panic(fmt.Sprintf("failed to decode candidates: %s", err))
		}

		pubKeyIDs := map[types.Pubkey]uint32{}
		for _, v := range pubIDs {
			pubKeyIDs[v.PubKey] = v.ID
			if v.ID > maxID {
				maxID = v.ID
			}
		}
		c.setPubKeyIDs(pubKeyIDs)
	}

	path := []byte{mainPrefix}
	_, enc := c.immutableTree().Get(path)
	if len(enc) != 0 {
		var candidates []*Candidate
		if err := rlp.DecodeBytes(enc, &candidates); err != nil {
			panic(fmt.Sprintf("failed to decode candidates: %s", err))
		}
		for _, candidate := range candidates {
			// load total stake
			path = append([]byte{mainPrefix}, candidate.idBytes()...)
			path = append(path, totalStakePrefix)
			_, enc = c.immutableTree().Get(path)

			if len(enc) == 0 {
				candidate.totalBipStake = big.NewInt(0)
			} else {
				candidate.totalBipStake = big.NewInt(0).SetBytes(enc)
			}

			candidate.setTmAddress()
			c.setToMap(candidate.PubKey, candidate)
		}
	}

	return maxID
}

func (c *Candidates) checkAndSetLoaded() bool {
	c.lock.RLock()
	if c.loaded {
		c.lock.RUnlock()
		return true
	}
	c.lock.RUnlock()

	c.lock.Lock()
	c.loaded = true
	c.lock.Unlock()
	return false
}

func (c *Candidates) TotalStakes() *big.Int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return big.NewInt(0).Set(c.totalStakes)
}

// LoadStakes loads all stakes of candidates
func (c *Candidates) LoadStakes() {
	//c.lock.Lock()
	//defer c.lock.Unlock()

	c.loadStakes()
}

// loadStakes loads all stakes of candidates
func (c *Candidates) loadStakes() {
	c.totalStakes.SetInt64(0)
	for pubkey := range c.pubKeyIDs {
		c.totalStakes.Add(c.totalStakes, c.LoadStakesOfCandidate(pubkey))
	}
}

func (c *Candidates) calculateBipValue(coinID types.CoinID, amount *big.Int, includeSelf, includeUpdates bool, coinsCache *coinsCache) *big.Int {
	if coinID.IsBaseCoin() {
		return big.NewInt(0).Set(amount)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0)
	}

	coin := c.bus.Coins().GetCoin(coinID)

	totalDelegatedBasecoin, totalDelegatedValue := big.NewInt(0), big.NewInt(0)
	if coinsCache.Exists(coinID) {
		totalDelegatedBasecoin, totalDelegatedValue = coinsCache.Get(coinID)
	}

	if includeSelf {
		totalDelegatedValue.Add(totalDelegatedValue, amount)
	}

	if !coinsCache.Exists(coinID) {
		candidates := c.GetCandidates()
		for _, candidate := range candidates {
			for _, stake := range candidate.stakes {
				if stake != nil && stake.Coin == coinID {
					totalDelegatedValue.Add(totalDelegatedValue, stake.Value)
				}
			}

			if includeUpdates {
				for _, update := range candidate.updates {
					if update.Coin == coinID {
						totalDelegatedValue.Add(totalDelegatedValue, update.Value)
					}
				}
			}
		}

		nonLockedSupply := big.NewInt(0).Sub(coin.Volume, totalDelegatedValue)
		totalDelegatedBasecoin = big.NewInt(0).Sub(coin.Reserve, formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, nonLockedSupply))
		coinsCache.Set(coinID, totalDelegatedBasecoin, totalDelegatedValue)
	}

	return big.NewInt(0).Div(big.NewInt(0).Mul(totalDelegatedBasecoin, amount), totalDelegatedValue)
}

// Punish punished a candidate with given tendermint-address
func (c *Candidates) Punish(height uint64, address types.TmAddress) {
	candidate := c.GetCandidateByTendermintAddress(address)
	jailUntil := height + types.GetJailPeriod()
	candidate.jainUntil(jailUntil)
	c.bus.Events().AddEvent(&eventsdb.JailEvent{ValidatorPubKey: candidate.PubKey, JailedUntil: jailUntil})
}

// SetStakes Sets stakes and updates of a candidate. Used in Import.
func (c *Candidates) SetStakes(pubkey types.Pubkey, stakes []types.Stake, updates []types.Stake) {
	candidate := c.GetCandidate(pubkey)
	candidate.stakesCount = len(stakes)

	for _, u := range updates {
		coin := types.CoinID(u.Coin)
		value := helpers.StringToBigInt(u.Value)
		candidate.addUpdate(&stake{
			Owner:    u.Owner,
			Coin:     coin,
			Value:    value,
			BipValue: helpers.StringToBigInt(u.BipValue),
		})
		c.bus.Checker().AddCoin(coin, value)
	}

	count := len(stakes)
	if count > MaxDelegatorsPerCandidate {
		count = MaxDelegatorsPerCandidate

		for _, u := range stakes[1000:] {
			coin := types.CoinID(u.Coin)
			value := helpers.StringToBigInt(u.Value)
			candidate.addUpdate(&stake{
				Owner:    u.Owner,
				Coin:     coin,
				Value:    value,
				BipValue: helpers.StringToBigInt(u.BipValue),
			})
			c.bus.Checker().AddCoin(coin, value)
		}
	}

	for i, s := range stakes[:count] {
		coin := types.CoinID(s.Coin)
		value := helpers.StringToBigInt(s.Value)
		candidate.stakes[i] = &stake{
			Owner:    s.Owner,
			Coin:     coin,
			Value:    value,
			BipValue: helpers.StringToBigInt(s.BipValue),
			markDirty: func(index int) {
				candidate.dirtyStakes[index] = true
			},
			index: i,
		}
		c.bus.Checker().AddCoin(coin, value)
		candidate.stakes[i].markDirty(i)
	}
}

// Export exports all data to the given state
func (c *Candidates) Export(state *types.AppState) {
	c.LoadCandidatesDeliver()
	c.loadStakes()

	candidates := c.GetCandidates()
	state.Candidates = make([]types.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidateStakes := c.GetStakes(candidate.PubKey)
		stakes := make([]types.Stake, len(candidateStakes))
		for i, s := range candidateStakes {
			stakes[i] = types.Stake{
				Owner:    s.Owner,
				Coin:     uint64(s.Coin),
				Value:    s.Value.String(),
				BipValue: s.BipValue.String(),
			}
		}

		updates := make([]types.Stake, len(candidate.updates))
		for i, u := range candidate.updates {
			updates[i] = types.Stake{
				Owner:    u.Owner,
				Coin:     uint64(u.Coin),
				Value:    u.Value.String(),
				BipValue: u.BipValue.String(),
			}
		}

		state.Candidates = append(state.Candidates, types.Candidate{
			ID:                       uint64(candidate.ID),
			RewardAddress:            candidate.RewardAddress,
			OwnerAddress:             candidate.OwnerAddress,
			ControlAddress:           candidate.ControlAddress,
			TotalBipStake:            candidate.GetTotalBipStake().String(),
			PubKey:                   candidate.PubKey,
			Commission:               uint64(candidate.Commission),
			Status:                   uint64(candidate.Status),
			Updates:                  updates,
			Stakes:                   stakes,
			JailedUntil:              candidate.JailedUntil,
			LastEditCommissionHeight: candidate.LastEditCommissionHeight,
		})
	}

	for pubkey := range c.blockList {
		state.BlockListCandidates = append(state.BlockListCandidates, pubkey)
	}
	sort.SliceStable(state.BlockListCandidates, func(i, j int) bool {
		return bytes.Compare(state.BlockListCandidates[i].Bytes(), state.BlockListCandidates[j].Bytes()) == 1
	})

	c.loadDeletedCandidates()
	for _, c := range c.deletedCandidates {
		state.DeletedCandidates = append(state.DeletedCandidates, types.DeletedCandidate{
			ID:     uint64(c.ID),
			PubKey: c.PubKey,
		})
	}
	sort.SliceStable(state.DeletedCandidates, func(i, j int) bool {
		if state.DeletedCandidates[i].ID == state.DeletedCandidates[j].ID {
			return state.DeletedCandidates[i].PubKey.String() > state.DeletedCandidates[j].PubKey.String()
		}
		return state.DeletedCandidates[i].ID < state.DeletedCandidates[j].ID
	})
}

func (c *Candidates) DeletedCandidates() (result []*deletedID) {
	c.muDeletedCandidates.Lock()
	defer c.muDeletedCandidates.Unlock()

	c.loadDeletedCandidates()
	for _, c := range c.deletedCandidates {
		if c.PubKey.String() == invalidIDCandidate {
			continue
		}
		result = append(result, &deletedID{
			ID:     c.ID,
			PubKey: c.PubKey,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ID == result[j].ID {
			return result[i].PubKey.String() > result[j].PubKey.String()
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// Deprecated: Use getOrderedCandidatesLessID
func (c *Candidates) getOrderedCandidates() []*Candidate {
	c.lock.RLock()
	var candidates []*Candidate
	for _, candidate := range c.list {
		candidate.lock.RLock()
		candidates = append(candidates, candidate)
		candidate.lock.RUnlock()
	}
	c.lock.RUnlock()

	sort.SliceStable(candidates, func(i, j int) bool {
		cmp := candidates[i].GetTotalBipStake().Cmp(candidates[j].GetTotalBipStake())
		if cmp == 0 {
			return candidates[i].ID > candidates[j].ID
		}
		return cmp == 1
	})

	return candidates
}

func (c *Candidates) getOrderedCandidatesLessID() []*Candidate {
	c.lock.RLock()
	var candidates []*Candidate
	for _, candidate := range c.list {
		candidate.lock.RLock()
		candidates = append(candidates, candidate)
		candidate.lock.RUnlock()
	}
	c.lock.RUnlock()

	sort.SliceStable(candidates, func(i, j int) bool {
		cmp := candidates[i].GetTotalBipStake().Cmp(candidates[j].GetTotalBipStake())
		if cmp == 0 {
			return candidates[i].ID < candidates[j].ID
		}
		return cmp == 1
	})

	return candidates
}

func (c *Candidates) getFromMap(pubkey types.Pubkey) *Candidate {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[c.id(pubkey)]
}

func (c *Candidates) setToMap(pubkey types.Pubkey, model *Candidate) {
	model.lock.RLock()
	id := model.ID
	model.lock.RUnlock()
	if id == 0 {
		id = c.getOrNewID(pubkey)

		model.lock.Lock()
		model.ID = id
		model.lock.Unlock()
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[id] = model
}

func (c *Candidates) setBlockList(blockList map[types.Pubkey]struct{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.blockList = blockList
}

func (c *Candidates) setPubKeyIDs(list map[types.Pubkey]uint32) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.pubKeyIDs = list
}

// SetTotalStake sets candidate's total bip stake. Used in Import.
func (c *Candidates) SetTotalStake(pubkey types.Pubkey, stake *big.Int) {
	c.GetCandidate(pubkey).setTotalBipStake(stake)
}

// LoadStakesOfCandidate loads stakes of given candidate from disk
func (c *Candidates) LoadStakesOfCandidate(pubkey types.Pubkey) *big.Int {
	candidate := c.GetCandidate(pubkey)

	// load stakes
	stakesCount := 0
	for index := 0; index < MaxDelegatorsPerCandidate; index++ {
		path := []byte{mainPrefix}
		path = append(path, candidate.idBytes()...)
		path = append(path, stakesPrefix)
		path = append(path, big.NewInt(int64(index)).Bytes()...)
		_, enc := c.immutableTree().Get(path)
		if len(enc) == 0 {
			candidate.lock.Lock()
			candidate.stakes[index] = nil
			candidate.lock.Unlock()
			continue
		}
		stake := &stake{}
		if err := rlp.DecodeBytes(enc, stake); err != nil {
			panic(fmt.Sprintf("failed to decode stake: %s", err))
		}
		candidate.setStakeAtIndex(index, stake, false)

		stakesCount++
	}

	candidate.lock.Lock()
	candidate.stakesCount = stakesCount
	candidate.lock.Unlock()

	// load updates
	path := []byte{mainPrefix}
	path = append(path, candidate.idBytes()...)
	path = append(path, updatesPrefix)
	_, enc := c.immutableTree().Get(path)

	candidate.lock.Lock()
	if len(enc) == 0 {
		candidate.updates = nil
	} else {
		var updates []*stake
		if err := rlp.DecodeBytes(enc, &updates); err != nil {
			panic(fmt.Sprintf("failed to decode updated: %s", err))
		}

		for _, update := range updates {
			update.markDirty = (func(candidate *Candidate) func(int) {
				return func(i int) {
					candidate.lock.Lock()
					defer candidate.lock.Unlock()
					candidate.isUpdatesDirty = true
				}
			})(candidate)
		}

		candidate.updates = updates
	}
	candidate.lock.Unlock()

	// load total stake
	path = append([]byte{mainPrefix}, candidate.idBytes()...)
	path = append(path, totalStakePrefix)
	_, enc = c.immutableTree().Get(path)

	candidate.lock.Lock()
	if len(enc) == 0 {
		candidate.totalBipStake = big.NewInt(0)
	} else {
		candidate.totalBipStake = big.NewInt(0).SetBytes(enc)
	}
	candidate.lock.Unlock()

	candidate.setTmAddress()
	c.setToMap(candidate.PubKey, candidate)

	return big.NewInt(0).Set(candidate.totalBipStake)
}

// ChangePubKey change public key of a candidate from old to new
func (c *Candidates) ChangePubKey(old types.Pubkey, new types.Pubkey) {
	if c.isBlocked(new) {
		panic("Candidate with such public key (" + new.String() + ") exists in block list")
	}

	c.getFromMap(old).setPublicKey(new)
	c.setBlockPubKey(old)

	c.lock.RLock()
	id := c.pubKeyIDs[old]
	c.lock.RUnlock()

	c.setPubKeyID(new, id)

	c.lock.Lock()
	delete(c.pubKeyIDs, old)
	c.isChangedPublicKeys = true
	c.lock.Unlock()
}

func (c *Candidates) getOrNewID(pubKey types.Pubkey) uint32 {
	c.lock.RLock()
	id := c.id(pubKey)
	c.lock.RUnlock()
	if id != 0 {
		return id
	}

	c.lock.Lock()
	c.isDirty = true
	c.maxID++
	id = c.maxID
	c.lock.Unlock()

	c.setPubKeyID(pubKey, id)
	return id
}

func (c *Candidates) id(pubKey types.Pubkey) uint32 {
	id, ok := c.pubKeyIDs[pubKey]
	if ok {
		return id
	}

	c.muDeletedCandidates.Lock()
	defer c.muDeletedCandidates.Unlock()

	c.loadDeletedCandidates()
	deleted := c.deletedCandidates[pubKey]
	if deleted == nil {
		return 0
	}
	return deleted.ID
}

// ID returns an id of candidate by it's public key
func (c *Candidates) ID(pubKey types.Pubkey) uint32 {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.id(pubKey)
}

const invalidIDCandidate = "Mp0e11415ef24919557dcea4890d9e8aa26dc31ef8e77c6343114e1180ebeccde3"

// PubKey returns a public key of candidate by it's ID
func (c *Candidates) PubKey(id uint32) types.Pubkey {
	c.lock.RLock()
	defer c.lock.RUnlock()

	candidate, ok := c.list[id]
	if !ok {
		c.muDeletedCandidates.Lock()
		defer c.muDeletedCandidates.Unlock()

		c.loadDeletedCandidates()
		for pubkey, d := range c.deletedCandidates {
			if d.ID == id && d.PubKey.String() != invalidIDCandidate {
				return pubkey
			}
		}
		return types.Pubkey{}
	}

	candidate.lock.RLock()
	defer candidate.lock.RUnlock()

	return candidate.PubKey
}

func (c *Candidates) setPubKeyID(pubkey types.Pubkey, id uint32) {
	if id == 0 {
		panic("public key of candidate cannot be equal 0")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.maxID < id {
		c.maxID = id
	}

	c.pubKeyIDs[pubkey] = id
	c.isDirty = true
}

func (c *Candidates) setBlockPubKey(p types.Pubkey) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.blockList[p] = struct{}{}
	c.isDirty = true
}

// AddToBlockPubKey blacklists given publickey
func (c *Candidates) AddToBlockPubKey(p types.Pubkey) {
	c.setBlockPubKey(p)
}

func (c *Candidates) maxIDBytes() []byte {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, c.maxID)
	return bs
}

func (c *Candidates) DeleteCandidate(height uint64, candidate *Candidate) {
	if c.bus.Validators().IsValidator(candidate.PubKey) {
		return
	}

	c.AddToBlockPubKey(candidate.PubKey)
	c.bus.Events().AddEvent(&eventsdb.RemoveCandidateEvent{CandidatePubKey: candidate.PubKey})

	for _, s := range candidate.stakes {
		if s == nil {
			continue
		}
		c.bus.Events().AddEvent(&eventsdb.UnbondEvent{
			Address:         s.Owner,
			Amount:          s.Value.String(),
			Coin:            uint64(s.Coin),
			ValidatorPubKey: &candidate.PubKey,
		})
		c.bus.FrozenFunds().AddFrozenFund(height+types.GetUnbondPeriod(), s.Owner, &candidate.PubKey, candidate.ID, s.Coin, s.Value)
		c.bus.Checker().AddCoin(s.Coin, big.NewInt(0).Neg(s.Value))
		s.setValue(big.NewInt(0))
	}
	for _, u := range candidate.updates {
		if u == nil {
			continue
		}
		c.bus.Events().AddEvent(&eventsdb.UnbondEvent{
			Address:         u.Owner,
			Amount:          u.Value.String(),
			Coin:            uint64(u.Coin),
			ValidatorPubKey: &candidate.PubKey,
		})
		c.bus.FrozenFunds().AddFrozenFund(height+types.GetUnbondPeriod(), u.Owner, &candidate.PubKey, candidate.ID, u.Coin, u.Value)
		c.bus.Checker().AddCoin(u.Coin, big.NewInt(0).Neg(u.Value))
		u.setValue(big.NewInt(0))
	}

	c.lock.Lock()
	c.deleteCandidateFromList(candidate)
	c.totalStakes.Sub(c.totalStakes, candidate.totalBipStake)
	c.lock.Unlock()
}

func (c *Candidates) deleteCandidateFromList(candidate *Candidate) {
	c.muDeletedCandidates.Lock()
	defer c.muDeletedCandidates.Unlock()

	c.loadDeletedCandidates()
	c.deletedCandidates[candidate.PubKey] = &deletedID{
		ID:      candidate.ID,
		PubKey:  candidate.PubKey,
		isDirty: true,
	}
	c.dirtyDeletedCandidates = true

	c.isDirty = true
	delete(c.pubKeyIDs, candidate.PubKey)

	delete(c.list, candidate.ID)
}

func (c *Candidates) loadDeletedCandidates() {
	if len(c.deletedCandidates) != 0 {
		return
	}

	c.deletedCandidates = make(map[types.Pubkey]*deletedID)
	_, data := c.immutableTree().Get([]byte{deleteCandidatesPrefix})
	if len(data) == 0 {
		return
	}

	var list []*deletedID
	err := rlp.DecodeBytes(data, &list)
	if err != nil {
		panic(fmt.Errorf("can't decode deleted candidates: %v", err))
	}

	for _, id := range list {
		c.deletedCandidates[id.PubKey] = id
	}
}

func (c *Candidates) FixStakesAfter10509400() *big.Int {
	var correctStakes = map[string]map[string]string{"Mp000060164538dd9f298e3faddb0ce348fbded8349c90db3f0573325ba5a20099": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "33510969217991442153900", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "33408765612618024913051"}, "Mp0030667774229676f4641f9dc1d90ac47c3b8f1297fac16bed5950f863c20030": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "1397458333959849258126", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "1394137769636730129853"}, "Mp01cc99ae5a349ecaeef187dcbb12816bf2b3d8eae80f654034b21213aa445b2c": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "122236583817092774239266", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "121852818315430984601480"}, "Mp03478aae43a1a660573fab0763ae44492cdaf8deffc3fcbcc844acd67dfb2db6": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "36489144992929948883753", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "36366090138829703729864"}, "Mp03aae1d17c2e4ef6905adb44518be95aa0317cb7cbbe00236a36aab2e07c5b2c": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "11350715581955065344122", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "11315380800345116798827"}, "Mp120c15e48aed0ac866a1a918bd367cfa31909a6b09f328a18bd18f32edef2be8": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "59674171681036550468720", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "59478585952151012246159"}, "Mp1ada5ac409b965623bf6a4320260190038ae27230abfb5ebc9158280cdffffff": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "40715047444152333375286", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "40581105994884729017480"}, "Mp31d08d6f64f7a8a528ed2df77de2a02e4d8cefae93c771eb0b7de97322901215": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "17827247136406525739291", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "17773749366383914419151"}, "Mp3b6e2632cd0c91ef96ea4f4a16f554ff1b4dd41324fd421a0161947c50603b9b": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "44899967318280509746347", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "44760630408246579870429"}, "Mp3ff4490148c0cf42a9780dbb24a08e012d30d6f371021d28f8e4d3a8e2d6d9cf": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "3476934078882983241629", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "3465527934463811596493"}, "Mp46d3d6afe0084fcf530b03d1f4427e516a1cb4ec542640bcbc84c2c4b4f53c13": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "140947445307816038395793", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "140485616519116360309551"}, "Mp47f5c19c3ac5d66c960f36aa1a2d74ec8127f96308b67054332c988ee9eafaf0": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "53595138386224727399730", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "53428106357227266514222"}, "Mp4881ad167ca5fb5886322841f992d68aed894ffcb58abc080e8ad3b156f1045b": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "242478737788852874937553", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "241644884104251511311737"}, "Mp4881fc597a3921a690d3a9f95897ae42698992d6e9dfa57173209c6320a36ae8": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "8284619252877106502232", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "8258867378276467997585"}, "Mp5e3e1da62c7eabd9d8d168a36a92727fc1970a54ec61eadd285d4199c41191d7": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "14330419815717211490036", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "14284247551990214535705"}, "Mp629b5528f09d1c74a83d18414f2e4263e14850c47a3fac3f855f200111111111": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "410847584516443595057425", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "409438887362638749528200"}, "Mp65758496f8a5d626cac77f5a38894beae0050fdef862da201964cd2fd8111111": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "59240647446041884589682", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "59056162007039988072980"}, "Mp6ff5bd93c430e06e70ef9aff1eec8c617e03464bef2545180b9de1757fb578ac": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "1064278153008371196619", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "1060866143872181753974"}, "Mp7373d6d5a4ec18d87d766e5b4c6d3a0c94b357c4460a9ed377bf5d09fcd77373": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "8139977794481804031787", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "8115095298417052033113"}, "Mp7771ad434bd6df3519ee5bd5331e66815263a3d8afbdd6768003ff7f790db6bc": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "4786423506308566885", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "4784167411444000855"}, "Mp7779ec1c6492e7c71a36f4009d7ee5a43fed2fe4048882b8a099e748869f0777": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "10270355665428688472884", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "10236819291495837950276"}, "Mp77f71834108e9b5e65237a39263631b4f99a9d58437a1385c930c13ee1d4e2a6": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "21658501534219455755192", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "21584220501120727823524"}, "Mp7979c5a8d974e0c8c99f568b91454853f0c48fbbe68ac39d40efd0b330f30885": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "18956447281849717829149", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "18895438864841794085442"}, "Mp831167f2a292afb0f523ca451e181c6c1d2b75991a9f11ead019012841699739": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "210315706450222123970", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "209968785731925814855"}, "Mp83d44bdc43d39df6fba726cdfea81444fcd07b1e1cb9caf61835a4f31031fe9d": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "4703803980156876541275", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "4687857553613951468619"}, "Mp88883bcd4e9a4eeb7f5a7b7d1f4c02ac0fadc268824694fcccc84e39f4e08888": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "10154431165301281635317", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "10120666415083474476662"}, "Mp8fc37a738c22f75de69d057302c82a583294650d9aa25408cca6052d6d5272ee": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "2648999810684953089608", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "2639882872010359087603"}, "Mp95b76c6893dc28a34f005b9708bac59eae238232ef86798d672387bbb849bd22": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "6559073248119206936596", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "6538799947073029404432"}, "Mp97a5a64cbbf65a3474beb0a0cbf8eb054b14a59623f1416580099f4433333333": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "37505677161780553806034", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "37380118944053064240364"}, "Mp9fdd4c54b5a1ea0c014c8a8cf012cdb756630d0c35728c8659d6f69d2d00ffa5": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "2458651610276579346815", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "2451004079265544401877"}, "Mpa6a3137cd3a7a2442295911f0ccbfbfd050f34c17d7e3edc56f78464b880c649": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "12754869705378793678501", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "12712366362959729672751"}, "Mpaaaaa16ebd6af229b4cfc02c3ab40bd25c1051c3aa2120f07d08c1bd01777777": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "332575858670917958871169", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "331531644370337584703170"}, "Mpadeb65fa666a4a8e033d20c1a48d09b3f024eb8756da7f7a21e0932582222222": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "94145720673344281867639", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "93825349513615687726169"}, "Mpaf167d3b1d37733928b28615f4168e79de3d45529945a3630a10e0767e03b6e3": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "2221286644566503660875", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "2217641859368552681743"}, "Mpb451f898f2d5e054b9edc4b06c2cbcf1c318348593a05cae32565ec665758207": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "5724623743459580565593", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "5709967969656718374421"}, "Mpb5e24c1a451f26d6663b50f09bcf40c1e62f44d8bb81ad72d0252a81fe856ac4": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "4683711291880656684085", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "4669636350217621414848"}, "Mpb7a29db246fb2ce80c4f44549875d629d7fc0f02466f2b8a278510e742aa044e": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "14109872528042662790083", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "14064048777019753144531"}, "Mpc6df0dd44cae3cb49ee0ef8bd78750b961e64e231e6dfdc22fbd8e0c476d66d2": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "23875647121955165456115", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "23801800432186577215825"}, "Mpc9fc1052e075054cdbfb6443a6d14d97be9d4f19a10505c4323b52a78ca4bb18": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "28245183247759980423291", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "28157382419702975072432"}, "Mpd7a9eae76d3e98145b6b897c00c407b42ae8c42f937527ebdcd0239a4d88c0a5": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "25254200615429064426676", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "25171437563717707087830"}, "Mpe782c9a2c62f085f4d1bedf307de525b13226c20c597e66b0cf246a061f31b2d": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "11475971864115562515281", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "11441879881376464316452"}, "Mpeee9614b63a7ed6370ccd1fa227222fa30d6106770145c55bd4b482b88888888": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "79319104895478577411477", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "79072605151822268619155"}, "Mpf64c5956078cd8e5db5b3e7ccfc4e2f9b596fa3801eb3dbfef691623df972279": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "21005930049612665889858", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "20933576115781198383101"}, "Mpf979659e45f25defab5ef8624f8060e27b1440157f86779b12b98e7e293136cb": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "15396370097976565180343", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "15348446969626047200130"}, "Mpfe176f944623a8ca9f409a62f0ea3ca75c1cf8d89970adf9384fc9ae8d77fa0b": {"Mx688568d9d70c57e71d0b9de6480afb0d317f885c": "134569605678669021917879", "Mx7f0fc21d932f38ca9444f61703174569066cfa50": "134128024316356823282029"}}

	moreStakes := big.NewInt(0)
	candidates := c.getOrderedCandidatesLessID()
	for _, candidate := range candidates {
		moreCandidateStake := big.NewInt(0)
		var gotDAO, gotDEV bool
		for _, stake := range candidate.stakes {
			if stake == nil {
				continue
			}
			if stake.Owner == dao.Address {
				value := helpers.StringToBigInt(correctStakes[candidate.PubKey.String()][stake.Owner.String()])

				moreCandidateStake.Add(moreCandidateStake, big.NewInt(0).Sub(stake.Value, value))

				stake.setValue(value)
				stake.setBipValue(value)
				log.Println("fixStake", candidate.PubKey.String(), stake.Owner.String(), value)

				gotDAO = true
			}
			if stake.Owner == developers.Address {
				value := helpers.StringToBigInt(correctStakes[candidate.PubKey.String()][stake.Owner.String()])

				moreCandidateStake.Add(moreCandidateStake, big.NewInt(0).Sub(stake.Value, value))

				stake.setValue(value)
				stake.setBipValue(value)
				log.Println("fixStake", candidate.PubKey.String(), stake.Owner.String(), value)

				gotDEV = true
			}
			if gotDEV && gotDAO {
				break
			}
		}
		fixTotalStake := big.NewInt(0).Sub(candidate.totalBipStake, moreCandidateStake)
		log.Println("fixTotalStake", candidate.PubKey.String(), fixTotalStake)
		candidate.setTotalBipStake(fixTotalStake)
		moreStakes.Add(moreStakes, moreCandidateStake)
	}

	return moreStakes
}
