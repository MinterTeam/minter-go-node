package state

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/core/types"
	compact_db "github.com/klim0v/compact-db"
	db "github.com/tendermint/tm-db"
	"github.com/xujiajun/nutsdb"
	"math/big"
	"os"
	"testing"
)

func TestSimpleDelegate(t *testing.T) {
	st := getState()

	address := types.Address{}
	coin := types.GetBaseCoin()
	amount := big.NewInt(1)
	pubkey := createTestCandidate(st)

	st.Candidates.Delegate(address, pubkey, coin, amount, big.NewInt(0))
	st.Candidates.RecalculateStakes(1)

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
	coin := types.GetBaseCoin()
	amount := big.NewInt(1)
	totalAmount := big.NewInt(0)
	pubkey := createTestCandidate(st)

	for i := 0; i < 10000; i++ {
		st.Candidates.Delegate(address, pubkey, coin, amount, big.NewInt(0))
		totalAmount.Add(totalAmount, amount)
	}

	st.Candidates.RecalculateStakes(1)

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

func TestDelegateALot(t *testing.T) {
	st := getState()

	coin := types.GetBaseCoin()
	amount := big.NewInt(1)
	pubkey := createTestCandidate(st)

	for i := uint64(0); i < 2000; i++ {
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)
		st.Candidates.Delegate(addr, pubkey, coin, amount, big.NewInt(0))
	}

	st.Candidates.RecalculateStakes(1)

	for i := uint64(0); i < 1000; i++ {
		var addr types.Address
		binary.BigEndian.PutUint64(addr[:], i)

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
}

func getState() *State {
	opt := nutsdb.DefaultOptions
	opt.Dir = "/tmp/nutsdb"
	_ = os.RemoveAll(opt.Dir)
	nuts, err := nutsdb.Open(opt)

	s, err := NewState(0, db.NewMemDB(), nuts, emptyEvents{}, 1, 1)

	if err != nil {
		panic(err)
	}

	return s
}

func createTestCandidate(stateDB *State) types.Pubkey {
	address := types.Address{}
	pubkey := types.Pubkey{}
	_, _ = rand.Read(pubkey[:])

	stateDB.Candidates.Create(address, address, pubkey, 10)

	return pubkey
}

type emptyEvents struct{}

func (e emptyEvents) AddEvent(height uint32, event compact_db.Event) {}
func (e emptyEvents) LoadEvents(height uint32) compact_db.Events     { return compact_db.Events{} }
func (e emptyEvents) CommitEvents() error                            { return nil }
