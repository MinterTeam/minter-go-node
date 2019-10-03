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
	stakes        [MaxDelegatorsPerCandidate]*Stake
	tmAddress     *types.TmAddress
}

type Stake struct {
	Owner    types.Address
	Coin     types.CoinSymbol
	Value    *big.Int
	BipValue *big.Int
}
