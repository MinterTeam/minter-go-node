package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
)

const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02

	UnbondPeriod              = 518400
	MaxDelegatorsPerCandidate = 1000

	mainPrefix       = 'с'
	stakesPrefix     = 's'
	totalStakePrefix = 't'
)

type Candidates struct {
	iavl tree.Tree
}

func NewCandidates(iavl tree.Tree) (*Candidates, error) {
	return &Candidates{iavl: iavl}, nil
}

func (v *Candidates) Commit() error {
	panic("implement me")
}

func (v *Candidates) GetCandidateByTendermintAddress(address types.TmAddress) *Candidate {
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

func (v *Candidates) PunishByzantineCandidate(tmAddress types.TmAddress) {
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

func (v *Candidates) SubStake(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	panic("implement me")
}

func (v *Candidates) GetCandidates() []Candidate {
	panic("implement me")
}

func (v *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	panic("implement me")
}

func (v *Candidates) GetStakes(pubkey types.Pubkey) []Stake {
	panic("implement me")
}

func (v *Candidates) StakesCount(pubkey types.Pubkey) int {
	panic("implement me")
}

func (v *Candidates) GetStakeOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinSymbol) *big.Int {
	panic("implement me")
}
