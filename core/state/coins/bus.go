package coins

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Bus struct {
	coins *Coins
}

func NewBus(coins *Coins) *Bus {
	return &Bus{coins: coins}
}

func (b *Bus) GetCoin(id types.CoinID) *bus.Coin {
	coin := b.coins.GetCoin(id)
	if coin == nil {
		return nil
	}

	return &bus.Coin{
		ID:           coin.id,
		Name:         coin.Name(),
		Crr:          coin.Crr(),
		Symbol:       coin.Symbol(),
		Volume:       coin.Volume(),
		Reserve:      coin.Reserve(),
		Version:      coin.Version(),
		OwnerAddress: coin.OwnerAddress(),
	}
}

func (b *Bus) SubCoinVolume(id types.CoinID, amount *big.Int) {
	b.coins.SubVolume(id, amount)
}

func (b *Bus) SubCoinReserve(id types.CoinID, amount *big.Int) {
	b.coins.SubReserve(id, amount)
}
