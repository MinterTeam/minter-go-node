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

func (v *Candidates) Exists(pubkey types.Pubkey) bool {
	panic("implement me")
}

func (v *Candidates) Count() int {
	panic("implement me")
}

func (v *Candidates) IsNewCandidateStakeSufficient(coin types.CoinSymbol, stake *big.Int) bool {
	panic("implement me")
}

func (v *Candidates) Create(ownerAddress types.Address, rewardAddress types.Address, pubkey types.Pubkey, commission uint, coin types.CoinSymbol, stake *big.Int) {
	panic("implement me")
}

func (v *Candidates) GetCandidate(pubkey []byte) *Candidate {
	panic("implement me")
}

func (v *Candidates) IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, amount *big.Int) bool {
	panic("implement me")
}

func (v *Candidates) Delegate(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	panic("implement me")
}

func (v *Candidates) Edit(pubkey types.Pubkey, rewardAddress types.Address, ownerAddress types.Address) {
	panic("implement me")
}

func (v *Candidates) SetOnline(pubkey types.Pubkey) {
	panic("implement me")
}

func (v *Candidates) SetOffline(pubkey types.Pubkey) {
	panic("implement me")
}

func (v *Candidates) SubStake(address types.Address, oubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	panic("implement me")
}

func (v *Candidates) GetCandidates() []Candidate {
	panic("implement me")
}

type Candidate struct {
	RewardAddress  types.Address
	OwnerAddress   types.Address
	TotalBipStake  *big.Int
	PubKey         types.Pubkey
	Commission     uint
	CreatedAtBlock uint
	Status         byte

	tmAddress *[20]byte
}

func (candidate *Candidate) GetStakeOfAddress(address types.Address, coin types.CoinSymbol) *big.Int {
	panic("implement me")
}

func (candidate *Candidate) StakesCount() int {
	panic("implement me")
}

func (candidate *Candidate) Stakes() []Stake {
	panic("implement me")
}

type Stake struct {
	Owner    types.Address
	Coin     types.CoinSymbol
	Value    *big.Int
	BipValue *big.Int
}