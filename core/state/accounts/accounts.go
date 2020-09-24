package accounts

import (
	"bytes"
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

type RAccounts interface {
	Export(state *types.AppState)
	GetAccount(address types.Address) *Model
	GetNonce(address types.Address) uint64
	GetBalance(address types.Address, coin types.CoinID) *big.Int
	GetBalances(address types.Address) []Balance
	ExistsMultisig(msigAddress types.Address) bool
}

type Accounts struct {
	list  map[types.Address]*Model
	dirty map[types.Address]struct{}

	iavl tree.MTree
	bus  *bus.Bus

	lock sync.RWMutex
}

type Balance struct {
	Coin  bus.Coin
	Value *big.Int
}

func NewAccounts(stateBus *bus.Bus, iavl tree.MTree) (*Accounts, error) {
	accounts := &Accounts{iavl: iavl, bus: stateBus, list: map[types.Address]*Model{}, dirty: map[types.Address]struct{}{}}
	accounts.bus.SetAccounts(NewBus(accounts))

	return accounts, nil
}

func (a *Accounts) Commit() error {
	accounts := a.getOrderedDirtyAccounts()
	for _, address := range accounts {
		account := a.getFromMap(address)
		a.lock.Lock()
		delete(a.dirty, address)
		a.lock.Unlock()

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
			coinsList, err := rlp.EncodeToBytes(account.coins)
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
				path = append(path, coin.Bytes()...)

				balance := account.getBalance(coin)
				if balance.Cmp(big.NewInt(0)) == 0 {
					a.iavl.Remove(path)
				} else {
					a.iavl.Set(path, balance.Bytes())
				}
			}

			account.dirtyBalances = map[types.CoinID]struct{}{}
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

func (a *Accounts) AddBalance(address types.Address, coin types.CoinID, amount *big.Int) {
	balance := a.GetBalance(address, coin)
	a.SetBalance(address, coin, big.NewInt(0).Add(balance, amount))
}

func (a *Accounts) GetBalance(address types.Address, coin types.CoinID) *big.Int {
	account := a.getOrNew(address)
	if !account.hasCoin(coin) {
		return big.NewInt(0)
	}

	if _, ok := account.balances[coin]; !ok {
		balance := big.NewInt(0)

		path := []byte{mainPrefix}
		path = append(path, address[:]...)
		path = append(path, balancePrefix)
		path = append(path, coin.Bytes()...)

		_, enc := a.iavl.Get(path)
		if len(enc) != 0 {
			balance = big.NewInt(0).SetBytes(enc)
		}

		account.balances[coin] = balance
	}

	return big.NewInt(0).Set(account.balances[coin])
}

func (a *Accounts) SubBalance(address types.Address, coin types.CoinID, amount *big.Int) {
	balance := big.NewInt(0).Sub(a.GetBalance(address, coin), amount)
	a.SetBalance(address, coin, balance)
}

func (a *Accounts) SetBalance(address types.Address, coin types.CoinID, amount *big.Int) {
	account := a.getOrNew(address)
	oldBalance := a.GetBalance(address, coin)
	a.bus.Checker().AddCoin(coin, big.NewInt(0).Sub(amount, oldBalance))

	account.setBalance(coin, amount)
}

func (a *Accounts) SetNonce(address types.Address, nonce uint64) {
	account := a.getOrNew(address)
	account.setNonce(nonce)
}

func (a *Accounts) ExistsMultisig(msigAddress types.Address) bool {
	acc := a.get(msigAddress)
	if acc == nil {
		return false
	}

	if acc.IsMultisig() {
		return true
	}

	if acc.Nonce > 0 {
		return true
	}

	return false
}

func (a *Accounts) CreateMultisig(weights []uint32, addresses []types.Address, threshold uint32, height uint64, address types.Address) types.Address {
	msig := Multisig{
		Weights:   weights,
		Threshold: threshold,
		Addresses: addresses,
	}

	account := a.get(address)

	if account == nil {
		account = &Model{
			Nonce:         0,
			address:       address,
			coins:         []types.CoinID{},
			balances:      map[types.CoinID]*big.Int{},
			markDirty:     a.markDirty,
			isNew:         true,
			dirtyBalances: map[types.CoinID]struct{}{},
		}
	}

	account.MultisigData = msig
	account.markDirty(account.address)
	account.isDirty = true
	a.setToMap(address, account)

	return address
}

func (a *Accounts) EditMultisig(threshold uint32, weights []uint32, addresses []types.Address, address types.Address) types.Address {
	account := a.get(address)

	msig := Multisig{
		Threshold: threshold,
		Weights:   weights,
		Addresses: addresses,
	}

	account.MultisigData = msig
	account.markDirty(account.address)
	account.isDirty = true
	a.setToMap(address, account)

	return address
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
	account.balances = map[types.CoinID]*big.Int{}
	account.markDirty = a.markDirty
	account.dirtyBalances = map[types.CoinID]struct{}{}

	// load coins
	path = []byte{mainPrefix}
	path = append(path, address[:]...)
	path = append(path, coinsPrefix)
	_, enc = a.iavl.Get(path)
	if len(enc) != 0 {
		var coins []types.CoinID
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
			coins:         []types.CoinID{},
			balances:      map[types.CoinID]*big.Int{},
			markDirty:     a.markDirty,
			dirtyBalances: map[types.CoinID]struct{}{},
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

func (a *Accounts) GetBalances(address types.Address) []Balance {
	account := a.getOrNew(address)

	balances := make([]Balance, len(account.coins))
	for key, id := range account.coins {
		balances[key] = Balance{
			Coin:  *a.bus.Coins().GetCoin(id),
			Value: a.GetBalance(address, id),
		}
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
			for _, b := range a.GetBalances(account.address) {
				balance = append(balance, types.Balance{
					Coin:  uint64(b.Coin.ID),
					Value: b.Value.String(),
				})
			}

			// sort balances by coin symbol
			sort.SliceStable(balance, func(i, j int) bool {
				return bytes.Compare(types.CoinID(balance[i].Coin).Bytes(), types.CoinID(balance[j].Coin).Bytes()) == 1
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
