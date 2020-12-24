package waitlist

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

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

type Item struct {
	CandidateId uint32
	Coin        types.CoinID
	Value       *big.Int
	Locked      []bus.WaitlistItemLock `rlp:"tail"` // must be on last field
}

func (i *Item) GetAll() *big.Int {
	return big.NewInt(0).Set(i.Value)
}
func (i *Item) GetLocked() []bus.WaitlistItemLock {
	return i.Locked
}
func (i *Item) GetFree(height uint64) *big.Int {
	value := new(big.Int).Set(i.Value)
	for _, locked := range i.Locked {
		if locked.GetToHeight() > height {
			value.Sub(value, locked.GetValue())
		}
	}
	return value
}

type Model struct {
	List []*Item

	address   types.Address
	markDirty func(address types.Address)
}

func (m *Model) AddToList(candidateId uint32, coin types.CoinID, value *big.Int, l *lockedValue) {
	var lock []bus.WaitlistItemLock
	if l != nil {
		lock = append(lock, l)
	}
	for _, item := range m.List {
		if item.Coin == coin && item.CandidateId == candidateId {
			item.Value = big.NewInt(0).Add(item.Value, value)
			item.Locked = append(item.Locked, lock...)
			return
		}
	}
	m.List = append(m.List, &Item{
		CandidateId: candidateId,
		Coin:        coin,
		Value:       new(big.Int).Set(value),
		Locked:      lock,
	})
}
