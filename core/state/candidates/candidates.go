package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
	"math/big"
)

const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02

	UnbondPeriod              = 518400
	MaxDelegatorsPerCandidate = 1000
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
	panic("implement me")
}

func (v *Candidates) RecalculateTotalStakeValues() {
	panic("implement me")
}

func (v *Candidates) Clear() {
	panic("implement me") // clear stakes and candidates
}

func (v *Candidates) GetNewCandidates(valCount int, height int64) []Candidate {
	panic("implement me")
}

func (v *Candidates) PunishByzantineCandidate(tmAddress [20]byte) {
	panic("implement me")
}

func (v *Candidates) PayRewards() {
	panic("implement me")
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
