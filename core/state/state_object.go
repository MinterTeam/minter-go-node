package state

import (
	"io"
	"math/big"

	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"sort"
)

// stateAccount represents an Minter account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateAccount struct {
	address types.Address
	data    Account

	deleted bool
	onDirty func(addr types.Address) // Callback method to mark a state object newly dirty
}

// empty returns whether the account is considered empty.
func (s *stateAccount) empty() bool {
	return false
	//return s.data.Nonce == 0 && s.data.Balance.Sign() == 0
}

type Balances struct {
	Data map[types.CoinSymbol]*big.Int
}

type Balance struct {
	Coin   types.CoinSymbol
	Amount *big.Int
}

func (b Balances) getCoins() []types.CoinSymbol {
	keys := make([]types.CoinSymbol, 0, len(b.Data))
	for k, v := range b.Data {
		if v.Cmp(types.Big0) > 0 {
			keys = append(keys, k)
		}
	}

	sort.Slice(keys, func(a, b int) bool {
		return big.NewInt(0).SetBytes(keys[a].Bytes()).Cmp(big.NewInt(0).SetBytes(keys[b].Bytes())) == 1
	})

	return keys
}

func (b Balances) EncodeRLP(w io.Writer) error {

	var balances []Balance

	for _, k := range b.getCoins() {
		balances = append(balances, Balance{k, b.Data[k]})
	}

	return rlp.Encode(w, balances)
}

func (b *Balances) DecodeRLP(s *rlp.Stream) error {

	if b.Data == nil {
		b.Data = make(map[types.CoinSymbol]*big.Int)
	}

	var balances []Balance

	if err := s.Decode(&balances); err != nil {
		return err
	}

	for _, v := range balances {
		b.Data[v.Coin] = v.Amount
	}

	return nil
}

// Account is the Minter consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce   uint64
	Balance Balances
	Root    types.Hash // merkle root of the storage trie
}

// newObject creates a state object.
func newObject(db *StateDB, address types.Address, data Account, onDirty func(addr types.Address)) *stateAccount {
	if data.Balance.Data == nil {
		data.Balance.Data = make(map[types.CoinSymbol]*big.Int)
	}

	return &stateAccount{
		address: address,
		data:    data,
		onDirty: onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *stateAccount) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateAccount) touch() {
	if c.onDirty != nil {
		c.onDirty(c.Address())
		c.onDirty = nil
	}
}

// AddBalance removes amount from c's balance.
// It is used to add funds to the destination account of a transfer.
func (c *stateAccount) AddBalance(coinSymbol types.CoinSymbol, amount *big.Int) {
	if amount.Sign() == 0 {
		if c.empty() {
			c.touch()
		}

		return
	}
	c.SetBalance(coinSymbol, new(big.Int).Add(c.Balance(coinSymbol), amount))
}

// SubBalance removes amount from c's balance.
// It is used to remove funds from the origin account of a transfer.
func (c *stateAccount) SubBalance(coinSymbol types.CoinSymbol, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetBalance(coinSymbol, new(big.Int).Sub(c.Balance(coinSymbol), amount))
}

func (self *stateAccount) SetBalance(coinSymbol types.CoinSymbol, amount *big.Int) {
	EmitBalanceChange(self.address, coinSymbol, amount)
	self.setBalance(coinSymbol, amount)
}

func (self *stateAccount) setBalance(coinSymbol types.CoinSymbol, amount *big.Int) {

	if self.data.Balance.Data == nil {
		self.data.Balance.Data = make(map[types.CoinSymbol]*big.Int)
	}

	self.data.Balance.Data[coinSymbol] = amount
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *stateAccount) ReturnGas(gas *big.Int) {}

func (self *stateAccount) deepCopy(db *StateDB, onDirty func(addr types.Address)) *stateAccount {
	stateObject := newObject(db, self.address, self.data, onDirty)
	stateObject.deleted = self.deleted
	return stateObject
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (c *stateAccount) Address() types.Address {
	return c.address
}

func (self *stateAccount) SetNonce(nonce uint64) {
	self.setNonce(nonce)
}

func (self *stateAccount) setNonce(nonce uint64) {
	self.data.Nonce = nonce
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *stateAccount) Balance(coinSymbol types.CoinSymbol) *big.Int {

	if self.data.Balance.Data == nil {
		return big.NewInt(0)
	}

	if self.data.Balance.Data[coinSymbol] == nil {
		return big.NewInt(0)
	}

	return self.data.Balance.Data[coinSymbol]
}

func (self *stateAccount) Balances() Balances {
	return self.data.Balance
}

func (self *stateAccount) Nonce() uint64 {
	return self.data.Nonce
}
