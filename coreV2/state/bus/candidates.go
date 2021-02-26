package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Candidates interface {
	GetStakes(types.Pubkey) []Stake
	Punish(uint64, types.TmAddress) *big.Int
	GetCandidate(types.Pubkey) *Candidate
	SetOffline(types.Pubkey)
	GetCandidateByTendermintAddress(types.TmAddress) *Candidate
}

type Stake struct {
	Owner    types.Address
	Value    *big.Int
	Coin     types.CoinID
	BipValue *big.Int
}

type Candidate struct {
	ID             uint32
	PubKey         types.Pubkey
	RewardAddress  types.Address
	OwnerAddress   types.Address
	ControlAddress types.Address
	Commission     uint32
	Status         byte
}
