package accounts

import "github.com/MinterTeam/minter-go-node/core/types"

type Bus struct {
	accounts *Accounts
}

func (b *Bus) DeleteCoin(address types.Address, coin types.CoinSymbol) {
	b.accounts.DeleteCoin(address, coin)
}

func NewBus(accounts *Accounts) *Bus {
	return &Bus{accounts: accounts}
}
