package waitlist

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type lockedValue struct {
	ToHeight uint64
	Value    *big.Int
}

type Item struct {
	CandidateId uint32
	Coin        types.CoinID
	Value       *big.Int
	Locked      []*lockedValue `rlp:"tail"` // must be on last field
}

func (i *Item) GetAll() *big.Int {
	return big.NewInt(0).Set(i.Value)
}
func (i *Item) GetFree(height uint64) *big.Int {
	value := new(big.Int).Set(i.Value)
	for _, locked := range i.Locked {
		if locked.ToHeight > height {
			value.Sub(value, locked.Value)
		}
	}
	return value
}

type Model struct {
	List []*Item

	address   types.Address
	markDirty func(address types.Address)
}

func (m *Model) AddToList(candidateId uint32, coin types.CoinID, value *big.Int, height uint64) {
	var lock []*lockedValue
	if height != 0 {
		lock = append(lock, &lockedValue{ToHeight: height, Value: value})
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
