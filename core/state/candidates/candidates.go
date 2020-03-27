package candidates

import (
	"bytes"
	"fmt"
	eventsdb "github.com/MinterTeam/events-db"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/MinterTeam/minter-go-node/upgrades"
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
	stakesPrefix     = 's'
	totalStakePrefix = 't'
	updatesPrefix    = 'u'
)

type Candidates struct {
	list map[types.Pubkey]*Candidate

	iavl tree.Tree
	bus  *bus.Bus

	lock   sync.RWMutex
	loaded bool
}

func NewCandidates(bus *bus.Bus, iavl tree.Tree) (*Candidates, error) {
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

	for _, pubkey := range keys {
		candidate := c.getFromMap(pubkey)
		candidate.isDirty = false

		if candidate.isTotalStakeDirty {
			path := []byte{mainPrefix}
			path = append(path, pubkey[:]...)
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
			path = append(path, pubkey[:]...)
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
			path = append(path, pubkey[:]...)
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

		if candidate.totalBipStake.Cmp(big.NewInt(0)) == 0 {
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

func (c *Candidates) Create(ownerAddress types.Address, rewardAddress types.Address, pubkey types.Pubkey, commission uint) {
	candidate := &Candidate{
		PubKey:            pubkey,
		RewardAddress:     rewardAddress,
		OwnerAddress:      ownerAddress,
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

			c.bus.Coins().SubCoinVolume(coin.Symbol, slashed)
			c.bus.Coins().SubCoinReserve(coin.Symbol, ret)

			c.bus.App().AddTotalSlashed(ret)
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(ret))
		} else {
			c.bus.App().AddTotalSlashed(slashed)
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(slashed))
		}

		c.bus.Events().AddEvent(uint32(height), eventsdb.SlashEvent{
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
	if height >= upgrades.UpgradeBlock3 {
		c.recalculateStakesNew(height)
	} else if height >= upgrades.UpgradeBlock2 {
		c.recalculateStakesOld2(height)
	} else {
		c.recalculateStakesOld1(height)
	}
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
					c.bus.Events().AddEvent(uint32(height), eventsdb.UnbondEvent{
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
					c.bus.Events().AddEvent(uint32(height), eventsdb.UnbondEvent{
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
				c.bus.Events().AddEvent(uint32(height), eventsdb.UnbondEvent{
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
				c.bus.Events().AddEvent(uint32(height), eventsdb.UnbondEvent{
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
				c.bus.Events().AddEvent(uint32(height), eventsdb.UnbondEvent{
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
				c.bus.Events().AddEvent(uint32(height), eventsdb.UnbondEvent{
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

	_, exists := c.list[pubkey]

	return exists
}

func (c *Candidates) Count() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.list)
}

func (c *Candidates) IsNewCandidateStakeSufficient(coin types.CoinSymbol, stake *big.Int, limit int) bool {
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

func (c *Candidates) IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, amount *big.Int) bool {
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

func (c *Candidates) Delegate(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int, bipValue *big.Int) {
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

func (c *Candidates) Edit(pubkey types.Pubkey, rewardAddress types.Address, ownerAddress types.Address) {
	candidate := c.getFromMap(pubkey)
	candidate.setOwner(ownerAddress)
	candidate.setReward(rewardAddress)
}

func (c *Candidates) SetOnline(pubkey types.Pubkey) {
	c.getFromMap(pubkey).setStatus(CandidateStatusOnline)
}

func (c *Candidates) SetOffline(pubkey types.Pubkey) {
	c.getFromMap(pubkey).setStatus(CandidateStatusOffline)
}

func (c *Candidates) SubStake(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
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

func (c *Candidates) GetStakeOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinSymbol) *Stake {
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

func (c *Candidates) GetStakeValueOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinSymbol) *big.Int {
	stake := c.GetStakeOfAddress(pubkey, address, coin)
	if stake == nil {
		return nil
	}

	return stake.Value
}

func (c *Candidates) GetCandidateOwner(pubkey types.Pubkey) types.Address {
	return c.getFromMap(pubkey).OwnerAddress
}

func (c *Candidates) LoadCandidates() {
	if c.loaded {
		return
	}
	c.loaded = true

	path := []byte{mainPrefix}
	_, enc := c.iavl.Get(path)
	if len(enc) == 0 {
		c.list = map[types.Pubkey]*Candidate{}
		return
	}

	var candidates []*Candidate
	if err := rlp.DecodeBytes(enc, &candidates); err != nil {
		panic(fmt.Sprintf("failed to decode candidates: %s", err))
	}

	c.list = map[types.Pubkey]*Candidate{}
	for _, candidate := range candidates {
		// load total stake
		path = append([]byte{mainPrefix}, candidate.PubKey.Bytes()...)
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

func (c *Candidates) LoadStakes() {
	for pubkey := range c.list {
		c.LoadStakesOfCandidate(pubkey)
	}
}

func (c *Candidates) calculateBipValue(coinSymbol types.CoinSymbol, amount *big.Int, includeSelf, includeUpdates bool, coinsCache *coinsCache) *big.Int {
	if coinSymbol.IsBaseCoin() {
		return big.NewInt(0).Set(amount)
	}

	totalAmount := big.NewInt(0)
	if includeSelf {
		totalAmount.Set(amount)
	}

	var totalPower *big.Int

	if coinsCache.Exists(coinSymbol) {
		totalPower, totalAmount = coinsCache.Get(coinSymbol)
	} else {
		candidates := c.GetCandidates()
		for _, candidate := range candidates {
			for _, stake := range candidate.stakes {
				if stake != nil && stake.Coin == coinSymbol {
					totalAmount.Add(totalAmount, stake.Value)
				}
			}

			if includeUpdates {
				for _, update := range candidate.updates {
					if update.Coin == coinSymbol {
						totalAmount.Add(totalAmount, update.Value)
					}
				}
			}
		}

		coin := c.bus.Coins().GetCoin(coinSymbol)

		totalPower = formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, totalAmount)
		coinsCache.Set(coinSymbol, totalPower, totalAmount)
	}

	return big.NewInt(0).Div(big.NewInt(0).Mul(totalPower, amount), totalAmount)
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

			c.bus.Coins().SubCoinVolume(coin.Symbol, slashed)
			c.bus.Coins().SubCoinReserve(coin.Symbol, ret)

			c.bus.App().AddTotalSlashed(ret)
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(ret))
		} else {
			c.bus.App().AddTotalSlashed(slashed)
			c.bus.Checker().AddCoin(stake.Coin, big.NewInt(0).Neg(slashed))
		}

		c.bus.Events().AddEvent(uint32(height), eventsdb.SlashEvent{
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

func (c *Candidates) SetStakes(pubkey types.Pubkey, stakes []types.Stake) {
	candidate := c.GetCandidate(pubkey)
	candidate.stakesCount = len(stakes)
	if candidate.stakesCount > MaxDelegatorsPerCandidate {
		candidate.stakesCount = MaxDelegatorsPerCandidate
		for i := MaxDelegatorsPerCandidate; i < len(stakes); i++ {
			stake := stakes[i]
			candidate.addUpdate(&Stake{
				Owner:    stake.Owner,
				Coin:     stake.Coin,
				Value:    helpers.StringToBigInt(stake.Value),
				BipValue: helpers.StringToBigInt(stake.BipValue),
			})
		}
	}
	for i := 0; i < candidate.stakesCount; i++ {
		stake := stakes[i]
		candidate.stakes[i] = &Stake{
			Owner:    stake.Owner,
			Coin:     stake.Coin,
			Value:    helpers.StringToBigInt(stake.Value),
			BipValue: helpers.StringToBigInt(stake.BipValue),
			index:    i,
			markDirty: func(index int) {
				candidate.dirtyStakes[index] = true
			},
		}
		candidate.stakes[i].markDirty(i)
	}
}

func (c *Candidates) Export(state *types.AppState) {
	c.LoadCandidates()
	c.LoadStakes()

	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		var stakes []types.Stake

		for _, s := range c.GetStakes(candidate.PubKey) {
			stakes = append(stakes, types.Stake{
				Owner:    s.Owner,
				Coin:     s.Coin,
				Value:    s.Value.String(),
				BipValue: s.BipValue.String(),
			})
		}

		state.Candidates = append(state.Candidates, types.Candidate{
			RewardAddress: candidate.RewardAddress,
			OwnerAddress:  candidate.OwnerAddress,
			TotalBipStake: candidate.GetTotalBipStake().String(),
			PubKey:        candidate.PubKey,
			Commission:    candidate.Commission,
			Stakes:        stakes,
			Status:        candidate.Status,
		})
	}

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

	return c.list[pubkey]
}

func (c *Candidates) setToMap(pubkey types.Pubkey, model *Candidate) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[pubkey] = model
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
		path = append(path, candidate.PubKey.Bytes()...)
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
	path = append(path, candidate.PubKey.Bytes()...)
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
	path = append([]byte{mainPrefix}, candidate.PubKey.Bytes()...)
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
