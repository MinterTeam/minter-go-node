package candidates

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
	"sort"
)

type pubkeyID struct {
	PubKey types.Pubkey
	ID     uint32
}

// Candidate represents candidate object which is stored on disk
type Candidate struct {
	PubKey         types.Pubkey
	RewardAddress  types.Address
	OwnerAddress   types.Address
	ControlAddress types.Address
	Commission     uint
	Status         byte
	ID             uint32

	totalBipStake *big.Int
	stakesCount   int
	stakes        [MaxDelegatorsPerCandidate]*stake
	updates       []*stake
	tmAddress     *types.TmAddress

	isDirty           bool
	isTotalStakeDirty bool
	isUpdatesDirty    bool
	dirtyStakes       [MaxDelegatorsPerCandidate]bool
}

func (candidate *Candidate) idBytes() []byte {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, candidate.ID)
	return bs
}

func (candidate *Candidate) setStatus(status byte) {
	candidate.isDirty = true
	candidate.Status = status
}

func (candidate *Candidate) setOwner(address types.Address) {
	candidate.isDirty = true
	candidate.OwnerAddress = address
}

func (candidate *Candidate) setReward(address types.Address) {
	candidate.isDirty = true
	candidate.RewardAddress = address
}

func (candidate *Candidate) setControl(address types.Address) {
	candidate.isDirty = true
	candidate.ControlAddress = address
}

func (candidate *Candidate) addUpdate(stake *stake) {
	candidate.isUpdatesDirty = true
	stake.markDirty = func(i int) {
		candidate.isUpdatesDirty = true
	}
	candidate.updates = append(candidate.updates, stake)
}

func (candidate *Candidate) clearUpdates() {
	if len(candidate.updates) != 0 {
		candidate.isUpdatesDirty = true
	}

	candidate.updates = nil
}

func (candidate *Candidate) setTotalBipStake(totalBipValue *big.Int) {
	if totalBipValue.Cmp(candidate.totalBipStake) != 0 {
		candidate.isTotalStakeDirty = true
	}

	candidate.totalBipStake.Set(totalBipValue)
}

// GetTmAddress returns tendermint-address of a candidate
func (candidate *Candidate) GetTmAddress() types.TmAddress {
	return *candidate.tmAddress
}

func (candidate *Candidate) setTmAddress() {
	// set tm address
	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], candidate.PubKey[:])

	var address types.TmAddress
	copy(address[:], pubkey.Address().Bytes())

	candidate.tmAddress = &address
}

// GetFilteredUpdates returns updates which is > 0 in their value + merge similar updates
func (candidate *Candidate) GetFilteredUpdates() []*stake {
	var updates []*stake
	for _, update := range candidate.updates {
		// skip updates with 0 stakes
		if update.Value.Cmp(big.NewInt(0)) != 1 {
			continue
		}

		// merge updates
		merged := false
		for _, u := range updates {
			if u.Coin == update.Coin && u.Owner == update.Owner {
				u.Value.Add(u.Value, update.Value)
				merged = true
				break
			}
		}

		if !merged {
			updates = append(updates, update)
		}
	}

	return updates
}

// FilterUpdates filters candidate updates: remove 0-valued updates and merge similar ones
func (candidate *Candidate) FilterUpdates() {
	var updates []*stake
	for _, update := range candidate.updates {
		// skip updates with 0 stakes
		if update.Value.Cmp(big.NewInt(0)) != 1 {
			continue
		}

		// merge updates
		merged := false
		for _, u := range updates {
			if u.Coin == update.Coin && u.Owner == update.Owner {
				u.Value.Add(u.Value, update.Value)
				merged = true
				break
			}
		}

		if !merged {
			updates = append(updates, update)
		}
	}

	sort.SliceStable(updates, func(i, j int) bool {
		return updates[i].BipValue.Cmp(updates[j].BipValue) == 1
	})

	candidate.updates = updates
	candidate.isUpdatesDirty = true
}

// GetTotalBipStake returns total stake value of a candidate
func (candidate *Candidate) GetTotalBipStake() *big.Int {
	return big.NewInt(0).Set(candidate.totalBipStake)
}

func (candidate *Candidate) setStakeAtIndex(index int, stake *stake, isDirty bool) {
	stake.markDirty = func(i int) {
		candidate.dirtyStakes[i] = true
	}
	stake.index = index

	candidate.stakes[index] = stake

	if isDirty {
		stake.markDirty(index)
	}
}

type stake struct {
	Owner    types.Address
	Coin     types.CoinID
	Value    *big.Int
	BipValue *big.Int

	index     int
	markDirty func(int)
}

func (stake *stake) addValue(value *big.Int) {
	stake.markDirty(stake.index)
	stake.Value.Add(stake.Value, value)
}

func (stake *stake) subValue(value *big.Int) {
	stake.markDirty(stake.index)
	stake.Value.Sub(stake.Value, value)
}

func (stake *stake) setBipValue(value *big.Int) {
	if stake.BipValue.Cmp(value) != 0 {
		stake.markDirty(stake.index)
	}

	stake.BipValue.Set(value)
}

func (stake *stake) setValue(ret *big.Int) {
	stake.markDirty(stake.index)
	stake.Value.Set(ret)
}
