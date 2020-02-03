package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

type Candidate struct {
	PubKey        types.Pubkey
	RewardAddress types.Address
	OwnerAddress  types.Address
	Commission    uint
	Status        byte

	totalBipStake *big.Int
	stakesCount   int
	stakes        [MaxDelegatorsPerCandidate]*Stake
	updates       []*Stake
	tmAddress     *types.TmAddress

	isDirty           bool
	isTotalStakeDirty bool
	isUpdatesDirty    bool
	dirtyStakes       [MaxDelegatorsPerCandidate]bool
}

func (candidate *Candidate) setStatus(status byte) {
	candidate.isDirty = true
	candidate.Status = status
}

func (candidate *Candidate) setOwner(address types.Address) {
	candidate.isDirty = true
	candidate.OwnerAddress = address
}

func (candidate *Candidate) setReward(address types.Address) {
	candidate.isDirty = true
	candidate.RewardAddress = address
}

func (candidate *Candidate) addUpdate(stake *Stake) {
	candidate.isUpdatesDirty = true
	stake.markDirty = func(i int) {
		candidate.isUpdatesDirty = true
	}
	candidate.updates = append(candidate.updates, stake)
}

func (candidate *Candidate) clearUpdates() {
	if candidate.updates != nil {
		candidate.isUpdatesDirty = true
	}

	candidate.updates = nil
}

func (candidate *Candidate) setTotalBipStake(totalBipValue *big.Int) {
	if totalBipValue.Cmp(candidate.totalBipStake) != 0 {
		candidate.isTotalStakeDirty = true
	}

	candidate.totalBipStake.Set(totalBipValue)
}

func (candidate *Candidate) GetTmAddress() types.TmAddress {
	return *candidate.tmAddress
}

func (candidate *Candidate) setTmAddress() {
	// set tm address
	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], candidate.PubKey[:])

	var address types.TmAddress
	copy(address[:], pubkey.Address().Bytes())

	candidate.tmAddress = &address
}

func (candidate *Candidate) GetFilteredUpdates() []*Stake {
	var updates []*Stake
	for _, update := range candidate.updates {
		// skip updates with 0 stakes
		if update.Value.Cmp(big.NewInt(0)) != 1 {
			continue
		}

		// merge updates
		merged := false
		for _, u := range updates {
			if u.Coin == update.Coin && u.Owner == update.Owner {
				u.Value.Add(u.Value, update.Value)
				merged = true
				break
			}
		}

		if !merged {
			updates = append(updates, update)
		}
	}

	return updates
}

func (candidate *Candidate) updateStakesCount() {
	count := 0
	for _, stake := range candidate.stakes {
		if stake != nil {
			count++
		}
	}
	candidate.stakesCount = count
}

func (candidate *Candidate) GetTotalBipStake() *big.Int {
	return big.NewInt(0).Set(candidate.totalBipStake)
}

func (candidate *Candidate) SetStakeAtIndex(index int, stake *Stake, isDirty bool) {
	stake.markDirty = func(i int) {
		candidate.dirtyStakes[i] = true
	}
	stake.index = index

	candidate.stakes[index] = stake

	if isDirty {
		stake.markDirty(index)
	}
}

type Stake struct {
	Owner    types.Address
	Coin     types.CoinSymbol
	Value    *big.Int
	BipValue *big.Int

	index     int
	markDirty func(int)
}

func (stake *Stake) addValue(value *big.Int) {
	stake.markDirty(stake.index)
	stake.Value.Add(stake.Value, value)
}

func (stake *Stake) subValue(value *big.Int) {
	stake.markDirty(stake.index)
	stake.Value.Sub(stake.Value, value)
}

func (stake *Stake) setBipValue(value *big.Int) {
	if stake.BipValue.Cmp(value) != 0 {
		stake.markDirty(stake.index)
	}

	stake.BipValue.Set(value)
}

func (stake *Stake) setNewOwner(coin types.CoinSymbol, owner types.Address) {
	stake.Coin = coin
	stake.Owner = owner
	stake.BipValue = big.NewInt(0)
	stake.Value = big.NewInt(0)
	stake.markDirty(stake.index)
}

func (stake *Stake) setValue(ret *big.Int) {
	stake.markDirty(stake.index)
	stake.Value.Set(ret)
}

func (stake *Stake) setCoin(coin types.CoinSymbol) {
	stake.markDirty(stake.index)
	stake.Coin = coin
}

type coinsCache struct {
	list map[types.CoinSymbol]*coinsCacheItem
}

func newCoinsCache() *coinsCache {
	return &coinsCache{list: map[types.CoinSymbol]*coinsCacheItem{}}
}

type coinsCacheItem struct {
	totalPower  *big.Int
	totalAmount *big.Int
}

func (c *coinsCache) Exists(symbol types.CoinSymbol) bool {
	if c == nil {
		return false
	}

	_, exists := c.list[symbol]

	return exists
}

func (c *coinsCache) Get(symbol types.CoinSymbol) (totalPower *big.Int, totalAmount *big.Int) {
	return c.list[symbol].totalPower, c.list[symbol].totalAmount
}

func (c *coinsCache) Set(symbol types.CoinSymbol, totalPower *big.Int, totalAmount *big.Int) {
	if c == nil {
		return
	}

	if c.list[symbol] == nil {
		c.list[symbol] = &coinsCacheItem{}
	}

	c.list[symbol].totalAmount = totalAmount
	c.list[symbol].totalPower = totalPower
}
