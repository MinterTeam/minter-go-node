package old

import (
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"io"

	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
)

// stateCoin represents a coin which is being modified.
type stateCoin struct {
	symbol    types.CoinSymbol
	data      Coin
	db        *StateDB
	isDeleted bool

	onDirty func(symbol types.CoinSymbol) // Callback method to mark a state coin newly dirty
}

type Coin struct {
	Name           string
	Symbol         types.CoinSymbol
	Volume         *big.Int
	Crr            uint
	ReserveBalance *big.Int
}

func (coin Coin) String() string {
	return fmt.Sprintf("%s (%s), volume: %s, reserve: %s, crr: %d", coin.Name, coin.Symbol, coin.Volume(),
		coin.Reserve(), coin.Crr())
}

func (c *stateCoin) IsToDelete() bool {
	if c == nil {
		return false
	}

	// Delete coin if reserve is less than 100 bips
	if c.ReserveBalance().Cmp(helpers.BipToPip(big.NewInt(100))) == -1 {
		return true
	}

	// Delete coin if volume is less than 1 coin
	if c.Volume().Cmp(helpers.BipToPip(big.NewInt(1))) == -1 {
		return true
	}

	// Delete coin if price of 1 coin is less than 0.0001 bip
	price := formula.CalculateSaleReturn(c.Volume(), c.ReserveBalance(), c.Crr(), helpers.BipToPip(big.NewInt(1)))
	minPrice := big.NewInt(100000000000000) // 0.0001 bip
	if price.Cmp(minPrice) == -1 {
		return true
	}

	return false
}

// newCoin creates a state coin.
func newCoin(db *StateDB, symbol types.CoinSymbol, data Coin, onDirty func(symbol types.CoinSymbol)) *stateCoin {
	coin := &stateCoin{
		db:      db,
		symbol:  symbol,
		data:    data,
		onDirty: onDirty,
	}

	coin.onDirty(coin.symbol)

	return coin
}

// EncodeRLP implements rlp.Encoder.
func (c *stateCoin) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateCoin) AddVolume(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetVolume(new(big.Int).Add(c.Volume(), amount))
}

func (c *stateCoin) SubVolume(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetVolume(new(big.Int).Sub(c.Volume(), amount))
}

func (c *stateCoin) SetVolume(amount *big.Int) {
	c.setVolume(amount)
}

func (c *stateCoin) setVolume(amount *big.Int) {

	c.data.Volume = amount

	if c.onDirty != nil {
		c.onDirty(c.Symbol())
		c.onDirty = nil
	}
}

func (c *stateCoin) AddReserve(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetReserve(new(big.Int).Add(c.ReserveBalance(), amount))
}

func (c *stateCoin) SubReserve(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetReserve(new(big.Int).Sub(c.ReserveBalance(), amount))
}

func (c *stateCoin) SetReserve(amount *big.Int) {
	c.setReserve(amount)
}

func (c *stateCoin) setReserve(amount *big.Int) {
	c.data.ReserveBalance = amount

	if c.onDirty != nil {
		c.onDirty(c.Symbol())
		c.onDirty = nil
	}
}

//
// Attribute accessors
//

func (c *stateCoin) Symbol() types.CoinSymbol {
	return c.symbol
}

func (c *stateCoin) Data() Coin {
	return c.data
}

func (c *stateCoin) Volume() *big.Int {
	return c.data.Volume
}

func (c *stateCoin) ReserveBalance() *big.Int {
	return c.data.ReserveBalance
}

func (c *stateCoin) Crr() uint {
	return c.data.Crr
}

func (c *stateCoin) Name() string {
	return c.data.Name
}
