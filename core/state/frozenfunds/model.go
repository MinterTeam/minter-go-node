package frozenfunds

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
)

type Item struct {
	Address      types.Address
	CandidateKey *types.Pubkey
	Coin         types.CoinSymbol
	Value        *big.Int
}

type Model struct {
	List []Item

	height    uint64
	deleted   bool
	markDirty func(height uint64)
}

func (m *Model) delete() {
	m.deleted = true
	m.markDirty(m.height)
}

func (m *Model) addFund(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	m.List = append(m.List, Item{
		Address:      address,
		CandidateKey: &pubkey,
		Coin:         coin,
		Value:        value,
	})
	m.markDirty(m.height)
}

func (m *Model) Height() uint64 {
	return m.height
}

func (m *Model) deleteCoin(coinSymbol types.CoinSymbol, bus *bus.Bus) {
	coin := bus.Coins().GetCoin(coinSymbol)
	var list []Item

	for _, ff := range m.List {
		if ff.Coin == coinSymbol {
			ret := formula.CalculateSaleReturn(coin.Volume, coin.Reserve, 100, ff.Value)

			list = append(list, Item{
				Address:      ff.Address,
				CandidateKey: ff.CandidateKey,
				Coin:         types.GetBaseCoin(),
				Value:        ret,
			})

			continue
		}

		list = append(list, ff)
	}

	m.List = list
	m.markDirty(m.height)
}
