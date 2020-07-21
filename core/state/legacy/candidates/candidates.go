package candidates

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
	"sync"
)

const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02

	UnbondPeriod              = 518400
	MaxDelegatorsPerCandidate = 1000

	mainPrefix       = 'c'
	stakesPrefix     = 's'
	totalStakePrefix = 't'
	updatesPrefix    = 'u'
)

type Candidates struct {
	list map[types.Pubkey]*Candidate

	iavl tree.MTree
	bus  *bus.Bus

	lock   sync.RWMutex
	loaded bool
}

func NewCandidates(bus *bus.Bus, iavl tree.MTree) (*Candidates, error) {
	candidates := &Candidates{iavl: iavl, bus: bus}

	return candidates, nil
}

func (c *Candidates) GetNewCandidates(valCount int) []Candidate {
	var result []Candidate

	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		if candidate.Status == CandidateStatusOffline {
			continue
		}

		if candidate.totalBipStake.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		result = append(result, *candidate)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].totalBipStake.Cmp(result[j].totalBipStake) == 1
	})

	if len(result) > valCount {
		result = result[:valCount]
	}

	return result
}

func (c *Candidates) Create(ownerAddress types.Address, rewardAddress types.Address, pubkey types.Pubkey, commission uint) {
	candidate := &Candidate{
		PubKey:            pubkey,
		RewardAddress:     rewardAddress,
		OwnerAddress:      ownerAddress,
		Commission:        commission,
		Status:            CandidateStatusOffline,
		totalBipStake:     big.NewInt(0),
		stakes:            [MaxDelegatorsPerCandidate]*Stake{},
		isDirty:           true,
		isTotalStakeDirty: true,
	}

	candidate.setTmAddress()
	c.setToMap(pubkey, candidate)
}

func (c *Candidates) GetCandidateByTendermintAddress(address types.TmAddress) *Candidate {
	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		if candidate.GetTmAddress() == address {
			return candidate
		}
	}

	return nil
}

func (c *Candidates) Exists(pubkey types.Pubkey) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, exists := c.list[pubkey]

	return exists
}

func (c *Candidates) Count() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.list)
}

func (c *Candidates) GetCandidate(pubkey types.Pubkey) *Candidate {
	return c.getFromMap(pubkey)
}

func (c *Candidates) GetCandidates() []*Candidate {
	var candidates []*Candidate
	for _, pubkey := range c.getOrderedCandidates() {
		candidates = append(candidates, c.getFromMap(pubkey))
	}

	return candidates
}

func (c *Candidates) GetTotalStake(pubkey types.Pubkey) *big.Int {
	candidate := c.getFromMap(pubkey)
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
	candidate := c.GetCandidate(pubkey)

	var stakes []*Stake
	for i := 0; i < MaxDelegatorsPerCandidate; i++ {
		stake := candidate.stakes[i]
		if stake == nil {
			continue
		}
		stakes = append(stakes, stake)
	}

	return stakes
}

func (c *Candidates) StakesCount(pubkey types.Pubkey) int {
	return c.GetCandidate(pubkey).stakesCount
}

