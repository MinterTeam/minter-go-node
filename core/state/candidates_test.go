package state

import (
	"crypto/rand"
	"encoding/binary"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

const height = 500000

func TestSimpleDelegate(t *testing.T) {
	st := getState()

	address := types.Address{}
	coin := types.GetBaseCoinID()
	amount := big.NewInt(1)
	pubkey := createTestCandidate(st)

	st.Candidates.Delegate(address, pubkey, coin, amount, big.NewInt(0))
	st.Candidates.RecalculateStakes(height)

	stake := st.Candidates.GetStakeOfAddress(pubkey, address, coin)
	if stake == nil {
		t.Fatalf("Stake of address %s not found", address.String())
	}

	if stake.Value.Cmp(amount) != 0 {
		t.Errorf("Stake of address %s should be %s, got %s", address.String(), amount.String(), stake.Value.String())
	}

	if stake.BipValue.Cmp(amount) != 0 {
		t.Errorf("Bip value of stake of address %s should be %s, got %s", address.String(), amount.String(), stake.BipValue.String())
	}
}

func TestDelegate(t *testing.T) {
	st := getState()

	address := types.Address{}
	coin := types.GetBaseCoinID()
	amount := big.NewInt(1)
	totalAmount := big.NewInt(0)
	pubkey := createTestCandidate(st)

	for i := 0; i < 10000; i++ {
		st.Candidates.Delegate(address, pubkey, coin, amount, big.NewInt(0))
		totalAmount.Add(totalAmount, amount)
	}

	st.Candidates.RecalculateStakes(height)

	stake := st.Candidates.GetStakeOfAddress(pubkey, address, coin)
	if stake == nil {
		t.Fatalf("Stake of address %s not found", address.String())
	}

	if stake.Value.Cmp(totalAmount) != 0 {
		t.Errorf("Stake of address %s should be %s, got %s", address.String(), amount.String(), stake.Value.String())
	}

	if stake.BipValue.Cmp(totalAmount) != 0 {
		t.Errorf("Bip value of stake of address %s should be %s, got %s", address.String(), amount.String(), stake.BipValue.String())
	}
}

func TestComplexDelegate(t *testing.T) {
	st := getState()

	coin := types.GetBaseCoinID()
	pubkey := createTestCandidate(st)

	for i := uint64(0); i < 2000; i++ {
		amount := big.NewInt(int64(2000 - i))
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)
		st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))
	}

	st.Candidates.RecalculateStakes(height)

	for i := uint64(0); i < 1000; i++ {
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)
		amount := big.NewInt(int64(2000 - i))

		stake := st.Candidates.GetStakeOfAddress(pubkey, addr, coin)
		if stake == nil {
			t.Fatalf("Stake of address %s not found", addr.String())
		}

		if stake.Value.Cmp(amount) != 0 {
			t.Errorf("Stake of address %s should be %s, got %s", addr.String(), amount.String(), stake.Value.String())
		}

		if stake.BipValue.Cmp(amount) != 0 {
			t.Errorf("Bip value of stake of address %s should be %s, got %s", addr.String(), amount.String(), stake.BipValue.String())
		}
	}

	for i := uint64(1000); i < 2000; i++ {
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)

		stake := st.Candidates.GetStakeOfAddress(pubkey, addr, coin)
		if stake != nil {
			t.Fatalf("Stake of address %s found, but should not be", addr.String())
		}
	}

	{
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 3000)
		st.Candidates.Delegate(addr, pubkey, coin, big.NewInt(3000), big.NewInt(0))

		st.Candidates.RecalculateStakes(height)

		replacedAddress := types.HexToAddress("Mx00000000000003e7000000000000000000000000")
		stake := st.Candidates.GetStakeOfAddress(pubkey, replacedAddress, coin)
		if stake != nil {
			t.Fatalf("Stake of address %s found, but should not be", replacedAddress.String())
		}

		stake = st.Candidates.GetStakeOfAddress(pubkey, addr, coin)
		if stake == nil {
			t.Fatalf("Stake of address %s not found, but should be", addr.String())
		}
	}

	{
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 4000)
		st.Candidates.Delegate(addr, pubkey, coin, big.NewInt(4000), big.NewInt(0))

		var addr2 types.Address
		binary.BigEndian.PutUint64(addr2[:], 3500)
		st.Candidates.Delegate(addr2, pubkey, coin, big.NewInt(3500), big.NewInt(0))

		st.Candidates.RecalculateStakes(height)

		stake := st.Candidates.GetStakeOfAddress(pubkey, addr, coin)
		if stake == nil {
			t.Fatalf("Stake of address %s not found, but should be", addr.String())
		}

		replacedAddress := types.HexToAddress("Mx00000000000003e5000000000000000000000000")
		stake = st.Candidates.GetStakeOfAddress(pubkey, replacedAddress, coin)
		if stake != nil {
			t.Fatalf("Stake of address %s found, but should not be", replacedAddress.String())
		}
	}

	{
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 4001)
		st.Candidates.Delegate(addr, pubkey, coin, big.NewInt(900), big.NewInt(0))

		st.Candidates.RecalculateStakes(height)

		stake := st.Candidates.GetStakeOfAddress(pubkey, addr, coin)
		if stake != nil {
			t.Fatalf("Stake of address %s found, but should not be", addr.String())
		}
	}
}

