package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Coins interface {
	GetCoin(types.CoinSymbol) *Coin
	SubCoinVolume(types.CoinSymbol, *big.Int)
	SubCoinReserve(types.CoinSymbol, *big.Int)
	SanitizeCoin(types.CoinSymbol)

	AddOwnerAddress(types.CoinSymbol, types.Address)
	RemoveOwnerAddress(types.CoinSymbol, types.Address)

	AddOwnerCandidate(types.CoinSymbol, types.Pubkey)
	RemoveOwnerCandidate(types.CoinSymbol, types.Pubkey)

	AddOwnerFrozenFund(types.CoinSymbol, uint64)
	RemoveOwnerFrozenFund(types.CoinSymbol, uint64)
}

type Coin struct {
	Name    string
	Crr     uint
	Symbol  types.CoinSymbol
	Volume  *big.Int
	Reserve *big.Int
}
