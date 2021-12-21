package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Candidates interface {
	GetStakes(types.Pubkey) []*Stake
	Punish(uint64, types.TmAddress)
	ID(types.Pubkey) uint32
	SetOffline(types.Pubkey)
	GetCandidate(types.Pubkey) *Candidate
	GetCandidateByTendermintAddress(types.TmAddress) *Candidate
}

type Stake struct {
	Owner    types.Address
	Value    *big.Int
	Coin     types.CoinID
	BipValue *big.Int
	AddValue func(*big.Int)
}

type Candidate struct {
	ID             uint32
	PubKey         types.Pubkey
	RewardAddress  types.Address
	OwnerAddress   types.Address
	ControlAddress types.Address
	Commission     uint32
	Status         byte
	AddUpdate      func(types.CoinID, *big.Int, *big.Int, types.Address)
}
