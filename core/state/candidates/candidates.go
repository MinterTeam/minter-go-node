package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
	"math/big"
)

const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02
)

type Candidates struct {
	db db.DB
}

func NewCandidates(db db.DB) (*Candidates, error) {
	return &Candidates{db: db}, nil
}

func (v *Candidates) Commit() error {
	panic("implement me")
}

func (v *Candidates) GetCandidateByTendermintAddress(address [20]byte) *Candidate {

}

type Candidate struct {
	RewardAddress  types.Address
	OwnerAddress   types.Address
	TotalBipStake  *big.Int
	PubKey         types.Pubkey
	Commission     uint
	Stakes         []Stake
	CreatedAtBlock uint
	Status         byte

	tmAddress *[20]byte
}