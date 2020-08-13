package waitlist

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Item struct {
	PublicKey types.Pubkey
	Coin      types.CoinID
	Value     *big.Int
}

type Model struct {
	List []Item

	address   types.Address
	markDirty func(address types.Address)
}

func (m *Model) AddToList(publicKey types.Pubkey, coin types.CoinID, value *big.Int) {
	m.List = append(m.List, Item{
		PublicKey: publicKey,
		Coin:      coin,
		Value:     value,
	})
}

func (m *Model) GetValue(pubkey types.Pubkey, coin types.CoinID) *big.Int {
	value := big.NewInt(0)

	for _, item := range m.List {
		if item.PublicKey == pubkey && item.Coin == coin {
			value.Add(value, item.Value)
		}
	}

	return value
}