func (c *Candidates) GetStakeOfAddress(pubkey types.Pubkey, address types.Address, coin types.CoinSymbol) *Stake {
	candidate := c.GetCandidate(pubkey)
	for _, stake := range candidate.stakes {
		if stake == nil {
			continue
		}

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
	return c.getFromMap(pubkey).OwnerAddress
}

func (c *Candidates) LoadCandidates() {
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

	var candidates []*Candidate
	if err := rlp.DecodeBytes(enc, &candidates); err != nil {
		panic(fmt.Sprintf("failed to decode candidates: %s", err))
	}

	c.list = map[types.Pubkey]*Candidate{}
	for _, candidate := range candidates {
		// load total stake
		path = append([]byte{mainPrefix}, candidate.PubKey.Bytes()...)
		path = append(path, totalStakePrefix)
		_, enc = c.iavl.Get(path)
		if len(enc) == 0 {
			candidate.totalBipStake = big.NewInt(0)
		} else {
			candidate.totalBipStake = big.NewInt(0).SetBytes(enc)
		}

		candidate.setTmAddress()
		c.setToMap(candidate.PubKey, candidate)
	}
}

func (c *Candidates) LoadStakes() {
	for pubkey := range c.list {
		c.LoadStakesOfCandidate(pubkey)
	}
}

func (c *Candidates) Export(state *types.AppState, coinsMap map[types.CoinSymbol]types.Coin) {
	c.LoadCandidates()
	c.LoadStakes()

	candidates, maxID := c.GetCandidates(), uint(1)
	for _, candidate := range candidates {
		candidateStakes := c.GetStakes(candidate.PubKey)
		stakes := make([]types.Stake, len(candidateStakes))
		for i, s := range candidateStakes {
			stakes[i] = types.Stake{
				Owner:    s.Owner,
				Coin:     coinsMap[s.Coin].ID,
				Value:    s.Value.String(),
				BipValue: s.BipValue.String(),
			}
		}

		updates := make([]types.Stake, len(candidate.updates))
		for i, u := range candidate.updates {
			updates[i] = types.Stake{
				Owner:    u.Owner,
				Coin:     coinsMap[u.Coin].ID,
				Value:    u.Value.String(),
				BipValue: u.BipValue.String(),
			}
		}

		state.Candidates = append(state.Candidates, types.Candidate{
			ID:            maxID,
			RewardAddress: candidate.RewardAddress,
			OwnerAddress:  candidate.OwnerAddress,
			TotalBipStake: candidate.GetTotalBipStake().String(),
			PubKey:        candidate.PubKey,
			Commission:    candidate.Commission,
			Status:        candidate.Status,
			Updates:       updates,
			Stakes:        stakes,
		})

		maxID++
	}

}

func (c *Candidates) getOrderedCandidates() []types.Pubkey {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var keys []types.Pubkey
	for _, candidate := range c.list {
		keys = append(keys, candidate.PubKey)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (c *Candidates) getFromMap(pubkey types.Pubkey) *Candidate {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[pubkey]
}

func (c *Candidates) setToMap(pubkey types.Pubkey, model *Candidate) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[pubkey] = model
}

func (c *Candidates) SetTotalStake(pubkey types.Pubkey, stake *big.Int) {
	c.GetCandidate(pubkey).setTotalBipStake(stake)
}

func (c *Candidates) LoadStakesOfCandidate(pubkey types.Pubkey) {
	candidate := c.GetCandidate(pubkey)

	// load stakes
	stakesCount := 0
	for index := 0; index < MaxDelegatorsPerCandidate; index++ {
		path := []byte{mainPrefix}
		path = append(path, candidate.PubKey.Bytes()...)
		path = append(path, stakesPrefix)
		path = append(path, []byte(fmt.Sprintf("%d", index))...)
		_, enc := c.iavl.Get(path)
		if len(enc) == 0 {
			candidate.stakes[index] = nil
			continue
		}

		stake := &Stake{}
		if err := rlp.DecodeBytes(enc, stake); err != nil {
			panic(fmt.Sprintf("failed to decode stake: %s", err))
		}

		candidate.SetStakeAtIndex(index, stake, false)

		stakesCount++
	}

	candidate.stakesCount = stakesCount

	// load updates
	path := []byte{mainPrefix}
	path = append(path, candidate.PubKey.Bytes()...)
	path = append(path, updatesPrefix)
	_, enc := c.iavl.Get(path)
	if len(enc) == 0 {
		candidate.updates = nil
	} else {
		var updates []*Stake
		if err := rlp.DecodeBytes(enc, &updates); err != nil {
			panic(fmt.Sprintf("failed to decode updated: %s", err))
		}

		for _, update := range updates {
			update.markDirty = (func(candidate *Candidate) func(int) {
				return func(i int) {
					candidate.isUpdatesDirty = true
				}
			})(candidate)
		}

		candidate.updates = updates
	}

	// load total stake
	path = append([]byte{mainPrefix}, candidate.PubKey.Bytes()...)
	path = append(path, totalStakePrefix)
	_, enc = c.iavl.Get(path)
	if len(enc) == 0 {
		candidate.totalBipStake = big.NewInt(0)
	} else {
		candidate.totalBipStake = big.NewInt(0).SetBytes(enc)
	}

	candidate.setTmAddress()
	c.setToMap(candidate.PubKey, candidate)
}