func TestStakeSufficiency(t *testing.T) {
	st := getState()

	coin := types.GetBaseCoinID()
	pubkey := createTestCandidate(st)

	for i := uint64(0); i < 1000; i++ {
		amount := big.NewInt(int64(1000 - i))
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)
		st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))
	}

	st.Candidates.RecalculateStakes(height)

	{
		stake := big.NewInt(1)
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 1001)

		result := st.Candidates.IsDelegatorStakeSufficient(addr, pubkey, coin, stake)
		if result {
			t.Fatalf("Stake %s %s of address %s shold not be sufficient", stake.String(), coin.String(), addr.String())
		}
	}

	{
		stake := big.NewInt(1)
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 1)

		result := st.Candidates.IsDelegatorStakeSufficient(addr, pubkey, coin, stake)
		if !result {
			t.Fatalf("Stake of %s %s of address %s shold be sufficient", stake.String(), coin.String(), addr.String())
		}
	}

	{
		stake := big.NewInt(1001)
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 1002)

		result := st.Candidates.IsDelegatorStakeSufficient(addr, pubkey, coin, stake)
		if !result {
			t.Fatalf("Stake of %s %s of address %s shold be sufficient", stake.String(), coin.String(), addr.String())
		}
	}
}

func TestDoubleSignPenalty(t *testing.T) {
	st := getState()

	pubkey := createTestCandidate(st)

	coin := types.GetBaseCoinID()
	amount := big.NewInt(100)
	var addr types.Address
	binary.BigEndian.PutUint64(addr[:], 1)
	st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))

	st.Candidates.RecalculateStakes(height)

	var pk ed25519.PubKeyEd25519
	copy(pk[:], pubkey[:])

	var tmAddr types.TmAddress
	copy(tmAddr[:], pk.Address().Bytes())

	st.Candidates.PunishByzantineCandidate(1, tmAddr)

	stake := st.Candidates.GetStakeValueOfAddress(pubkey, addr, coin)
	if stake.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Stake is not correct. Expected 0, got %s", stake.String())
	}

	ffs := st.FrozenFunds.GetFrozenFunds(1 + candidates.UnbondPeriod)
	exists := false
	for _, ff := range ffs.List {
		if ff.Address == addr {
			exists = true

			newValue := big.NewInt(0).Set(amount)
			newValue.Mul(newValue, big.NewInt(95))
			newValue.Div(newValue, big.NewInt(100))
			newValue.Sub(newValue, ff.Value)
			if newValue.Cmp(big.NewInt(0)) != 0 {
				t.Fatalf("Wrong frozen fund value. Expected %s, got %s", newValue.String(), ff.Value.String())
			}
		}
	}

	if !exists {
		t.Fatalf("Frozen fund not found")
	}
}

func TestAbsentPenalty(t *testing.T) {
	st := getState()

	pubkey := createTestCandidate(st)

	coin := types.GetBaseCoinID()
	amount := big.NewInt(100)
	var addr types.Address
	binary.BigEndian.PutUint64(addr[:], 1)
	st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))

	st.Candidates.RecalculateStakes(height)

	var pk ed25519.PubKeyEd25519
	copy(pk[:], pubkey[:])

	var tmAddr types.TmAddress
	copy(tmAddr[:], pk.Address().Bytes())

	st.Candidates.Punish(1, tmAddr)

	stake := st.Candidates.GetStakeValueOfAddress(pubkey, addr, coin)
	newValue := big.NewInt(0).Set(amount)
	newValue.Mul(newValue, big.NewInt(99))
	newValue.Div(newValue, big.NewInt(100))
	if stake.Cmp(newValue) != 0 {
		t.Fatalf("Stake is not correct. Expected %s, got %s", newValue, stake.String())
	}
}

func TestDoubleAbsentPenalty(t *testing.T) {
	st := getState()

	pubkey := createTestCandidate(st)

	coin := types.GetBaseCoinID()
	amount := big.NewInt(1000)
	var addr types.Address
	binary.BigEndian.PutUint64(addr[:], 1)
	st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))
	st.Candidates.SetOnline(pubkey)

	st.Candidates.RecalculateStakes(height)

	var pk ed25519.PubKeyEd25519
	copy(pk[:], pubkey[:])

	var tmAddr types.TmAddress
	copy(tmAddr[:], pk.Address().Bytes())

	st.Validators.SetNewValidators(st.Candidates.GetNewCandidates(1))

	for i := 1000; i < 1050; i++ {
		st.Validators.SetValidatorAbsent(uint64(i), tmAddr)
		st.Validators.SetNewValidators(st.Candidates.GetNewCandidates(1))
	}

	stake := st.Candidates.GetStakeValueOfAddress(pubkey, addr, coin)
	newValue := big.NewInt(0).Set(amount)
	newValue.Mul(newValue, big.NewInt(99))
	newValue.Div(newValue, big.NewInt(100))
	if stake.Cmp(newValue) != 0 {
		t.Fatalf("Stake is not correct. Expected %s, got %s", newValue, stake.String())
	}
}

