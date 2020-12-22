package candidates

import (
	"bytes"
	"encoding/binary"
	"fmt"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"

	"math/big"
	"sort"
	"sync"
)

// Common constants
const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02

	UnbondPeriod              = 518400
	MaxDelegatorsPerCandidate = 1000
)

const (
	mainPrefix       = 'c'
	pubKeyIDPrefix   = mainPrefix + 'p'
	blockListPrefix  = mainPrefix + 'b'
	maxIDPrefix      = mainPrefix + 'i'
	stakesPrefix     = 's'
	totalStakePrefix = 't'
	updatesPrefix    = 'u'
)

var (
	minValidatorBipStake = helpers.BipToPip(big.NewInt(1000))
)

// RCandidates interface represents Candidates state
type RCandidates interface {
	Export(state *types.AppState)
	Exists(pubkey types.Pubkey) bool
	IsBlockedPubKey(pubkey types.Pubkey) bool
	PubKey(id uint32) types.Pubkey
	Count() int
	IsNewCandidateStakeSufficient(coin types.CoinID, stake *big.Int, limit int) bool
	IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinID, amount *big.Int) bool
	GetStakeValueOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinID) *big.Int
	GetCandidateOwner(pubkey types.Pubkey) types.Address
	GetCandidateControl(pubkey types.Pubkey) types.Address
	GetTotalStake(pubkey types.Pubkey) *big.Int
	LoadCandidates()
	LoadStakesOfCandidate(pubkey types.Pubkey)
	GetCandidate(pubkey types.Pubkey) *Candidate
	LoadStakes()
	GetCandidates() []*Candidate
	GetStakes(pubkey types.Pubkey) []*stake
}

// Candidates struct is a store of Candidates state
type Candidates struct {
	list map[uint32]*Candidate

	isDirty   bool
	blockList map[types.Pubkey]struct{}
	pubKeyIDs map[types.Pubkey]uint32
	maxID     uint32

	iavl tree.MTree
	bus  *bus.Bus

	lock                sync.RWMutex
	loaded              bool
	isChangedPublicKeys bool
}

func (c *Candidates) IsChangedPublicKeys() bool {
	return c.isChangedPublicKeys
}
func (c *Candidates) ResetIsChangedPublicKeys() {
	c.isChangedPublicKeys = false
}

// NewCandidates returns newly created Candidates state with a given bus and iavl
func NewCandidates(bus *bus.Bus, iavl tree.MTree) (*Candidates, error) {
	candidates := &Candidates{
		iavl:      iavl,
		bus:       bus,
		blockList: map[types.Pubkey]struct{}{},
		pubKeyIDs: map[types.Pubkey]uint32{},
		list:      map[uint32]*Candidate{},
	}
	candidates.bus.SetCandidates(NewBus(candidates))

	return candidates, nil
}

