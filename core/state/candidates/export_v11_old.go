package candidates

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

type Candidate11 struct {
	PubKey        types.Pubkey
	RewardAddress types.Address
	OwnerAddress  types.Address
	Commission    uint
	Status        byte

	totalBipStake *big.Int
	stakesCount   int
	stakes        [MaxDelegatorsPerCandidate]*Stake
	updates       []*Stake
	tmAddress     *types.TmAddress

	isDirty           bool
	isTotalStakeDirty bool
	isUpdatesDirty    bool
	dirtyStakes       [MaxDelegatorsPerCandidate]bool
}

func (candidate *Candidate11) setTmAddress() {
	// set tm address
	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], candidate.PubKey[:])

	var address types.TmAddress
	copy(address[:], pubkey.Address().Bytes())

	candidate.tmAddress = &address
}

func (c *Candidates) Export11To12(state *types.AppState) {
	c.LoadCandidates11()
	c.LoadStakes11()

	candidates := c.GetCandidates()
	for _, candidate := range candidates {
		candidateStakes := c.GetStakes(candidate.PubKey)
		stakes := make([]types.Stake, len(candidateStakes))
		for i, s := range candidateStakes {
			stakes[i] = types.Stake{
				Owner:    s.Owner,
				Coin:     s.Coin,
				Value:    s.Value.String(),
				BipValue: s.BipValue.String(),
			}
		}

		updates := make([]types.Stake, len(candidate.updates))
		for i, u := range candidate.updates {
			updates[i] = types.Stake{
				Owner:    u.Owner,
				Coin:     u.Coin,
				Value:    u.Value.String(),
				BipValue: u.BipValue.String(),
			}
		}

		state.Candidates = append(state.Candidates, types.Candidate{
			ID:             c.getOrNewID(candidate.PubKey),
			RewardAddress:  candidate.RewardAddress,
			OwnerAddress:   candidate.OwnerAddress,
			ControlAddress: candidate.ControlAddress,
			TotalBipStake:  candidate.GetTotalBipStake().String(),
			PubKey:         candidate.PubKey,
			Commission:     candidate.Commission,
			Status:         candidate.Status,
			Updates:        updates,
			Stakes:         stakes,
		})
	}

	return
}

func (c *Candidates) LoadCandidates11() {
	if c.loaded {
		return
	}
	c.loaded = true

	path := []byte{mainPrefix}
	_, enc := c.iavl.Get(path)
	if len(enc) == 0 {
		c.list = map[uint]*Candidate{}
		return
	}

	var candidates []*Candidate11
	if err := rlp.DecodeBytes(enc, &candidates); err != nil {
		panic(fmt.Sprintf("failed to decode candidates: %s", err))
	}

	c.list = map[uint]*Candidate{}
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
		c.setToMap(candidate.PubKey, &Candidate{
			PubKey:         candidate.PubKey,
			RewardAddress:  candidate.RewardAddress,
			OwnerAddress:   candidate.OwnerAddress,
			ControlAddress: types.HexToAddress("Mxf8a88c0ed668e995639cb8a08ffb38541bcc9045"),
			Commission:     candidate.Commission,
			Status:         candidate.Status,
			ID:             c.getOrNewID(candidate.PubKey),
		})
	}
}

func (c *Candidates) LoadStakes11() {
	for _, candidate := range c.list {
		c.LoadStakesOfCandidate11(candidate.PubKey)
	}
}

func (c *Candidates) LoadStakesOfCandidate11(pubkey types.Pubkey) {
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
