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

type movedStake struct {
	ToHeight uint64
	To       types.Pubkey
	Owner    types.Address
	Coin     types.CoinID
	Value    *big.Int
}

// Candidate represents candidate object which is stored on disk
type Candidate struct {
	PubKey         types.Pubkey
	RewardAddress  types.Address
	OwnerAddress   types.Address
	ControlAddress types.Address
	Commission     uint32
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

	MovedStakes []*movedStake `rlp:"tail"` // must be on last field
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

func (candidate *Candidate) setCommission(commission uint32) {
	candidate.isDirty = true
	candidate.Commission = commission
}

func (candidate *Candidate) setReward(address types.Address) {
	candidate.isDirty = true
	candidate.RewardAddress = address
}

func (candidate *Candidate) setControl(address types.Address) {
	candidate.isDirty = true
	candidate.ControlAddress = address
}

func (candidate *Candidate) setPublicKey(pubKey types.Pubkey) {
	candidate.isDirty = true
	candidate.PubKey = pubKey
	candidate.setTmAddress()
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

// getFilteredUpdates returns updates which is > 0 in their value + merge similar updates
func (candidate *Candidate) getFilteredUpdates() []*stake {
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
				u.Value = big.NewInt(0).Add(u.Value, update.Value)
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

// filterUpdates filters candidate updates: remove 0-valued updates and merge similar ones
func (candidate *Candidate) filterUpdates() {
	if len(candidate.updates) == 0 {
		return
	}

	updates := candidate.getFilteredUpdates()

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

func (candidate *Candidate) punishStake(owner types.Address, coin types.CoinID, key uint32, height uint64) bool {
	for i, stake := range candidate.stakes {
		if stake.Owner != owner || stake.Coin != coin {
			continue
		}

		var locked []*lockedValue
		for _, value := range stake.Locked {
			if value.From != key || value.ToHeight != height {
				locked = append(locked, value)
				continue
			}

			stake.Value.Sub(stake.Value, value.Value)
			stake.markDirty(i)
		}
		stake.Locked = locked
		return true
	}
	return false
}

type lockedValue struct {
	ToHeight uint64
	Value    *big.Int
	From     uint32
}

func (l *lockedValue) GetToHeight() uint64 {
	return l.ToHeight
}
func (l *lockedValue) GetValue() *big.Int {
	return big.NewInt(0).Set(l.Value)
}
func (l *lockedValue) GetFrom() uint32 {
	return l.From
}

type stake struct {
	Owner    types.Address
	Coin     types.CoinID
	Value    *big.Int
	BipValue *big.Int

	index     int
	markDirty func(int)

	Locked []*lockedValue `rlp:"tail"` // must be on last field
}

func (stake *stake) addValue(value *big.Int) {
	stake.markDirty(stake.index)
	stake.Value = big.NewInt(0).Add(stake.Value, value)
}

func (stake *stake) subValue(value *big.Int) {
	stake.markDirty(stake.index)
	stake.Value = big.NewInt(0).Sub(stake.Value, value)
}

func (stake *stake) setBipValue(value *big.Int) {
	if stake.BipValue.Cmp(value) != 0 {
		stake.markDirty(stake.index)
	}

	stake.BipValue = big.NewInt(0).Set(value)
}

func (stake *stake) setValue(ret *big.Int) {
	stake.markDirty(stake.index)
	stake.Value = big.NewInt(0).Set(ret)
}