// Commit writes changes to iavl, may return an error
func (c *Candidates) Commit() error {
	keys := c.getOrderedCandidates()

	hasDirty := false
	for _, pubkey := range keys {
		if c.getFromMap(pubkey).isDirty {
			hasDirty = true
			break
		}
	}

	if hasDirty {
		var candidates []*Candidate
		for _, key := range keys {
			candidates = append(candidates, c.getFromMap(key))
		}
		data, err := rlp.EncodeToBytes(candidates)
		if err != nil {
			return fmt.Errorf("can't encode candidates: %v", err)
		}

		path := []byte{mainPrefix}
		c.iavl.Set(path, data)
	}

	if c.isDirty {
		c.isDirty = false
		var pubIDs []pubkeyID
		for pk, v := range c.pubKeyIDs {
			pubIDs = append(pubIDs, pubkeyID{
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

		c.iavl.Set([]byte{pubKeyIDPrefix}, pubIDData)

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
		c.iavl.Set([]byte{blockListPrefix}, blockListData)

		c.iavl.Set([]byte{maxIDPrefix}, c.maxIDBytes())
	}

	for _, pubkey := range keys {
		candidate := c.getFromMap(pubkey)
		candidate.isDirty = false

		if candidate.isTotalStakeDirty {
			path := []byte{mainPrefix}
			path = append(path, candidate.idBytes()...)
			path = append(path, totalStakePrefix)
			c.iavl.Set(path, candidate.totalBipStake.Bytes())
			candidate.isTotalStakeDirty = false
		}

		for index, stake := range candidate.stakes {
			if !candidate.dirtyStakes[index] {
				continue
			}

			candidate.dirtyStakes[index] = false

			path := []byte{mainPrefix}
			path = append(path, candidate.idBytes()...)
			path = append(path, stakesPrefix)
			path = append(path, []byte(fmt.Sprintf("%d", index))...) // todo big.NewInt(index).Bytes()

			if stake == nil || stake.Value.Sign() == 0 {
				c.iavl.Remove(path)
				candidate.stakes[index] = nil
				continue
			}

			data, err := rlp.EncodeToBytes(stake)
			if err != nil {
				return fmt.Errorf("can't encode stake: %v", err)
			}

			c.iavl.Set(path, data)
		}

		if candidate.isUpdatesDirty {
			data, err := rlp.EncodeToBytes(candidate.updates)
			if err != nil {
				return fmt.Errorf("can't encode candidates updates: %v", err)
			}

			path := []byte{mainPrefix}
			path = append(path, candidate.idBytes()...)
			path = append(path, updatesPrefix)
			c.iavl.Set(path, data)
			candidate.isUpdatesDirty = false
		}
	}

	return nil
}

// GetNewCandidates returns list of candidates that can be the new validators
// Skips offline candidates and candidates with stake less than minValidatorBipStake
// Result is sorted by candidates stakes and limited to valCount
func (c *Candidates) GetNewCandidates(valCount int) []Candidate {
	var result []Candidate

	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		if candidate.Status == CandidateStatusOffline {
			continue
		}

		if candidate.totalBipStake.Cmp(minValidatorBipStake) == -1 {
			continue
		}

		result = append(result, *candidate)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].totalBipStake.Cmp(result[j].totalBipStake) == 1
	})

	if len(result) > valCount {
		result = result[:valCount]
	}

	return result
}

// Create creates a new candidate with given params and adds it to state
func (c *Candidates) Create(ownerAddress, rewardAddress, controlAddress types.Address, pubkey types.Pubkey, commission uint32) {
	candidate := &Candidate{
		ID:                0,
		PubKey:            pubkey,
		RewardAddress:     rewardAddress,
		OwnerAddress:      ownerAddress,
		ControlAddress:    controlAddress,
		Commission:        commission,
		Status:            CandidateStatusOffline,
		totalBipStake:     big.NewInt(0),
		stakes:            [MaxDelegatorsPerCandidate]*stake{},
		isDirty:           true,
		isTotalStakeDirty: true,
	}

	candidate.setTmAddress()
	c.setToMap(pubkey, candidate)
}

// CreateWithID creates a new candidate with given params and adds it to state
// CreateWithID uses given ID to be associated with public key of a candidate
func (c *Candidates) CreateWithID(ownerAddress, rewardAddress, controlAddress types.Address, pubkey types.Pubkey, commission uint32, id uint32) {
	c.setPubKeyID(pubkey, id)
	c.Create(ownerAddress, rewardAddress, controlAddress, pubkey, commission)
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

		c.bus.Events().AddEvent(uint32(height), &eventsdb.SlashEvent{
			Address:         stake.Owner,
			Amount:          slashed.String(),
			Coin:            uint64(stake.Coin),
			ValidatorPubKey: candidate.PubKey,
		})

		c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(newValue))
		c.bus.FrozenFunds().AddFrozenFund(height+UnbondPeriod, stake.Owner, candidate.PubKey, candidate.ID, stake.Coin, newValue)
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

func (c *Candidates) recalculateStakes(height uint64) {
	coinsCache := newCoinsCache()

	for _, pubkey := range c.getOrderedCandidates() {
		candidate := c.getFromMap(pubkey)
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
	}
}

