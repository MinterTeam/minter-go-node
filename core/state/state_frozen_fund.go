package state

import (
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/formula"
	"io"

	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

// stateFrozenFund represents a frozen fund which is being modified.
type stateFrozenFund struct {
	blockHeight uint64
	deleted     bool
	data        FrozenFunds
	db          *StateDB

	onDirty func(blockHeight uint64)
}

type FrozenFund struct {
	Address      types.Address
	CandidateKey []byte
	Coin         types.CoinSymbol
	Value        *big.Int
}

type FrozenFunds struct {
	BlockHeight uint64
	List        []FrozenFund
}

func (f FrozenFunds) String() string {
	return fmt.Sprintf("Frozen funds at block %d (%d items)", f.BlockHeight, len(f.List))
}

// newFrozenFund creates a state frozen fund.
func newFrozenFund(db *StateDB, blockHeight uint64, data FrozenFunds,
	onDirty func(blockHeight uint64)) *stateFrozenFund {
	frozenFund := &stateFrozenFund{
		db:          db,
		blockHeight: blockHeight,
		data:        data,
		onDirty:     onDirty,
	}

	frozenFund.onDirty(frozenFund.blockHeight)

	return frozenFund
}

// EncodeRLP implements rlp.Encoder.
func (c *stateFrozenFund) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateFrozenFund) Delete() {
	c.deleted = true
	c.onDirty(c.blockHeight)
}

func (c *stateFrozenFund) AddFund(address types.Address, candidateKey []byte, coin types.CoinSymbol, value *big.Int) {
	c.addFund(FrozenFund{
		Address:      address,
		CandidateKey: candidateKey,
		Coin:         coin,
		Value:        value,
	})
}

func (c *stateFrozenFund) addFund(fund FrozenFund) {

	c.data.List = append(c.data.List, fund)

	if c.onDirty != nil {
		c.onDirty(c.blockHeight)
		c.onDirty = nil
	}
}

// punish fund with given candidate key (used in byzantine validator's punishment)
func (c *stateFrozenFund) PunishFund(context *StateDB, candidateAddress [20]byte, fromBlock uint64) {
	c.punishFund(context, candidateAddress, fromBlock)
}

func (c *stateFrozenFund) punishFund(context *StateDB, candidateAddress [20]byte, fromBlock uint64) {
	edb := eventsdb.GetCurrent()

	newList := make([]FrozenFund, len(c.data.List))
	for i, item := range c.data.List {
		// skip fund with given candidate key
		var pubkey ed25519.PubKeyEd25519
		copy(pubkey[:], item.CandidateKey)

		var address [20]byte
		copy(address[:], pubkey.Address().Bytes())

		if candidateAddress == address {
			newValue := big.NewInt(0).Set(item.Value)
			newValue.Mul(newValue, big.NewInt(95))
			newValue.Div(newValue, big.NewInt(100))

			slashed := big.NewInt(0).Set(item.Value)
			slashed.Sub(slashed, newValue)

			if !item.Coin.IsBaseCoin() {
				coin := context.GetStateCoin(item.Coin).Data()
				ret := formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, slashed)

				context.SubCoinVolume(coin.Symbol, slashed)
				context.SubCoinReserve(coin.Symbol, ret)

				context.AddTotalSlashed(ret)
			} else {
				context.AddTotalSlashed(slashed)
			}

			edb.AddEvent(fromBlock, eventsdb.SlashEvent{
				Address:         item.Address,
				Amount:          slashed.Bytes(),
				Coin:            item.Coin,
				ValidatorPubKey: item.CandidateKey,
			})

			item.Value = newValue
			context.DeleteCoinIfZeroReserve(item.Coin)
		}

		newList[i] = item
	}

	c.data.List = newList

	if c.onDirty != nil {
		c.onDirty(c.blockHeight)
		c.onDirty = nil
	}
}

//
// Attribute accessors
//

func (c *stateFrozenFund) BlockHeight() uint64 {
	return c.blockHeight
}

func (c *stateFrozenFund) List() []FrozenFund {
	return c.data.List
}

func (c *stateFrozenFund) Data() FrozenFunds {
	return c.data
}
