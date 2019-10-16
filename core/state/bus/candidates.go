package bus

import "github.com/MinterTeam/minter-go-node/core/types"

type Candidates interface {
	DeleteCoin(types.CoinSymbol)
}
