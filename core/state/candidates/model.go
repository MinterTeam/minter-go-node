package candidates

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
	"sort"
	"sync"
)

type pubkeyID struct {
	PubKey types.Pubkey
	ID     uint32
}

// Candidate represents candidate object which is stored on disk
type Candidate struct {
	totalBipStake *big.Int
	stakesCount   int
	stakes        [MaxDelegatorsPerCandidate]*stake
	updates       []*stake
	tmAddress     *types.TmAddress
	sync.RWMutex

	isDirty           bool
	isTotalStakeDirty bool
	isUpdatesDirty    bool
	dirtyStakes       [MaxDelegatorsPerCandidate]bool

	PubKey                   types.Pubkey
	RewardAddress            types.Address
	OwnerAddress             types.Address
	ControlAddress           types.Address
	Commission               uint32
	Status                   byte
	ID                       uint32
	LastEditCommissionHeight uint64
}

func (candidate *Candidate) idBytes() []byte {
	bs := make([]byte, 4)

	candidate.RLock()
	defer candidate.RUnlock()

	binary.LittleEndian.PutUint32(bs, candidate.ID)
	return bs
}

func (candidate *Candidate) setStatus(status byte) {
	candidate.Lock()
	defer candidate.Unlock()

	candidate.isDirty = true
	candidate.Status = status
}

func (candidate *Candidate) setOwner(address types.Address) {
	candidate.Lock()
	defer candidate.Unlock()

	candidate.isDirty = true
	candidate.OwnerAddress = address
}

func (candidate *Candidate) setCommission(commission uint32, height uint64) {
	candidate.Lock()
	defer candidate.Unlock()

	candidate.isDirty = true
	candidate.Commission = commission
	candidate.LastEditCommissionHeight = height
}

func (candidate *Candidate) setReward(address types.Address) {
	candidate.Lock()
	defer candidate.Unlock()

	candidate.isDirty = true
	candidate.RewardAddress = address
}

func (candidate *Candidate) setControl(address types.Address) {
	candidate.Lock()
	defer candidate.Unlock()

	candidate.isDirty = true
	candidate.ControlAddress = address
}

func (candidate *Candidate) setPublicKey(pubKey types.Pubkey) {
	candidate.Lock()
	candidate.isDirty = true
	candidate.PubKey = pubKey
	candidate.Unlock()

	candidate.setTmAddress()
}

func (candidate *Candidate) addUpdate(stake *stake) {
	candidate.Lock()
	defer candidate.Unlock()

	candidate.isUpdatesDirty = true
	stake.markDirty = func(i int) {
		candidate.Lock()
		defer candidate.Unlock()
		candidate.isUpdatesDirty = true
	}
	candidate.updates = append(candidate.updates, stake)
}

func (candidate *Candidate) clearUpdates() {
	candidate.Lock()
	defer candidate.Unlock()

	if len(candidate.updates) != 0 {
		candidate.isUpdatesDirty = true
	}

	candidate.updates = nil
}

func (candidate *Candidate) setTotalBipStake(totalBipValue *big.Int) {
	candidate.Lock()
	defer candidate.Unlock()

	if totalBipValue.Cmp(candidate.totalBipStake) != 0 {
		candidate.isTotalStakeDirty = true
	}

	candidate.totalBipStake.Set(totalBipValue)
}

// GetTmAddress returns tendermint-address of a candidate
func (candidate *Candidate) GetTmAddress() types.TmAddress {
	candidate.RLock()
	defer candidate.RUnlock()

	return *candidate.tmAddress
}

func (candidate *Candidate) setTmAddress() {
	candidate.Lock()
	defer candidate.Unlock()

	var address types.TmAddress
	copy(address[:], ed25519.PubKey(candidate.PubKey[:]).Address().Bytes())

	candidate.tmAddress = &address
}

// getFilteredUpdates returns updates which is > 0 in their value + merge similar updates
func (candidate *Candidate) getFilteredUpdates() []*stake {
	candidate.RLock()
	defer candidate.RUnlock()

	var updates []*stake
	for _, update := range candidate.updates {
		// skip updates with 0 stakes
		update.RLock()
		if update.Value.Cmp(big.NewInt(0)) != 1 {
			update.RUnlock()
			continue
		}

		// merge updates
		merged := false
		for _, u := range updates {

			u.Lock()
			if u.Coin == update.Coin && u.Owner == update.Owner {
				u.Value = big.NewInt(0).Add(u.Value, update.Value)
				u.Unlock()

				merged = true
				break
			}
			u.Unlock()
		}

		update.RUnlock()

		if !merged {
			updates = append(updates, update)
		}
	}

	return updates
}

// filterUpdates filters candidate updates: remove 0-valued updates and merge similar ones
func (candidate *Candidate) filterUpdates() {

	candidate.RLock()
	if len(candidate.updates) == 0 {
		candidate.RUnlock()
		return
	}
	candidate.RUnlock()

	updates := candidate.getFilteredUpdates()

	sort.SliceStable(updates, func(i, j int) bool {
		return updates[i].BipValue.Cmp(updates[j].BipValue) == 1
	})

	candidate.Lock()
	candidate.updates = updates
	candidate.isUpdatesDirty = true
	candidate.Unlock()
}

// GetTotalBipStake returns total stake value of a candidate
func (candidate *Candidate) GetTotalBipStake() *big.Int {
	candidate.RLock()
	defer candidate.RUnlock()

	return big.NewInt(0).Set(candidate.totalBipStake)
}

func (candidate *Candidate) setStakeAtIndex(index int, stake *stake, isDirty bool) {

	stake.markDirty = func(i int) {
		candidate.Lock()
		defer candidate.Unlock()
		candidate.dirtyStakes[i] = true
	}
	stake.index = index

	candidate.Lock()
	candidate.stakes[index] = stake
	candidate.Unlock()

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
	sync.RWMutex
}

func (stake *stake) addValue(value *big.Int) {
	stake.markDirty(stake.index)

	stake.Lock()
	defer stake.Unlock()

	stake.Value = big.NewInt(0).Add(stake.Value, value)
}

func (stake *stake) subValue(value *big.Int) {
	stake.markDirty(stake.index)

	stake.Lock()
	defer stake.Unlock()

	stake.Value = big.NewInt(0).Sub(stake.Value, value)
}

func (stake *stake) setBipValue(value *big.Int) {
	stake.Lock()
	defer stake.Unlock()

	if stake.BipValue.Cmp(value) != 0 {
		stake.markDirty(stake.index)
	}

	stake.BipValue = big.NewInt(0).Set(value)
}

func (stake *stake) setValue(ret *big.Int) {
	stake.markDirty(stake.index)

	stake.Lock()
	defer stake.Unlock()

	stake.Value = big.NewInt(0).Set(ret)
}
