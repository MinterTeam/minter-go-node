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

	isDirty bool
}

type Stake struct {
	Owner    types.Address
	Coin     types.CoinSymbol
	Value    *big.Int
	BipValue *big.Int

	isDirty bool
}

type stakesState struct {
	Tail int32

	isDirty bool
}
