package accounts

import (
	"bytes"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"sort"
)

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
	// todo: clear 0 balances

	if !account.HasCoin(coin) {
		account.hasDirtyCoins = true
		account.coins = append(account.coins, coin)
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