func TestZeroStakePenalty(t *testing.T) {
	st := getState()

	pubkey := createTestCandidate(st)

	coin := types.GetBaseCoinID()
	amount := big.NewInt(10000)
	var addr types.Address
	binary.BigEndian.PutUint64(addr[:], 1)
	st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))

	st.Candidates.RecalculateStakes(height)

	st.Candidates.SubStake(addr, pubkey, coin, amount)
	st.FrozenFunds.AddFund(518400, addr, pubkey, coin, amount)

	var pk ed25519.PubKeyEd25519
	copy(pk[:], pubkey[:])

	var tmAddr types.TmAddress
	copy(tmAddr[:], pk.Address().Bytes())

	st.Candidates.Punish(1, tmAddr)

	stake := st.Candidates.GetStakeValueOfAddress(pubkey, addr, coin)
	newValue := big.NewInt(0)

	if stake.Cmp(newValue) != 0 {
		t.Fatalf("Stake is not correct. Expected %s, got %s", newValue, stake.String())
	}
}

func TestDelegationAfterUnbond(t *testing.T) {
	st := getState()

	coin := types.GetBaseCoinID()
	pubkey := createTestCandidate(st)

	for i := uint64(0); i < 1000; i++ {
		amount := big.NewInt(int64(1000 - i))
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)
		st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))
	}

	st.Candidates.RecalculateStakes(height)

	// unbond
	{
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 2)
		amount := big.NewInt(int64(1000 - 2))

		st.Candidates.SubStake(addr, pubkey, coin, amount)
		st.Candidates.RecalculateStakes(height)
		st.Candidates.Commit()
	}

	// delegate
	{
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], 2000)
		amount := big.NewInt(2000)

		st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))
		st.Candidates.RecalculateStakes(height)

		value := st.Candidates.GetStakeValueOfAddress(pubkey, addr, coin)
		if value == nil || value.Cmp(amount) != 0 {
			t.Fatalf("Stake of address %s is not correct", addr.String())
		}
	}

	for i := uint64(0); i < 1000; i++ {
		if i == 2 {
			continue
		}

		amount := big.NewInt(int64(1000 - i))
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)
		value := st.Candidates.GetStakeValueOfAddress(pubkey, addr, coin)
		if value == nil || value.Cmp(amount) != 0 {
			t.Fatalf("Stake of address %s is not correct", addr.String())
		}
	}

}

func TestDelegationWithWatchList(t *testing.T) {
	st := getState()

	coin := types.GetBaseCoinID()
	pubkey := createTestCandidate(st)
	stakeAmount := big.NewInt(10000)
	watchlistAmount := big.NewInt(1000)
	amount := new(big.Int).Add(stakeAmount, watchlistAmount)

	var addr types.Address
	binary.BigEndian.PutUint64(addr[:], 1)

	st.Watchlist.AddWatchList(addr, pubkey, coin, watchlistAmount)

	if watchList := st.Watchlist.Get(addr, pubkey, coin); watchList != nil {
		stakeAmount.Add(stakeAmount, watchList.Value)
		st.Watchlist.Delete(addr, pubkey, coin)
	}

	st.Candidates.Delegate(addr, pubkey, coin, stakeAmount, stakeAmount)
	st.Candidates.RecalculateStakes(height)

	value := st.Candidates.GetStakeValueOfAddress(pubkey, addr, coin)
	if value == nil || value.Cmp(amount) != 0 {
		t.Fatalf("Stake of address %s is not correct. Expected %s, got %s", addr.String(), amount, value)
	}

	wl := st.Watchlist.Get(addr, pubkey, coin)
	if wl != nil {
		t.Fatalf("Watchlist is not deleted")
	}
}

func getState() *State {
	s, err := NewState(0, db.NewMemDB(), emptyEvents{}, 1, 1)

	if err != nil {
		panic(err)
	}

	return s
}

func createTestCandidate(stateDB *State) types.Pubkey {
	address := types.Address{}
	pubkey := types.Pubkey{}
	_, _ = rand.Read(pubkey[:])

	stateDB.Candidates.Create(address, address, address, pubkey, 10)

	return pubkey
}

type emptyEvents struct{}

func (e emptyEvents) AddEvent(height uint32, event eventsdb.Event) {}
func (e emptyEvents) LoadEvents(height uint32) eventsdb.Events     { return eventsdb.Events{} }
func (e emptyEvents) CommitEvents() error                          { return nil }
