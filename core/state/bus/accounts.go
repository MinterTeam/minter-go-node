package bus

import "github.com/MinterTeam/minter-go-node/core/types"

type Accounts interface {
	DeleteCoin(types.Address, types.CoinSymbol)
}
