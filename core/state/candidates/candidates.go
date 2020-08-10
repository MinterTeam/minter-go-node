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

const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02

	UnbondPeriod              = 518400
	MaxDelegatorsPerCandidate = 1000

	mainPrefix       = 'c'
	pubKeyIDPrefix   = mainPrefix + 'p'
	blockListPrefix  = mainPrefix + 'b'
	maxIDPrefix      = mainPrefix + 'i'
	stakesPrefix     = 's'
	totalStakePrefix = 't'
	updatesPrefix    = 'u'
)

var (
	minValidatorBipStake = big.NewInt(1000)
)

type RCandidates interface {
	Export(state *types.AppState)
	Exists(pubkey types.Pubkey) bool
	IsBlockedPubKey(pubkey types.Pubkey) bool
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
	GetStakes(pubkey types.Pubkey) []*Stake
}

type Candidates struct {
	list map[uint32]*Candidate

	isDirty   bool
	blockList map[types.Pubkey]struct{}
	pubKeyIDs map[types.Pubkey]uint32
	maxID     uint32

	iavl tree.MTree
	bus  *bus.Bus

	lock   sync.RWMutex
	loaded bool
}

func NewCandidates(bus *bus.Bus, iavl tree.MTree) (*Candidates, error) {
	candidates := &Candidates{iavl: iavl, bus: bus}
	candidates.bus.SetCandidates(NewBus(candidates))

	return candidates, nil
}

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
		pubIdData, err := rlp.EncodeToBytes(pubIDs)
		if err != nil {
			panic(fmt.Sprintf("failed to encode candidates public key with ID: %s", err))
		}

		c.iavl.Set([]byte{pubKeyIDPrefix}, pubIdData)

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
			path = append(path, []byte(fmt.Sprintf("%d", index))...)

			if stake == nil || stake.Value.Cmp(big.NewInt(0)) == 0 {
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

func (c *Candidates) Create(ownerAddress, rewardAddress, controlAddress types.Address, pubkey types.Pubkey, commission uint) {
	candidate := &Candidate{
		ID:                c.getOrNewID(pubkey),
		PubKey:            pubkey,
		RewardAddress:     rewardAddress,
		OwnerAddress:      ownerAddress,
		ControlAddress:    controlAddress,
		Commission:        commission,
		Status:            CandidateStatusOffline,
		totalBipStake:     big.NewInt(0),
		stakes:            [MaxDelegatorsPerCandidate]*Stake{},
		isDirty:           true,
		isTotalStakeDirty: true,
	}

	candidate.setTmAddress()
	c.setToMap(pubkey, candidate)
}

func (c *Candidates) CreateWithID(ownerAddress, rewardAddress, controlAddress types.Address, pubkey types.Pubkey, commission uint, id uint32) {
	c.setPubKeyID(pubkey, id)
	c.Create(ownerAddress, rewardAddress, controlAddress, pubkey, commission)
}

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
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(ret))
		} else {
			c.bus.App().AddTotalSlashed(slashed)
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(slashed))
		}

		c.bus.Events().AddEvent(uint32(height), &eventsdb.SlashEvent{
			Address:         stake.Owner,
			Amount:          slashed.String(),
			Coin:            stake.Coin,
			ValidatorPubKey: candidate.PubKey,
		})

		c.bus.FrozenFunds().AddFrozenFund(height+UnbondPeriod, stake.Owner, candidate.PubKey, stake.Coin, newValue)
		stake.setValue(big.NewInt(0))
	}
}

func (c *Candidates) GetCandidateByTendermintAddress(address types.TmAddress) *Candidate {
	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		if candidate.GetTmAddress() == address {
			return candidate
		}
	}

	return nil
}

func (c *Candidates) RecalculateStakes(height uint64) {
	c.recalculateStakesNew(height)
}

