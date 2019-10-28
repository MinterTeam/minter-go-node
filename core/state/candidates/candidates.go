package candidates

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	compact "github.com/klim0v/compact-db"
	"math/big"
	"sort"
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
	list   map[types.Pubkey]*Candidate
	loaded bool

	iavl tree.Tree
	bus  *bus.Bus
}

func NewCandidates(bus *bus.Bus, iavl tree.Tree) (*Candidates, error) {
	candidates := &Candidates{iavl: iavl, bus: bus}
	candidates.bus.SetCandidates(NewBus(candidates))
	candidates.loadCandidates()

	return candidates, nil
}

func (c *Candidates) Commit() error {
	keys := c.getOrderedDirtyCandidates()
	if len(keys) > 0 {
		data, err := rlp.EncodeToBytes(c.list)
		if err != nil {
			return fmt.Errorf("can't encode candidates: %v", err)
		}

		path := []byte{mainPrefix}
		c.iavl.Set(path, data)
	}

	for _, pubkey := range keys {
		candidate := c.list[pubkey]
		candidate.isDirty = false

		if candidate.isTotalStakeDirty {
			path := []byte{mainPrefix}
			path = append(path, pubkey[:]...)
			path = append(path, totalStakePrefix)
			c.iavl.Set(path, candidate.totalBipStake.Bytes())
			candidate.isTotalStakeDirty = false
		}

		stakes := candidate.stakes
		for index, stake := range stakes {
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

func (c *Candidates) GetNewCandidates(valCount int, height int64) []Candidate {
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

func (c *Candidates) DeleteCoin(pubkey types.Pubkey, coinSymbol types.CoinSymbol) {
	stakes := c.GetStakes(pubkey)
	coin := c.bus.Coins().GetCoin(coinSymbol)
	candidate := c.GetCandidate(pubkey)

	for index, stake := range stakes {
		if stake.Coin != coinSymbol {
			continue
		}

		ret := formula.CalculateSaleReturn(coin.Volume, coin.Reserve, 100, stake.Value)

		tStake := c.GetStakeOfAddress(pubkey, stake.Owner, types.GetBaseCoin())
		if tStake == nil {
			stake.setValue(ret)
			stake.setCoin(types.GetBaseCoin())
		} else {
			candidate.stakes[index] = nil
			candidate.dirtyStakes[index] = true
			stake.Value.Add(stake.Value, ret)
		}
	}

	for _, update := range candidate.updates {
		if update.Coin != coinSymbol {
			continue
		}

		candidate.isUpdatesDirty = true
		update.Coin = types.GetBaseCoin()
		update.Value = formula.CalculateSaleReturn(coin.Volume, coin.Reserve, 100, update.Value)
	}
}

func (c *Candidates) Create(ownerAddress types.Address, rewardAddress types.Address, pubkey types.Pubkey, commission uint) {
	candidate := &Candidate{
		PubKey:            pubkey,
		RewardAddress:     rewardAddress,
		OwnerAddress:      ownerAddress,
		Commission:        commission,
		Status:            CandidateStatusOffline,
		totalBipStake:     big.NewInt(0),
		stakes:            [1000]*Stake{},
		isDirty:           true,
		isTotalStakeDirty: true,
	}
	candidate.setTmAddress()

	c.list[pubkey] = candidate
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
		} else {
			c.bus.App().AddTotalSlashed(slashed)
		}

		c.bus.Events().AddEvent(uint32(height), compact.SlashEvent{
			Address:         stake.Owner,
			Amount:          slashed.Bytes(),
			Coin:            stake.Coin,
			ValidatorPubKey: candidate.PubKey,
		})

		c.bus.FrozenFunds().AddFrozenFund(height+UnbondPeriod, stake.Owner, candidate.PubKey, stake.Coin, newValue)
		c.bus.Coins().SanitizeCoin(stake.Coin)
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
	for _, candidate := range c.list {
		stakes := c.GetStakes(candidate.PubKey)
		for _, stake := range stakes {
			stake.setBipValue(c.calculateBipValue(stake.Coin, stake.Value, false, true))
		}

		// apply updates for existing stakes
		for _, update := range candidate.updates {
			stake := c.GetStakeOfAddress(candidate.PubKey, update.Owner, update.Coin)
			if stake != nil {
				stake.addValue(update.Value)
				update.Value = big.NewInt(0)
				stake.setBipValue(c.calculateBipValue(stake.Coin, stake.Value, false, true))
			}
		}

		updates := candidate.GetFilteredUpdates()
		for _, update := range updates {
			update.setBipValue(c.calculateBipValue(update.Coin, update.Value, false, true))
		}
		// Sort updates in descending order
		sort.SliceStable(updates, func(i, j int) bool {
			return updates[i].BipValue.Cmp(updates[j].BipValue) == 1
		})

		for _, update := range updates {
			// find and replace smallest stake
			index := -1
			var smallestStake *big.Int
			for i, stake := range stakes {
				if stake == nil {
					index = i
					break
				}

				if smallestStake == nil || smallestStake.Cmp(stake.BipValue) == 1 {
					smallestStake = big.NewInt(0).Set(stake.BipValue)
					index = i
				}
			}

			if smallestStake != nil && smallestStake.Cmp(update.BipValue) == 1 {
				c.bus.Events().AddEvent(uint32(height), compact.UnbondEvent{
					Address:         update.Owner,
					Amount:          update.Value.Bytes(),
					Coin:            update.Coin,
					ValidatorPubKey: candidate.PubKey,
				})
				c.bus.Accounts().AddBalance(update.Owner, update.Coin, update.Value)
				update.setValue(big.NewInt(0))
				continue
			}

			stakes[index] = update
		}

		candidate.clearUpdates()

		totalBipValue := big.NewInt(0)
		for _, stake := range stakes {
			if stake == nil {
				continue
			}
			totalBipValue.Add(totalBipValue, stake.BipValue)
		}

		candidate.setTotalBipValue(totalBipValue)
		candidate.updateStakesCount()
	}
}

func (c *Candidates) Exists(pubkey types.Pubkey) bool {
	_, exists := c.list[pubkey]

	return exists
}

func (c *Candidates) Count() int {
	return len(c.list)
}

func (c *Candidates) IsNewCandidateStakeSufficient(coin types.CoinSymbol, stake *big.Int) bool {
	bipValue := c.calculateBipValue(coin, stake, true, true)
	candidates := c.list

	for _, candidate := range candidates {
		if candidate.totalBipStake.Cmp(bipValue) == -1 {
			return true
		}
	}

	return false
}

func (c *Candidates) GetCandidate(pubkey types.Pubkey) *Candidate {
	return c.list[pubkey]
}

func (c *Candidates) IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, amount *big.Int) bool {
	stakes := c.GetStakes(pubkey)
	if len(stakes) < MaxDelegatorsPerCandidate {
		return true
	}

	stakeValue := c.calculateBipValue(coin, amount, true, true)
	for _, stake := range stakes {
		if stakeValue.Cmp(stake.BipValue) == -1 {
			return true
		}
	}

	return false
}

func (c *Candidates) Delegate(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int, bipValue *big.Int) {
	stake := &Stake{
		Owner:    address,
		Coin:     coin,
		Value:    value,
		BipValue: bipValue,
	}

	c.bus.Coins().AddOwnerCandidate(coin, pubkey)

	candidate := c.GetCandidate(pubkey)
	candidate.addUpdate(stake)
}

func (c *Candidates) Edit(pubkey types.Pubkey, rewardAddress types.Address, ownerAddress types.Address) {
	c.list[pubkey].setOwner(ownerAddress)
	c.list[pubkey].setReward(rewardAddress)
}

func (c *Candidates) SetOnline(pubkey types.Pubkey) {
	c.list[pubkey].setStatus(CandidateStatusOnline)
}

func (c *Candidates) SetOffline(pubkey types.Pubkey) {
	c.list[pubkey].setStatus(CandidateStatusOffline)
}

func (c *Candidates) SubStake(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	stake := c.GetStakeOfAddress(pubkey, address, coin)
	stake.subValue(value)
}

func (c *Candidates) GetCandidates() []*Candidate {
	var candidates []*Candidate
	for _, candidate := range c.list {
		candidates = append(candidates, candidate)
	}

	return candidates
}

func (c *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	candidate := c.list[pubkey]
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
	stakes := c.GetStakes(pubkey)
	for _, stake := range stakes {
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
	return c.list[pubkey].OwnerAddress
}

func (c *Candidates) loadCandidates() {
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
	if err := rlp.DecodeBytes(enc, candidates); err != nil {
		panic(fmt.Sprintf("failed to decode candidates: %s", err))
		return
	}

	c.list = map[types.Pubkey]*Candidate{}
	for _, candidate := range candidates {
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

			var stake Stake
			if err := rlp.DecodeBytes(enc, &stake); err != nil {
				panic(fmt.Sprintf("failed to decode stake: %s", err))
			}

			stake.markDirty = func(index int) {
				candidate.dirtyStakes[index] = true
			}

			candidate.stakes[index] = &stake
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
				update.markDirty = func(i int) {
					candidate.isUpdatesDirty = true
				}
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
		c.list[candidate.PubKey] = candidate
	}
}

func (c *Candidates) calculateBipValue(coinSymbol types.CoinSymbol, amount *big.Int, includeSelf, includeUpdates bool) *big.Int {
	if coinSymbol.IsBaseCoin() {
		return big.NewInt(0).Set(amount)
	}

	totalAmount := big.NewInt(0)
	if includeSelf {
		totalAmount.Set(amount)
	}

	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		stakes := c.GetStakes(candidate.PubKey)
		for _, stake := range stakes {
			if stake.Coin == coinSymbol {
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

	return formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, totalAmount)
}

func (c *Candidates) getOrderedDirtyCandidates() []types.Pubkey {
	var keys []types.Pubkey
	for _, candidate := range c.list {
		if !candidate.HasDirty() {
			continue
		}
		keys = append(keys, candidate.PubKey)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
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
			c.bus.Coins().SanitizeCoin(stake.Coin)

			c.bus.App().AddTotalSlashed(ret)
		} else {
			c.bus.App().AddTotalSlashed(slashed)
		}

		c.bus.Events().AddEvent(uint32(height), compact.SlashEvent{
			Address:         stake.Owner,
			Amount:          slashed.Bytes(),
			Coin:            stake.Coin,
			ValidatorPubKey: candidate.PubKey,
		})

		stake.setValue(newValue)
		totalStake.Add(totalStake, newValue)
	}

	return totalStake
}

func (c *Candidates) SetCandidates(candidates []Candidate) {

}
