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

	mainPrefix        = 'с'
	stakesPrefix      = 's'
	stakesStatePrefix = 'q'
	totalStakePrefix  = 't'
)

type Candidates struct {
	list map[types.Pubkey]*Candidate

	iavl tree.Tree
}

func NewCandidates(iavl tree.Tree) (*Candidates, error) {
	return &Candidates{iavl: iavl}, nil
}

func (c *Candidates) Commit() error {
	panic("implement me")
}

func (c *Candidates) GetCandidateByTendermintAddress(address types.TmAddress) *Candidate {
	panic("implement me")
}

func (c *Candidates) RecalculateTotalStakeValues() {
	panic("implement me")
}

func (c *Candidates) GetNewCandidates(valCount int, height int64) []Candidate {
	panic("implement me")
}

func (c *Candidates) PunishByzantineCandidate(tmAddress types.TmAddress) {
	panic("implement me")
}

func (c *Candidates) Exists(pubkey types.Pubkey) bool {
	panic("implement me")
}

func (c *Candidates) Count() int {
	panic("implement me")
}

func (c *Candidates) IsNewCandidateStakeSufficient(coin types.CoinSymbol, stake *big.Int) bool {
	panic("implement me")
}

func (c *Candidates) Create(ownerAddress types.Address, rewardAddress types.Address, pubkey types.Pubkey, commission uint, coin types.CoinSymbol, stake *big.Int) {
	panic("implement me")
}

func (c *Candidates) GetCandidate(pubkey types.Pubkey) *Candidate {
	panic("implement me")
}

func (c *Candidates) IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, amount *big.Int) bool {
	panic("implement me")
}

func (c *Candidates) Delegate(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	panic("implement me")
}

func (c *Candidates) Edit(pubkey types.Pubkey, rewardAddress types.Address, ownerAddress types.Address) {
	panic("implement me")
}

func (c *Candidates) SetOnline(pubkey types.Pubkey) {
	panic("implement me")
}

func (c *Candidates) SetOffline(pubkey types.Pubkey) {
	panic("implement me")
}

func (c *Candidates) SubStake(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	panic("implement me")
}

func (c *Candidates) GetCandidates() []Candidate {
	panic("implement me")
}

func (c *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	panic("implement me")
}

func (c *Candidates) GetStakes(pubkey types.Pubkey) []Stake {
	panic("implement me")
}

func (c *Candidates) StakesCount(pubkey types.Pubkey) int {
	panic("implement me")
}

func (c *Candidates) GetStakeOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinSymbol) *big.Int {
	panic("implement me")
}