func (c *Candidates) recalculateStakesOld1(height uint64) {
	coinsCache := newCoinsCache()

	for _, pubkey := range c.getOrderedCandidates() {
		candidate := c.getFromMap(pubkey)
		stakes := c.GetStakes(candidate.PubKey)
		for _, stake := range stakes {
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

		updates := candidate.GetFilteredUpdates()
		for _, update := range updates {
			update.setBipValue(c.calculateBipValue(update.Coin, update.Value, false, true, coinsCache))
		}
		// Sort updates in descending order
		sort.SliceStable(updates, func(i, j int) bool {
			return updates[i].BipValue.Cmp(updates[j].BipValue) == 1
		})

		for _, update := range updates {
			if candidate.stakesCount < MaxDelegatorsPerCandidate {
				candidate.SetStakeAtIndex(candidate.stakesCount, update, true)
				candidate.stakesCount++
				stakes = c.GetStakes(candidate.PubKey)
			} else {
				// find and replace smallest stake
				index := -1
				var smallestStake *big.Int
				for i, stake := range stakes {
					if stake == nil {
						index = i
						smallestStake = big.NewInt(0)
						break
					}

					if smallestStake == nil || smallestStake.Cmp(stake.BipValue) == 1 {
						smallestStake = big.NewInt(0).Set(stake.BipValue)
						index = i
					}
				}

				if index == -1 || smallestStake.Cmp(update.BipValue) == 1 {
					c.bus.Events().AddEvent(uint32(height), &eventsdb.UnbondEvent{
						Address:         update.Owner,
						Amount:          update.Value.String(),
						Coin:            update.Coin,
						ValidatorPubKey: candidate.PubKey,
					})
					c.bus.Accounts().AddBalance(update.Owner, update.Coin, update.Value)
					c.bus.Checker().AddCoin(update.Coin, big.NewInt(0).Neg(update.Value))
					update.setValue(big.NewInt(0))
					continue
				}

				if stakes[index] != nil {
					c.bus.Events().AddEvent(uint32(height), &eventsdb.UnbondEvent{
						Address:         stakes[index].Owner,
						Amount:          stakes[index].Value.String(),
						Coin:            stakes[index].Coin,
						ValidatorPubKey: candidate.PubKey,
					})
					c.bus.Accounts().AddBalance(stakes[index].Owner, stakes[index].Coin, stakes[index].Value)
					c.bus.Checker().AddCoin(stakes[index].Coin, big.NewInt(0).Neg(stakes[index].Value))
				}

				candidate.SetStakeAtIndex(index, update, true)
				stakes = c.GetStakes(candidate.PubKey)
			}
		}

		candidate.clearUpdates()

		totalBipValue := big.NewInt(0)
		for _, stake := range c.GetStakes(candidate.PubKey) {
			if stake == nil {
				continue
			}
			totalBipValue.Add(totalBipValue, stake.BipValue)
		}

		candidate.setTotalBipStake(totalBipValue)
		candidate.updateStakesCount()
	}
}

func (c *Candidates) recalculateStakesOld2(height uint64) {
	coinsCache := newCoinsCache()

	for _, pubkey := range c.getOrderedCandidates() {
		candidate := c.getFromMap(pubkey)
		stakes := c.GetStakes(candidate.PubKey)
		for _, stake := range stakes {
			stake.setBipValue(c.calculateBipValue(stake.Coin, stake.Value, false, true, coinsCache))
		}

		// apply updates for existing stakes
		candidate.FilterUpdates()
		for _, update := range candidate.updates {
			stake := c.GetStakeOfAddress(candidate.PubKey, update.Owner, update.Coin)
			if stake != nil {
				stake.addValue(update.Value)
				update.setValue(big.NewInt(0))
				stake.setBipValue(c.calculateBipValue(stake.Coin, stake.Value, false, true, coinsCache))
			}
		}

		candidate.FilterUpdates()
		for _, update := range candidate.updates {
			update.setBipValue(c.calculateBipValue(update.Coin, update.Value, false, true, coinsCache))
		}

		for _, update := range candidate.updates {
			// find and replace smallest stake
			index := -1
			var smallestStake *big.Int

			if len(stakes) == 0 {
				index = 0
				smallestStake = big.NewInt(0)
			} else if len(stakes) < MaxDelegatorsPerCandidate {
				for i, stake := range stakes {
					if stake == nil {
						index = i
						break
					}
				}

				if index == -1 {
					index = len(stakes)
				}

				smallestStake = big.NewInt(0)
			} else {
				for i, stake := range stakes {
					if stake == nil {
						index = i
						smallestStake = big.NewInt(0)
						break
					}

					if smallestStake == nil || smallestStake.Cmp(stake.BipValue) == 1 {
						smallestStake = big.NewInt(0).Set(stake.BipValue)
						index = i
					}
				}
			}

			if index == -1 || smallestStake.Cmp(update.BipValue) == 1 {
				c.bus.Events().AddEvent(uint32(height), &eventsdb.UnbondEvent{
					Address:         update.Owner,
					Amount:          update.Value.String(),
					Coin:            update.Coin,
					ValidatorPubKey: candidate.PubKey,
				})
				c.bus.Accounts().AddBalance(update.Owner, update.Coin, update.Value)
				c.bus.Checker().AddCoin(update.Coin, big.NewInt(0).Neg(update.Value))
				update.setValue(big.NewInt(0))
				continue
			}

			if len(stakes) > index && stakes[index] != nil {
				c.bus.Events().AddEvent(uint32(height), &eventsdb.UnbondEvent{
					Address:         stakes[index].Owner,
					Amount:          stakes[index].Value.String(),
					Coin:            stakes[index].Coin,
					ValidatorPubKey: candidate.PubKey,
				})
				c.bus.Accounts().AddBalance(stakes[index].Owner, stakes[index].Coin, stakes[index].Value)
				c.bus.Checker().AddCoin(stakes[index].Coin, big.NewInt(0).Neg(stakes[index].Value))
			}

			candidate.SetStakeAtIndex(index, update, true)
			stakes = c.GetStakes(candidate.PubKey)
		}

		candidate.clearUpdates()

		totalBipValue := big.NewInt(0)
		for _, stake := range c.GetStakes(candidate.PubKey) {
			if stake == nil {
				continue
			}
			totalBipValue.Add(totalBipValue, stake.BipValue)
		}

		candidate.setTotalBipStake(totalBipValue)
		candidate.updateStakesCount()
	}
}

func (c *Candidates) recalculateStakesNew(height uint64) {
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

		candidate.FilterUpdates()
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
				c.bus.Events().AddEvent(uint32(height), &eventsdb.UnbondEvent{
					Address:         update.Owner,
					Amount:          update.Value.String(),
					Coin:            update.Coin,
					ValidatorPubKey: candidate.PubKey,
				})
				c.bus.Accounts().AddBalance(update.Owner, update.Coin, update.Value)
				c.bus.Checker().AddCoin(update.Coin, big.NewInt(0).Neg(update.Value))
				update.setValue(big.NewInt(0))
				continue
			}

			if stakes[index] != nil {
				c.bus.Events().AddEvent(uint32(height), &eventsdb.UnbondEvent{
					Address:         stakes[index].Owner,
					Amount:          stakes[index].Value.String(),
					Coin:            stakes[index].Coin,
					ValidatorPubKey: candidate.PubKey,
				})
				c.bus.Accounts().AddBalance(stakes[index].Owner, stakes[index].Coin, stakes[index].Value)
				c.bus.Checker().AddCoin(stakes[index].Coin, big.NewInt(0).Neg(stakes[index].Value))
			}

			candidate.SetStakeAtIndex(index, update, true)
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

func (c *Candidates) Exists(pubkey types.Pubkey) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.existPubKey(pubkey)
}

