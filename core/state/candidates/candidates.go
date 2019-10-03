package candidates

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
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
	list   map[types.Pubkey]*Candidate
	loaded bool

	iavl tree.Tree
}

func NewCandidates(iavl tree.Tree) (*Candidates, error) {
	return &Candidates{iavl: iavl}, nil
}

func (c *Candidates) Commit() error {
	if !c.loaded {
		return nil
	}

	// todo commit

	return nil
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
	c.loadCandidates()
	_, exists := c.list[pubkey]

	return exists
}

func (c *Candidates) Count() int {
	c.loadCandidates()

	return len(c.list)
}

func (c *Candidates) IsNewCandidateStakeSufficient(coin types.CoinSymbol, stake *big.Int) bool {
	panic("implement me")
}

func (c *Candidates) Create(ownerAddress types.Address, rewardAddress types.Address, pubkey types.Pubkey, commission uint, coin types.CoinSymbol, stake *big.Int) {
	panic("implement me")
}

func (c *Candidates) GetCandidate(pubkey types.Pubkey) *Candidate {
	c.loadCandidates()

	return c.list[pubkey]
}

func (c *Candidates) IsDelegatorStakeSufficient(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, amount *big.Int) bool {
	panic("implement me")
}

func (c *Candidates) Delegate(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	panic("implement me")
}

func (c *Candidates) Edit(pubkey types.Pubkey, rewardAddress types.Address, ownerAddress types.Address) {
	c.loadCandidates()
	c.list[pubkey].setOwner(ownerAddress)
	c.list[pubkey].setReward(rewardAddress)
}

func (c *Candidates) SetOnline(pubkey types.Pubkey) {
	c.loadCandidates()
	c.list[pubkey].setStatus(CandidateStatusOnline)
}

func (c *Candidates) SetOffline(pubkey types.Pubkey) {
	c.loadCandidates()
	c.list[pubkey].setStatus(CandidateStatusOffline)
}

func (c *Candidates) SubStake(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	panic("implement me")
}

func (c *Candidates) GetCandidates() []Candidate {
	c.loadCandidates()
	var candidates []Candidate
	for _, candidate := range c.list {
		candidates = append(candidates, *candidate)
	}

	return candidates
}

func (c *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	c.loadCandidates()
	candidate := c.list[pubkey]
	if candidate.totalBipStake == nil {
		path := []byte{mainPrefix}
		path = append(path, pubkey[:]...)
		path = append(path, totalStakePrefix)
		_, enc := c.iavl.Get(path)
		if len(enc) == 0 {
			candidate.totalBipStake = big.NewInt(0)
			return big.NewInt(0)
		}

		candidate.totalBipStake = big.NewInt(0).SetBytes(enc)
	}

	return candidate.totalBipStake
}

func (c *Candidates) GetStakes(pubkey types.Pubkey) []*Stake {
	state := c.getStakeState(pubkey)

	var stakes []*Stake
	i := 0
	currentIndex := state.Tail
	for i < state.Count && currentIndex != -1 {
		stake := c.getStakeAtIndex(pubkey, currentIndex)
		stakes = append(stakes, stake)

		currentIndex = stake.PrevStakeIndex
	}

	return stakes
}

func (c *Candidates) StakesCount(pubkey types.Pubkey) int {
	state := c.getStakeState(pubkey)

	return state.Count
}

func (c *Candidates) GetStakeOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinSymbol) *Stake {
	stakes := c.GetStakes(pubkey)
	for _, stake := range stakes {
		if stake.Owner == address && stake.Coin == coin {
			return stake
		}
	}

	return nil
}

func (c *Candidates) GetStakeValueOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinSymbol) *big.Int {
	stake := c.GetStakeOfAddress(pubkey, address, coin)
	if stake == nil {
		return nil
	}

	return stake.Value
}

func (c *Candidates) GetCandidateOwner(pubkey types.Pubkey) types.Address {
	c.loadCandidates()

	return c.list[pubkey].OwnerAddress
}

func (c *Candidates) loadCandidates() {
	if c.loaded {
		return
	}

	c.loaded = true

	path := []byte{mainPrefix}
	_, enc := c.iavl.Get(path)
	if len(enc) == 0 {
		c.list = map[types.Pubkey]*Candidate{}
		return
	}

	var candidates []Candidate
	if err := rlp.DecodeBytes(enc, candidates); err != nil {
		panic(fmt.Sprintf("failed to decode candidates: %s", err))
		return
	}

	list := map[types.Pubkey]*Candidate{}
	for _, candidate := range candidates {
		list[candidate.PubKey] = &candidate
	}
}

func (c *Candidates) getStakeState(pubkey types.Pubkey) *stakesState {
	c.loadCandidates()
	candidate := c.list[pubkey]
	if candidate.stakesState == nil {
		path := []byte{mainPrefix}
		path = append(path, pubkey[:]...)
		path = append(path, totalStakePrefix)
		_, enc := c.iavl.Get(path)
		if len(enc) == 0 {
			candidate.stakesState = &stakesState{
				Count:   0,
				Tail:    0,
				isDirty: false,
			}
			return candidate.stakesState
		}

		var stakesState stakesState
		if err := rlp.DecodeBytes(enc, &stakesState); err != nil {
			panic(fmt.Sprintf("failed to decode stakes state: %s", err))
		}

		candidate.stakesState = &stakesState
	}

	return candidate.stakesState
}

func (c *Candidates) getStakeAtIndex(pubkey types.Pubkey, index int) *Stake {
	c.loadCandidates()
	candidate := c.list[pubkey]
	if candidate.stakes[index] == nil {
		path := []byte{mainPrefix}
		path = append(path, pubkey[:]...)
		path = append(path, stakesPrefix)
		path = append(path, []byte(fmt.Sprintf("%d", index))...)
		_, enc := c.iavl.Get(path)
		if len(enc) == 0 {
			return nil
		}

		var stake Stake
		if err := rlp.DecodeBytes(enc, &stake); err != nil {
			panic(fmt.Sprintf("failed to decode stake: %s", err))
		}

		candidate.stakes[index] = &stake
	}

	return candidate.stakes[index]
}
