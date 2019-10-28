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
	candidate.updates = append(candidate.updates, stake)
}

func (candidate *Candidate) clearUpdates() {
	candidate.updates = nil
	candidate.isUpdatesDirty = true
}

func (candidate *Candidate) setTotalBipValue(totalBipValue *big.Int) {
	candidate.isTotalStakeDirty = true
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

func (candidate *Candidate) HasDirty() bool {
	isDirty := candidate.isDirty || candidate.isUpdatesDirty || candidate.isTotalStakeDirty
	if isDirty {
		return isDirty
	}

	for _, isDirty := range candidate.dirtyStakes {
		if isDirty {
			return true
		}
	}

	return false
}

func (candidate *Candidate) GetFilteredUpdates() []*Stake {
	var updates []*Stake
	for _, update := range candidate.updates {
		// skip updates with 0 stakes
		if update.Value.Cmp(big.NewInt(0)) != 1 {
			continue
		}

		// merge updates
		for _, u := range updates {
			if u.Coin == update.Coin && u.Owner == update.Owner {
				u.Value.Add(u.Value, update.Value)
				continue
			}
		}

		updates = append(updates, update)
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

func (candidate *Candidate) SetStakeAtIndex(index int, stake *Stake) {
	candidate.stakes[index] = stake
	stake.markDirty(index)
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
	stake.markDirty(stake.index)
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
