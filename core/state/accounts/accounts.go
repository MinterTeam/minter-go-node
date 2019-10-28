package accounts

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
)

const mainPrefix = byte('a')
const coinsPrefix = byte('c')
const balancePrefix = byte('b')

type Accounts struct {
	list  map[types.Address]*Model
	dirty map[types.Address]struct{}

	iavl tree.Tree
	bus  *bus.Bus
}

func NewAccounts(stateBus *bus.Bus, iavl tree.Tree) (*Accounts, error) {
	accounts := &Accounts{iavl: iavl, bus: stateBus, list: map[types.Address]*Model{}, dirty: map[types.Address]struct{}{}}
	accounts.bus.SetAccounts(NewBus(accounts))

	return accounts, nil
}

func (a *Accounts) Commit() error {
	accounts := a.getOrderedDirtyAccounts()
	for _, address := range accounts {
		account := a.list[address]
		delete(a.dirty, address)

		// save info (nonce and multisig data)
		if account.isDirty || account.isNew {
			data, err := rlp.EncodeToBytes(account)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, address[:]...)
			a.iavl.Set(path, data)
			account.isDirty = false
			account.isNew = false
		}

		// save coins list
		if account.hasDirtyCoins {
			coins := account.getOrderedCoins()
			coinsList, err := rlp.EncodeToBytes(coins)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, address[:]...)
			path = append(path, coinsPrefix)
			a.iavl.Set(path, coinsList)
			account.hasDirtyCoins = false
		}

		// save balances
		if account.hasDirtyBalances() {
			coins := account.getOrderedCoins()
			for _, coin := range coins {
				if !account.isBalanceDirty(coin) {
					continue
				}

				path := []byte{mainPrefix}
				path = append(path, address[:]...)
				path = append(path, balancePrefix)
				path = append(path, coin[:]...)

				balance := account.getBalance(coin)
				if balance.Cmp(big.NewInt(0)) == 0 {
					a.iavl.Remove(path)
				} else {
					a.iavl.Set(path, balance.Bytes())
				}
			}

			account.dirtyBalances = map[types.CoinSymbol]struct{}{}
		}
	}

	return nil
}

func (a *Accounts) getOrderedDirtyAccounts() []types.Address {
	keys := make([]types.Address, 0, len(a.dirty))
	for k := range a.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (a *Accounts) AddBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	balance := a.GetBalance(address, coin)
	if balance.Cmp(big.NewInt(0)) == 0 {
		a.bus.Coins().AddOwnerAddress(coin, address)
	}
	a.SetBalance(address, coin, big.NewInt(0).Add(balance, amount))
}

func (a *Accounts) GetBalance(address types.Address, coin types.CoinSymbol) *big.Int {
	account := a.getOrNew(address)
	if !account.hasCoin(coin) {
		return big.NewInt(0)
	}

	if _, ok := account.balances[coin]; !ok {
		balance := big.NewInt(0)

		path := []byte{mainPrefix}
		path = append(path, address[:]...)
		path = append(path, balancePrefix)
		path = append(path, coin[:]...)

		_, enc := a.iavl.Get(path)
		if len(enc) != 0 {
			balance = big.NewInt(0).SetBytes(enc)
		}

		account.balances[coin] = balance
	}

	return account.balances[coin]
}

func (a *Accounts) SubBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	balance := big.NewInt(0).Sub(a.GetBalance(address, coin), amount)
	if balance.Cmp(big.NewInt(0)) == 0 {
		a.bus.Coins().RemoveOwnerAddress(coin, address)
	}
	a.SetBalance(address, coin, balance)
}

func (a *Accounts) SetBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	if amount.Cmp(big.NewInt(0)) == 0 {
		a.bus.Coins().RemoveOwnerAddress(coin, address)
	}
	account := a.getOrNew(address)
	if a.GetBalance(address, coin).Cmp(big.NewInt(0)) == 0 {
		a.bus.Coins().AddOwnerAddress(coin, address)
	}
	account.setBalance(coin, amount)
}

func (a *Accounts) SetNonce(address types.Address, nonce uint64) {
	account := a.getOrNew(address)
	account.setNonce(nonce)
}

func (a *Accounts) Exists(msigAddress types.Address) bool {
	panic("implement me")
}

func (a *Accounts) CreateMultisig(weights []uint, addresses []types.Address, threshold uint) types.Address {
	panic("implement me")
}

func (a *Accounts) get(address types.Address) *Model {
	if account := a.list[address]; account != nil {
		return account
	}

	path := []byte{mainPrefix}
	path = append(path, address[:]...)
	_, enc := a.iavl.Get(path)
	if len(enc) == 0 {
		return nil
	}

	account := &Model{}
	if err := rlp.DecodeBytes(enc, account); err != nil {
		panic(fmt.Sprintf("failed to decode account at address %s: %s", address.String(), err))
		return nil
	}

	account.address = address
	account.balances = map[types.CoinSymbol]*big.Int{}
	account.markDirty = a.markDirty
	account.dirtyBalances = map[types.CoinSymbol]struct{}{}

	// load coins
	path = []byte{mainPrefix}
	path = append(path, address[:]...)
	path = append(path, coinsPrefix)
	_, enc = a.iavl.Get(path)
	if len(enc) != 0 {
		var coins []types.CoinSymbol
		if err := rlp.DecodeBytes(enc, &coins); err != nil {
			panic(fmt.Sprintf("failed to decode coins list at address %s: %s", address.String(), err))
		}

		account.coins = coins
	}

	a.list[address] = account
	return account
}

func (a *Accounts) getOrNew(address types.Address) *Model {
	account := a.get(address)
	if account == nil {
		account = &Model{
			Nonce:         0,
			address:       address,
			coins:         []types.CoinSymbol{},
			balances:      map[types.CoinSymbol]*big.Int{},
			markDirty:     a.markDirty,
			dirtyBalances: map[types.CoinSymbol]struct{}{},
			isNew:         true,
		}
		a.list[address] = account
	}

	return account
}

func (a *Accounts) GetNonce(address types.Address) uint64 {
	account := a.getOrNew(address)

	return account.Nonce
}

func (a *Accounts) GetBalances(address types.Address) map[types.CoinSymbol]*big.Int {
	account := a.getOrNew(address)

	balances := map[types.CoinSymbol]*big.Int{}
	for _, coin := range account.coins {
		balances[coin] = a.GetBalance(address, coin)
	}

	return balances
}

func (a *Accounts) DeleteCoin(address types.Address, symbol types.CoinSymbol) {
	balance := a.GetBalance(address, symbol)
	coin := a.bus.Coins().GetCoin(symbol)

	ret := formula.CalculateSaleReturn(coin.Volume, coin.Reserve, 100, balance)

	a.AddBalance(address, types.GetBaseCoin(), ret)
	a.SetBalance(address, symbol, big.NewInt(0))
}

func (a *Accounts) markDirty(addr types.Address) {
	a.dirty[addr] = struct{}{}
}
