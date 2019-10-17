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
	stakesState   *stakesState
	stakes        [MaxDelegatorsPerCandidate]*Stake
	updates       []*Stake
	tmAddress     *types.TmAddress

	isDirty           bool
	isTotalStakeDirty bool
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
	if candidate.tmAddress != nil {
		return *candidate.tmAddress
	}

	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], candidate.PubKey[:])

	var address types.TmAddress
	copy(address[:], pubkey.Address().Bytes())

	candidate.tmAddress = &address

	return address
}

type Stake struct {
	Owner          types.Address
	Coin           types.CoinSymbol
	Value          *big.Int
	BipValue       *big.Int
	PrevStakeIndex int

	isDirty bool
}

func (stake *Stake) addValue(value *big.Int) {
	stake.isDirty = true
	stake.Value.Add(stake.Value, value)
}

func (stake *Stake) subValue(value *big.Int) {
	stake.isDirty = true
	stake.Value.Sub(stake.Value, value)
}

func (stake *Stake) setBipValue(value *big.Int) {
	stake.isDirty = true
	stake.BipValue.Set(value)
}

type stakesState struct {
	Count int
	Tail  int

	isDirty bool
}

func (state *stakesState) SetTail(tail int) {
	state.Tail = tail
	state.isDirty = true
}
