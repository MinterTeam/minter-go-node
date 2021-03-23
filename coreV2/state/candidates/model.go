package candidates

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
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
	lock          sync.RWMutex

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
	JailedUntil              uint64
}

func (candidate *Candidate) idBytes() []byte {
	bs := make([]byte, 4)

	candidate.lock.RLock()
	defer candidate.lock.RUnlock()

	binary.LittleEndian.PutUint32(bs, candidate.ID)
	return bs
}

func (candidate *Candidate) setStatus(status byte) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	candidate.isDirty = true
	candidate.Status = status
}

func (candidate *Candidate) setOwner(address types.Address) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	candidate.isDirty = true
	candidate.OwnerAddress = address
}

func (candidate *Candidate) setCommission(commission uint32, height uint64) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	candidate.isDirty = true
	candidate.Commission = commission
	candidate.LastEditCommissionHeight = height
}

func (candidate *Candidate) jainUntil(height uint64) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	candidate.isDirty = true
	candidate.JailedUntil = height
}

func (candidate *Candidate) setReward(address types.Address) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	candidate.isDirty = true
	candidate.RewardAddress = address
}

func (candidate *Candidate) setControl(address types.Address) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	candidate.isDirty = true
	candidate.ControlAddress = address
}

func (candidate *Candidate) setPublicKey(pubKey types.Pubkey) {
	candidate.lock.Lock()
	candidate.isDirty = true
	candidate.PubKey = pubKey
	candidate.lock.Unlock()

	candidate.setTmAddress()
}

func (candidate *Candidate) addUpdate(stake *stake) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	candidate.isUpdatesDirty = true
	stake.markDirty = func(i int) {
		candidate.lock.Lock()
		defer candidate.lock.Unlock()
		candidate.isUpdatesDirty = true
	}
	candidate.updates = append(candidate.updates, stake)
}

func (candidate *Candidate) clearUpdates() {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	if len(candidate.updates) != 0 {
		candidate.isUpdatesDirty = true
	}

	candidate.updates = nil
}

func (candidate *Candidate) setTotalBipStake(totalBipValue *big.Int) {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	if totalBipValue.Cmp(candidate.totalBipStake) != 0 {
		candidate.isTotalStakeDirty = true
	}

	candidate.totalBipStake.Set(totalBipValue)
}

// GetTmAddress returns tendermint-address of a candidate
func (candidate *Candidate) GetTmAddress() types.TmAddress {
	candidate.lock.RLock()
	defer candidate.lock.RUnlock()

	return *candidate.tmAddress
}

func (candidate *Candidate) setTmAddress() {
	candidate.lock.Lock()
	defer candidate.lock.Unlock()

	var address types.TmAddress
	copy(address[:], ed25519.PubKey(candidate.PubKey[:]).Address().Bytes())

	candidate.tmAddress = &address
}

// getFilteredUpdates returns updates which is > 0 in their value + merge similar updates
func (candidate *Candidate) getFilteredUpdates() []*stake {
	candidate.lock.RLock()
	defer candidate.lock.RUnlock()

	var updates []*stake
	for _, update := range candidate.updates {
		// skip updates with 0 stakes
		update.lock.RLock()
		if update.Value.Cmp(big.NewInt(0)) != 1 {
			update.lock.RUnlock()
			continue
		}

		// merge updates
		merged := false
		for _, u := range updates {

			u.lock.Lock()
			if u.Coin == update.Coin && u.Owner == update.Owner {
				u.Value = big.NewInt(0).Add(u.Value, update.Value)
				u.lock.Unlock()

				merged = true
				break
			}
			u.lock.Unlock()
		}

		update.lock.RUnlock()

		if !merged {
			updates = append(updates, update)
		}
	}

	return updates
}

// filterUpdates filters candidate updates: remove 0-valued updates and merge similar ones
func (candidate *Candidate) filterUpdates() {

	candidate.lock.RLock()
	if len(candidate.updates) == 0 {
		candidate.lock.RUnlock()
		return
	}
	candidate.lock.RUnlock()

	updates := candidate.getFilteredUpdates()

	sort.SliceStable(updates, func(i, j int) bool {
		return updates[i].BipValue.Cmp(updates[j].BipValue) == 1
	})

	candidate.lock.Lock()
	candidate.updates = updates
	candidate.isUpdatesDirty = true
	candidate.lock.Unlock()
}

// GetTotalBipStake returns total stake value of a candidate
func (candidate *Candidate) GetTotalBipStake() *big.Int {
	candidate.lock.RLock()
	defer candidate.lock.RUnlock()

	return big.NewInt(0).Set(candidate.totalBipStake)
}

// Status returns status of a candidate
func (candidate *Candidate) GetStatus() byte {
	candidate.lock.RLock()
	defer candidate.lock.RUnlock()

	return candidate.Status
}

func (candidate *Candidate) setStakeAtIndex(index int, stake *stake, isDirty bool) {

	stake.markDirty = func(i int) {
		candidate.lock.Lock()
		defer candidate.lock.Unlock()
		candidate.dirtyStakes[i] = true
	}
	stake.index = index

	candidate.lock.Lock()
	candidate.stakes[index] = stake
	candidate.lock.Unlock()

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
	lock      sync.RWMutex
}

func (stake *stake) addValue(value *big.Int) {
	stake.markDirty(stake.index)

	stake.lock.Lock()
	defer stake.lock.Unlock()

	stake.Value = big.NewInt(0).Add(stake.Value, value)
}

func (stake *stake) subValue(value *big.Int) {
	stake.markDirty(stake.index)

	stake.lock.Lock()
	defer stake.lock.Unlock()

	stake.Value = big.NewInt(0).Sub(stake.Value, value)
}

func (stake *stake) setBipValue(value *big.Int) {
	stake.lock.Lock()
	defer stake.lock.Unlock()

	if stake.BipValue.Cmp(value) != 0 {
		stake.markDirty(stake.index)
	}

	stake.BipValue = big.NewInt(0).Set(value)
}

func (stake *stake) setValue(ret *big.Int) {
	stake.markDirty(stake.index)

	stake.lock.Lock()
	defer stake.lock.Unlock()

	stake.Value = big.NewInt(0).Set(ret)
}