func (c *Candidates) existPubKey(pubKey types.Pubkey) bool {
	if c.pubKeyIDs == nil {
		return false
	}
	_, exists := c.pubKeyIDs[pubKey]
	return exists
}

func (c *Candidates) IsBlockedPubKey(pubkey types.Pubkey) bool {
	return c.isBlocked(pubkey)
}

func (c *Candidates) isBlocked(pubKey types.Pubkey) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, exists := c.blockList[pubKey]
	return exists
}

func (c *Candidates) Count() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.list)
}

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

func (c *Candidates) GetCandidate(pubkey types.Pubkey) *Candidate {
	return c.getFromMap(pubkey)
}

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

func (c *Candidates) Delegate(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int, bipValue *big.Int) {
	stake := &Stake{
		Owner:    address,
		Coin:     coin,
		Value:    big.NewInt(0).Set(value),
		BipValue: big.NewInt(0).Set(bipValue),
	}

	candidate := c.GetCandidate(pubkey)
	candidate.addUpdate(stake)

	c.bus.Checker().AddCoin(coin, value)
}

func (c *Candidates) Edit(pubkey types.Pubkey, rewardAddress types.Address, ownerAddress types.Address, controlAddress types.Address) {
	candidate := c.getFromMap(pubkey)
	candidate.setOwner(ownerAddress)
	candidate.setReward(rewardAddress)
	candidate.setControl(controlAddress)
}

