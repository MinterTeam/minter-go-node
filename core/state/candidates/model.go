package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/types"
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

type Stake struct {
	Owner          types.Address
	Coin           types.CoinSymbol
	Value          *big.Int
	BipValue       *big.Int
	PrevStakeIndex int

	isDirty bool
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
