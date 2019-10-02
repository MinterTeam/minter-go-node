package accounts

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
)

const mainPrefix = byte('a')
const coinsPrefix = byte('c')
const balancePrefix = byte('b')

type Accounts struct {
	list  map[types.Address]*Account
	dirty map[types.Address]struct{}

	iavl tree.Tree
}

func NewAccounts(iavl tree.Tree) (*Accounts, error) {
	return &Accounts{iavl: iavl}, nil
}

func (v *Accounts) Commit() error {
	accounts := v.getOrderedDirtyAccounts()
	for _, address := range accounts {
		account := v.list[address]

		// save info (nonce and multisig data)
		if account.isDirty {
			data, err := rlp.EncodeToBytes(account)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, address[:]...)
			v.iavl.Set(path, data)
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
			v.iavl.Set(path, coinsList)
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
				v.iavl.Set(path, account.GetBalance(coin).Bytes())
			}
		}
	}

	return nil
}

func (v *Accounts) getOrderedDirtyAccounts() []types.Address {
	keys := make([]types.Address, 0, len(v.dirty))
	for k := range v.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (v *Accounts) AddBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	balance := v.GetBalance(address, coin)
	v.SetBalance(address, coin, big.NewInt(0).Add(balance, amount))
}

func (v *Accounts) GetBalance(address types.Address, coin types.CoinSymbol) *big.Int {
	account := v.GetOrNew(address)

	if _, ok := account.balances[coin]; !ok {
		balance := big.NewInt(0)

		path := []byte{mainPrefix}
		path = append(path, address[:]...)
		path = append(path, balancePrefix)
		path = append(path, coin[:]...)

		_, enc := v.iavl.Get(path)
		if len(enc) != 0 {
			balance = big.NewInt(0).SetBytes(enc)
		}

		account.balances[coin] = balance
	}

	return account.balances[coin]
}

func (v *Accounts) SubBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	balance := v.GetBalance(address, coin)
	v.SetBalance(address, coin, big.NewInt(0).Sub(balance, amount))
}

func (v *Accounts) SetBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	account := v.GetOrNew(address)
	account.SetBalance(coin, amount)
}

func (v *Accounts) SetNonce(address types.Address, nonce uint64) {
	account := v.GetOrNew(address)
	account.SetNonce(nonce)
}

func (v *Accounts) Exists(msigAddress types.Address) bool {
	panic("implement me")
}

func (v *Accounts) CreateMultisig(weights []uint, addresses []types.Address, threshold uint) types.Address {
	panic("implement me")
}

func (v *Accounts) GetOrNew(address types.Address) *Account {
	if account := v.list[address]; account != nil {
		return account
	}

	path := []byte{mainPrefix}
	path = append(path, address[:]...)
	_, enc := v.iavl.Get(path)
	if len(enc) == 0 {
		return nil
	}

	account := &Account{}
	if err := rlp.DecodeBytes(enc, account); err != nil {
		panic(fmt.Sprintf("failed to decode account at address %s: %s", address.String(), err))
		return nil
	}

	account.balances = map[types.CoinSymbol]*big.Int{}

	// load coins
	path = []byte{mainPrefix}
	path = append(path, address[:]...)
	path = append(path, coinsPrefix)
	_, enc = v.iavl.Get(path)
	if len(enc) != 0 {
		var coins []types.CoinSymbol
		if err := rlp.DecodeBytes(enc, &coins); err != nil {
			panic(fmt.Sprintf("failed to decode coins list at address %s: %s", address.String(), err))
		}

		account.coins = coins
	}

	v.list[address] = account
	return account
}

func (v *Accounts) GetNonce(address types.Address) uint64 {
	account := v.GetOrNew(address)

	return account.Nonce
}

func (v *Accounts) GetBalances(address types.Address) map[types.CoinSymbol]*big.Int {
	account := v.GetOrNew(address)

	balances := map[types.CoinSymbol]*big.Int{}
	for _, coin := range account.coins {
		path := []byte{mainPrefix}
		path = append(path, address[:]...)
		path = append(path, balancePrefix)
		path = append(path, coin[:]...)

		_, enc := v.iavl.Get(path)
		if len(enc) == 0 {
			balances[coin] = big.NewInt(0)
		}

		balances[coin] = big.NewInt(0).SetBytes(enc)
	}

	return balances
}

func (v *Accounts) markDirty(addr types.Address) {
	v.dirty[addr] = struct{}{}
}

type Account struct {
	Nonce        uint64
	MultisigData *Multisig

	address  types.Address
	coins    []types.CoinSymbol
	balances map[types.CoinSymbol]*big.Int

	hasDirtyCoins bool
	dirtyBalances map[types.CoinSymbol]struct{}
	isDirty       bool // nonce or multisig data

	markDirty func(types.Address)
}

func (account *Account) SetNonce(nonce uint64) {
	account.Nonce = nonce
	account.markDirty(account.address)
}

func (account *Account) GetBalance(coin types.CoinSymbol) *big.Int {
	return account.balances[coin]
}

func (account *Account) hasDirtyBalances() bool {
	return len(account.dirtyBalances) > 0
}

func (account *Account) isBalanceDirty(coin types.CoinSymbol) bool {
	_, exists := account.dirtyBalances[coin]
	return exists
}

func (account *Account) getOrderedCoins() []types.CoinSymbol {
	keys := make([]types.CoinSymbol, 0, len(account.balances))
	for k := range account.balances {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (account *Account) IsMultisig() bool {
	return account.MultisigData != nil
}

func (account *Account) Multisig() *Multisig {
	return account.MultisigData
}

func (account *Account) SetBalance(coin types.CoinSymbol, amount *big.Int) {
	if !account.HasCoin(coin) {
		account.hasDirtyCoins = true
	}
	account.dirtyBalances[coin] = struct{}{}
	account.markDirty(account.address)
	account.balances[coin] = amount
}

func (account *Account) HasCoin(coin types.CoinSymbol) bool {
	for _, c := range account.coins {
		if c == coin {
			return true
		}
	}

	return false
}

type Multisig struct {
	Weights   []uint
	Threshold uint
	Addresses []types.Address
}

func (m *Multisig) Address() types.Address {
	b, err := rlp.EncodeToBytes(m)

	if err != nil {
		panic(err)
	}

	var addr types.Address
	copy(addr[:], crypto.Keccak256(b)[12:])

	return addr
}

func (m *Multisig) GetWeight(address types.Address) uint {
	panic("implement me")
}
