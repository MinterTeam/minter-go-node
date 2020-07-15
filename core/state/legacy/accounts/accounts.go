package accounts

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
	"sync"
)

const mainPrefix = byte('a')
const coinsPrefix = byte('c')
const balancePrefix = byte('b')

type Accounts struct {
	list  map[types.Address]*Model
	dirty map[types.Address]struct{}

	iavl tree.MTree
	bus  *bus.Bus

	lock sync.RWMutex
}

func NewAccounts(stateBus *bus.Bus, iavl tree.MTree) (*Accounts, error) {
	accounts := &Accounts{iavl: iavl, bus: stateBus, list: map[types.Address]*Model{}, dirty: map[types.Address]struct{}{}}
	return accounts, nil
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

	return big.NewInt(0).Set(account.balances[coin])
}

func (a *Accounts) get(address types.Address) *Model {
	if account := a.getFromMap(address); account != nil {
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

	a.setToMap(address, account)
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
		a.setToMap(address, account)
	}

	return account
}

func (a *Accounts) GetNonce(address types.Address) uint64 {
	account := a.getOrNew(address)

	return account.Nonce
}

func (a *Accounts) GetBalances(address types.Address) map[types.CoinSymbol]*big.Int {
	account := a.getOrNew(address)

	balances := make(map[types.CoinSymbol]*big.Int, len(account.coins))
	for _, coin := range account.coins {
		balances[coin] = a.GetBalance(address, coin)
	}

	return balances
}

func (a *Accounts) markDirty(addr types.Address) {
	a.dirty[addr] = struct{}{}
}

func (a *Accounts) Export(state *types.AppState) {
	// todo: iterate range?
	a.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == mainPrefix {
			addressPath := key[1:]
			if len(addressPath) > types.AddressLength {
				return false
			}

			address := types.BytesToAddress(addressPath)
			account := a.get(address)

			var balance []types.Balance
			for coin, value := range a.GetBalances(account.address) {
				id := types.GetBaseCoinID()
				for key, c := range state.Coins {
					if c.Symbol == coin {
						id = c.ID

						// save owner address for coin
						// accounts are not available during coins export
						if c.Volume == value.String() {
							state.Coins[key].OwnerAddress = &account.address
						}
					}
				}

				balance = append(balance, types.Balance{
					Coin:  id,
					Value: value.String(),
				})
			}

			// sort balances by coin symbol
			sort.SliceStable(balance, func(i, j int) bool {
				return balance[i].Coin > balance[j].Coin
			})

			acc := types.Account{
				Address: account.address,
				Balance: balance,
				Nonce:   account.Nonce,
			}

			if account.IsMultisig() {
				acc.MultisigData = &types.Multisig{
					Weights:   account.MultisigData.Weights,
					Threshold: account.MultisigData.Threshold,
					Addresses: account.MultisigData.Addresses,
				}
			}

			state.Accounts = append(state.Accounts, acc)
		}

		return false
	})
}

func (a *Accounts) GetAccount(address types.Address) *Model {
	return a.getOrNew(address)
}

func (a *Accounts) getFromMap(address types.Address) *Model {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.list[address]
}

func (a *Accounts) setToMap(address types.Address, model *Model) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.list[address] = model
}