func (c *Candidates) SetOnline(pubkey types.Pubkey) {
	c.getFromMap(pubkey).setStatus(CandidateStatusOnline)
}

func (c *Candidates) SetOffline(pubkey types.Pubkey) {
	c.getFromMap(pubkey).setStatus(CandidateStatusOffline)
}

func (c *Candidates) SubStake(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int) {
	stake := c.GetStakeOfAddress(pubkey, address, coin)
	stake.subValue(value)
	c.bus.Checker().AddCoin(coin, big.NewInt(0).Neg(value))
}

func (c *Candidates) GetCandidates() []*Candidate {
	var candidates []*Candidate
	for _, pubkey := range c.getOrderedCandidates() {
		candidates = append(candidates, c.getFromMap(pubkey))
	}

	return candidates
}

func (c *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	candidate := c.getFromMap(pubkey)
	if candidate.totalBipStake == nil {
		path := []byte{mainPrefix}
		path = append(path, pubkey[:]...)
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

func (c *Candidates) GetStakes(pubkey types.Pubkey) []*Stake {
	candidate := c.GetCandidate(pubkey)

	var stakes []*Stake
	for i := 0; i < MaxDelegatorsPerCandidate; i++ {
		stake := candidate.stakes[i]
		if stake == nil {
			continue
		}
		stakes = append(stakes, stake)
	}

	return stakes
}

func (c *Candidates) StakesCount(pubkey types.Pubkey) int {
	return c.GetCandidate(pubkey).stakesCount
}

func (c *Candidates) GetStakeOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinID) *Stake {
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

func (c *Candidates) GetStakeValueOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinID) *big.Int {
	stake := c.GetStakeOfAddress(pubkey, address, coin)
	if stake == nil {
		return nil
	}

	return stake.Value
}

func (c *Candidates) GetCandidateOwner(pubkey types.Pubkey) types.Address {
	return c.getFromMap(pubkey).OwnerAddress
}

func (c *Candidates) GetCandidateControl(pubkey types.Pubkey) types.Address {
	return c.getFromMap(pubkey).ControlAddress
}

// Load only list candidates (for read)
func (c *Candidates) LoadCandidates() {
	if c.checkAndSetLoaded() {
		return
	}

	_ = c.loadCandidatesList()
}

// Load full info about candidates (for edit)
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

func (c *Candidates) LoadStakes() {
	for pubkey := range c.pubKeyIDs {
		c.LoadStakesOfCandidate(pubkey)
	}
}

func (c *Candidates) calculateBipValue(coinID types.CoinID, amount *big.Int, includeSelf, includeUpdates bool, coinsCache *coinsCache) *big.Int {

	if coinID.IsBaseCoin() {
		return big.NewInt(0).Set(amount)
	}

	totalAmount := big.NewInt(0)
	if includeSelf {
		totalAmount.Set(amount)
	}

	var totalDelegatedValue *big.Int
	coin := c.bus.Coins().GetCoin(coinID)

	if coinsCache.Exists(coinID) {
		totalDelegatedValue, totalAmount = coinsCache.Get(coinID)
	} else {
		candidates := c.GetCandidates()
		for _, candidate := range candidates {
			for _, stake := range candidate.stakes {
				if stake != nil && stake.Coin == coinID {
					totalAmount.Add(totalAmount, stake.Value)
				}
			}

			if includeUpdates {
				for _, update := range candidate.updates {
					if update.Coin == coinID {
						totalAmount.Add(totalAmount, update.Value)
					}
				}
			}
		}

		coin := c.bus.Coins().GetCoin(coinID)

		totalDelegatedValue = formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, totalAmount)
		coinsCache.Set(coinID, totalDelegatedValue, totalAmount)
	}

	nonLockedSupply := big.NewInt(0).Sub(coin.Reserve, totalDelegatedValue)
	saleReturn := formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, nonLockedSupply)
	return big.NewInt(0).Div(big.NewInt(0).Mul(big.NewInt(0).Sub(coin.Reserve, saleReturn), amount), totalDelegatedValue)
}

