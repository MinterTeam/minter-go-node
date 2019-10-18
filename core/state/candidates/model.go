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
	candidate.isDirty = true
	candidate.updates = append(candidate.updates, stake)
}

func (candidate *Candidate) clearUpdates() {
	candidate.isDirty = true
	candidate.updates = nil
}

func (candidate *Candidate) setTotalBipValue(totalBipValue *big.Int) {
	candidate.isDirty = true
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
