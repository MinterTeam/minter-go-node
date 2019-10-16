package bus

import "github.com/MinterTeam/minter-go-node/core/types"

type FrozenFunds interface {
	DeleteCoin(types.CoinSymbol)
}