func (c *Candidates) Punish(height uint64, address types.TmAddress) *big.Int {
	totalStake := big.NewInt(0)

	candidate := c.GetCandidateByTendermintAddress(address)

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
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(ret))
		} else {
			c.bus.App().AddTotalSlashed(slashed)
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(slashed))
		}

		c.bus.Events().AddEvent(uint32(height), &eventsdb.SlashEvent{
			Address:         stake.Owner,
			Amount:          slashed.String(),
			Coin:            stake.Coin,
			ValidatorPubKey: candidate.PubKey,
		})

		stake.setValue(newValue)
		totalStake.Add(totalStake, newValue)
	}

	return totalStake
}

func (c *Candidates) SetStakes(pubkey types.Pubkey, stakes []types.Stake, updates []types.Stake) {
	candidate := c.GetCandidate(pubkey)
	candidate.stakesCount = len(stakes)

	for _, u := range updates {
		candidate.addUpdate(&Stake{
			Owner:    u.Owner,
			Coin:     u.Coin,
			Value:    helpers.StringToBigInt(u.Value),
			BipValue: helpers.StringToBigInt(u.BipValue),
		})
	}

	count := len(stakes)
	if count > MaxDelegatorsPerCandidate {
		count = MaxDelegatorsPerCandidate

		for _, u := range stakes[1000:] {
			candidate.addUpdate(&Stake{
				Owner:    u.Owner,
				Coin:     u.Coin,
				Value:    helpers.StringToBigInt(u.Value),
				BipValue: helpers.StringToBigInt(u.BipValue),
			})
		}
	}

	for i, s := range stakes[:count] {
		candidate.stakes[i] = &Stake{
			Owner:    s.Owner,
			Coin:     s.Coin,
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
				Coin:     s.Coin,
				Value:    s.Value.String(),
				BipValue: s.BipValue.String(),
			}
		}

		updates := make([]types.Stake, len(candidate.updates))
		for i, u := range candidate.updates {
			updates[i] = types.Stake{
				Owner:    u.Owner,
				Coin:     u.Coin,
				Value:    u.Value.String(),
				BipValue: u.BipValue.String(),
			}
		}

		state.Candidates = append(state.Candidates, types.Candidate{
			ID:             candidate.ID,
			RewardAddress:  candidate.RewardAddress,
			OwnerAddress:   candidate.OwnerAddress,
			ControlAddress: candidate.ControlAddress,
			TotalBipStake:  candidate.GetTotalBipStake().String(),
			PubKey:         candidate.PubKey,
			Commission:     candidate.Commission,
			Status:         candidate.Status,
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

	if c.list == nil {
		c.list = map[uint32]*Candidate{}
	}
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

func (c *Candidates) SetTotalStake(pubkey types.Pubkey, stake *big.Int) {
	c.GetCandidate(pubkey).setTotalBipStake(stake)
}

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

		stake := &Stake{}
		if err := rlp.DecodeBytes(enc, stake); err != nil {
			panic(fmt.Sprintf("failed to decode stake: %s", err))
		}

		candidate.SetStakeAtIndex(index, stake, false)

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
		var updates []*Stake
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

func (c *Candidates) ChangePubKey(old types.Pubkey, new types.Pubkey) {
	if c.isBlocked(new) {
		panic("Candidate with such public key (" + new.String() + ") exists in block list")
	}

	c.getFromMap(old).PubKey = new
	c.setBlockPubKey(old)
	c.setPubKeyID(new, c.pubKeyIDs[old])
	delete(c.pubKeyIDs, old)
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

func (c *Candidates) ID(pubKey types.Pubkey) uint32 {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.pubKeyIDs[pubKey]
}

func (c *Candidates) setPubKeyID(pubkey types.Pubkey, u uint32) {
	if u == 0 {
		panic("public key of candidate cannot be equal 0")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.pubKeyIDs == nil {
		c.pubKeyIDs = map[types.Pubkey]uint32{}
	}
	c.pubKeyIDs[pubkey] = u
	c.isDirty = true
}

func (c *Candidates) setBlockPubKey(p types.Pubkey) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.blockList == nil {
		c.blockList = map[types.Pubkey]struct{}{}
	}
	c.blockList[p] = struct{}{}
	c.isDirty = true
}

func (c *Candidates) AddToBlockPubKey(p types.Pubkey) {
	c.setBlockPubKey(p)
}

func (c *Candidates) maxIDBytes() []byte {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(c.maxID))
	return bs
}