func (c *Candidates) stakeKick(owner types.Address, value *big.Int, coin types.CoinID, pubKey types.Pubkey, height uint64) {
	c.bus.WaitList().AddToWaitList(owner, pubKey, coin, value)
	c.bus.Events().AddEvent(uint32(height), &eventsdb.StakeKickEvent{
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
	var candidates []*Candidate
	for _, pubkey := range c.getOrderedCandidates() {
		candidates = append(candidates, c.getFromMap(pubkey))
	}

	return candidates
}

// GetTotalStake calculates and returns total stake of a candidate
func (c *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	candidate := c.getFromMap(pubkey)
	if candidate.totalBipStake == nil {
		path := []byte{mainPrefix}
		path = append(path, candidate.idBytes()...)
		path = append(path, totalStakePrefix)
		_, enc := c.iavl.Get(path)
		if len(enc) == 0 {
			candidate.totalBipStake = big.NewInt(0)
			return big.NewInt(0)
		}

		candidate.totalBipStake = big.NewInt(0).SetBytes(enc)
	}

	return candidate.totalBipStake
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

	_, blockListEnc := c.iavl.Get([]byte{blockListPrefix})
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

	_, valueMaxID := c.iavl.Get([]byte{maxIDPrefix})
	if len(valueMaxID) != 0 {
		c.maxID = binary.LittleEndian.Uint32(valueMaxID)
	}

}

func (c *Candidates) loadCandidatesList() (maxID uint32) {
	_, pubIDenc := c.iavl.Get([]byte{pubKeyIDPrefix})
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
	_, enc := c.iavl.Get(path)
	if len(enc) != 0 {
		var candidates []*Candidate
		if err := rlp.DecodeBytes(enc, &candidates); err != nil {
			panic(fmt.Sprintf("failed to decode candidates: %s", err))
		}

		for _, candidate := range candidates {
			// load total stake
			path = append([]byte{mainPrefix}, candidate.idBytes()...)
			path = append(path, totalStakePrefix)
			_, enc = c.iavl.Get(path)
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

// LoadStakes loads all stakes of candidates
func (c *Candidates) LoadStakes() {
	for pubkey := range c.pubKeyIDs {
		c.LoadStakesOfCandidate(pubkey)
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
// 1. Subs 1% from each stake
// 2. Calculate and return new total stake
func (c *Candidates) Punish(height uint64, address types.TmAddress) *big.Int {

	candidate := c.GetCandidateByTendermintAddress(address)
	totalStake := new(big.Int).Set(candidate.totalBipStake)
	stakes := c.GetStakes(candidate.PubKey)
	for _, stake := range stakes {
		newValue := big.NewInt(0).Set(stake.Value)
		newValue.Mul(newValue, big.NewInt(99))
		newValue.Div(newValue, big.NewInt(100))

		slashed := big.NewInt(0).Set(stake.Value)
		slashed.Sub(slashed, newValue)

		if !stake.Coin.IsBaseCoin() {
			coin := c.bus.Coins().GetCoin(stake.Coin)
			ret := formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, slashed)

			c.bus.Coins().SubCoinVolume(coin.ID, slashed)
			c.bus.Coins().SubCoinReserve(coin.ID, ret)

			c.bus.App().AddTotalSlashed(ret)
			totalStake.Sub(totalStake, ret)
		} else {
			c.bus.App().AddTotalSlashed(slashed)
			totalStake.Sub(totalStake, slashed)
		}

		c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(slashed))

		c.bus.Events().AddEvent(uint32(height), &eventsdb.SlashEvent{
			Address:         stake.Owner,
			Amount:          slashed.String(),
			Coin:            uint64(stake.Coin),
			ValidatorPubKey: candidate.PubKey,
		})

		stake.setValue(newValue)
	}
	candidate.setTotalBipStake(totalStake)
	return totalStake
}

// SetStakes Sets stakes and updates of a candidate. Used in Import.
func (c *Candidates) SetStakes(pubkey types.Pubkey, stakes []types.Stake, updates []types.Stake) {
	candidate := c.GetCandidate(pubkey)
	candidate.stakesCount = len(stakes)

	for _, u := range updates {
		candidate.addUpdate(&stake{
			Owner:    u.Owner,
			Coin:     types.CoinID(u.Coin),
			Value:    helpers.StringToBigInt(u.Value),
			BipValue: helpers.StringToBigInt(u.BipValue),
		})
	}

	count := len(stakes)
	if count > MaxDelegatorsPerCandidate {
		count = MaxDelegatorsPerCandidate

		for _, u := range stakes[1000:] {
			candidate.addUpdate(&stake{
				Owner:    u.Owner,
				Coin:     types.CoinID(u.Coin),
				Value:    helpers.StringToBigInt(u.Value),
				BipValue: helpers.StringToBigInt(u.BipValue),
			})
		}
	}

	for i, s := range stakes[:count] {
		candidate.stakes[i] = &stake{
			Owner:    s.Owner,
			Coin:     types.CoinID(s.Coin),
			Value:    helpers.StringToBigInt(s.Value),
			BipValue: helpers.StringToBigInt(s.BipValue),
			markDirty: func(index int) {
				candidate.dirtyStakes[index] = true
			},
			index: i,
		}

		candidate.stakes[i].markDirty(i)
	}
}

// Export exports all data to the given state
func (c *Candidates) Export(state *types.AppState) {
	c.LoadCandidatesDeliver()
	c.LoadStakes()

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
			ID:             uint64(candidate.ID),
			RewardAddress:  candidate.RewardAddress,
			OwnerAddress:   candidate.OwnerAddress,
			ControlAddress: candidate.ControlAddress,
			TotalBipStake:  candidate.GetTotalBipStake().String(),
			PubKey:         candidate.PubKey,
			Commission:     uint64(candidate.Commission),
			Status:         uint64(candidate.Status),
			Updates:        updates,
			Stakes:         stakes,
		})
	}

	for pubkey := range c.blockList {
		state.BlockListCandidates = append(state.BlockListCandidates, pubkey)
	}
	sort.SliceStable(state.BlockListCandidates, func(i, j int) bool {
		return bytes.Compare(state.BlockListCandidates[i].Bytes(), state.BlockListCandidates[j].Bytes()) == 1
	})
}

func (c *Candidates) getOrderedCandidates() []types.Pubkey {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var keys []types.Pubkey
	for _, candidate := range c.list {
		keys = append(keys, candidate.PubKey)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (c *Candidates) getFromMap(pubkey types.Pubkey) *Candidate {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[c.id(pubkey)]
}

func (c *Candidates) setToMap(pubkey types.Pubkey, model *Candidate) {
	id := model.ID
	if id == 0 {
		id = c.getOrNewID(pubkey)
		model.ID = id
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
func (c *Candidates) LoadStakesOfCandidate(pubkey types.Pubkey) {
	candidate := c.GetCandidate(pubkey)

	// load stakes
	stakesCount := 0
	for index := 0; index < MaxDelegatorsPerCandidate; index++ {
		path := []byte{mainPrefix}
		path = append(path, candidate.idBytes()...)
		path = append(path, stakesPrefix)
		path = append(path, []byte(fmt.Sprintf("%d", index))...)
		_, enc := c.iavl.Get(path)
		if len(enc) == 0 {
			candidate.stakes[index] = nil
			continue
		}

		stake := &stake{}
		if err := rlp.DecodeBytes(enc, stake); err != nil {
			panic(fmt.Sprintf("failed to decode stake: %s", err))
		}

		candidate.setStakeAtIndex(index, stake, false)

		stakesCount++
	}

	candidate.stakesCount = stakesCount

	// load updates
	path := []byte{mainPrefix}
	path = append(path, candidate.idBytes()...)
	path = append(path, updatesPrefix)
	_, enc := c.iavl.Get(path)
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
					candidate.isUpdatesDirty = true
				}
			})(candidate)
		}

		candidate.updates = updates
	}

	// load total stake
	path = append([]byte{mainPrefix}, candidate.idBytes()...)
	path = append(path, totalStakePrefix)
	_, enc = c.iavl.Get(path)
	if len(enc) == 0 {
		candidate.totalBipStake = big.NewInt(0)
	} else {
		candidate.totalBipStake = big.NewInt(0).SetBytes(enc)
	}

	candidate.setTmAddress()
	c.setToMap(candidate.PubKey, candidate)
}

// ChangePubKey change public key of a candidate from old to new
func (c *Candidates) ChangePubKey(old types.Pubkey, new types.Pubkey) {
	if c.isBlocked(new) {
		panic("Candidate with such public key (" + new.String() + ") exists in block list")
	}

	c.getFromMap(old).setPublicKey(new)
	c.setBlockPubKey(old)
	c.setPubKeyID(new, c.pubKeyIDs[old])
	delete(c.pubKeyIDs, old)
	c.isChangedPublicKeys = true
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
	c.lock.Unlock()

	id = c.maxID
	c.setPubKeyID(pubKey, id)
	return id
}

func (c *Candidates) id(pubKey types.Pubkey) uint32 {
	return c.pubKeyIDs[pubKey]
}

// ID returns an id of candidate by it's public key
func (c *Candidates) ID(pubKey types.Pubkey) uint32 {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.id(pubKey)
}

// PubKey returns a public key of candidate by it's ID
func (c *Candidates) PubKey(id uint32) types.Pubkey {
	c.lock.RLock()
	defer c.lock.RUnlock()

	candidate, ok := c.list[id]
	if !ok {
		panic(fmt.Sprintf("candidate by ID %d not found", id))
	}

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
