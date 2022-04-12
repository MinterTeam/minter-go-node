package commission

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
)

const mainPrefix = byte('p')

type RCommission interface {
	// Deprecated
	ExportV1(state *types.AppState, id types.CoinID)

	Export(state *types.AppState)
	GetVotes(height uint64) []*Model
	GetCommissions() *Price
	IsVoteExists(height uint64, pubkey types.Pubkey) bool
}

type Commission struct {
	list      map[uint64][]*Model
	dirty     map[uint64]struct{}
	forDelete uint64

	currentPrice *Price
	dirtyCurrent bool

	db   atomic.Value
	lock sync.RWMutex
}

func NewCommission(db *iavl.ImmutableTree) *Commission {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	halts := &Commission{
		db:        immutableTree,
		list:      map[uint64][]*Model{},
		dirty:     map[uint64]struct{}{},
		forDelete: 0,
	}

	return halts
}

func (c *Commission) immutableTree() *iavl.ImmutableTree {
	db := c.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (c *Commission) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	c.db.Store(immutableTree)
}

func (c *Commission) Export(state *types.AppState) {
	c.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		if len(key) < 8 {
			return false
		}
		height := binary.LittleEndian.Uint64(key[1:])
		prices := c.get(height)
		if prices == nil {
			return false
		}

		for _, price := range prices {
			p := Decode(price.Price)
			state.CommissionVotes = append(state.CommissionVotes, types.CommissionVote{
				Height: height,
				Votes:  price.Votes,
				Commission: types.Commission{
					Coin:                    uint64(p.Coin),
					PayloadByte:             p.PayloadByte.String(),
					Send:                    p.Send.String(),
					BuyBancor:               p.BuyBancor.String(),
					SellBancor:              p.SellBancor.String(),
					SellAllBancor:           p.SellAllBancor.String(),
					BuyPoolBase:             p.BuyPoolBase.String(),
					BuyPoolDelta:            p.BuyPoolDelta.String(),
					SellPoolBase:            p.SellPoolBase.String(),
					SellPoolDelta:           p.SellPoolDelta.String(),
					SellAllPoolBase:         p.SellAllPoolBase.String(),
					SellAllPoolDelta:        p.SellAllPoolDelta.String(),
					CreateTicker3:           p.CreateTicker3.String(),
					CreateTicker4:           p.CreateTicker4.String(),
					CreateTicker5:           p.CreateTicker5.String(),
					CreateTicker6:           p.CreateTicker6.String(),
					CreateTicker7_10:        p.CreateTicker7to10.String(),
					CreateCoin:              p.CreateCoin.String(),
					CreateToken:             p.CreateToken.String(),
					RecreateCoin:            p.RecreateCoin.String(),
					RecreateToken:           p.RecreateToken.String(),
					DeclareCandidacy:        p.DeclareCandidacy.String(),
					Delegate:                p.Delegate.String(),
					Unbond:                  p.Unbond.String(),
					RedeemCheck:             p.RedeemCheck.String(),
					SetCandidateOn:          p.SetCandidateOn.String(),
					SetCandidateOff:         p.SetCandidateOff.String(),
					CreateMultisig:          p.CreateMultisig.String(),
					MultisendBase:           p.MultisendBase.String(),
					MultisendDelta:          p.MultisendDelta.String(),
					EditCandidate:           p.EditCandidate.String(),
					SetHaltBlock:            p.SetHaltBlock.String(),
					EditTickerOwner:         p.EditTickerOwner.String(),
					EditMultisig:            p.EditMultisig.String(),
					EditCandidatePublicKey:  p.EditCandidatePublicKey.String(),
					CreateSwapPool:          p.CreateSwapPool.String(),
					AddLiquidity:            p.AddLiquidity.String(),
					RemoveLiquidity:         p.RemoveLiquidity.String(),
					EditCandidateCommission: p.EditCandidateCommission.String(),
					MintToken:               p.MintToken.String(),
					BurnToken:               p.BurnToken.String(),
					VoteCommission:          p.VoteCommission.String(),
					VoteUpdate:              p.VoteUpdate.String(),
					FailedTx:                p.FailedTx.String(),
					AddLimitOrder:           p.AddLimitOrder.String(),
					RemoveLimitOrder:        p.RemoveLimitOrder.String(),
					MoveStake:               p.MoveStake.String(),
					LockStake:               p.LockStake.String(),
					Lock:                    p.Lock.String(),
				},
			})
		}

		return false
	})

	current := c.GetCommissions()
	state.Commission = types.Commission{
		Coin:                    uint64(current.Coin),
		PayloadByte:             current.PayloadByte.String(),
		Send:                    current.Send.String(),
		BuyBancor:               current.BuyBancor.String(),
		SellBancor:              current.SellBancor.String(),
		SellAllBancor:           current.SellAllBancor.String(),
		BuyPoolBase:             current.BuyPoolBase.String(),
		BuyPoolDelta:            current.BuyPoolDelta.String(),
		SellPoolBase:            current.SellPoolBase.String(),
		SellPoolDelta:           current.SellPoolDelta.String(),
		SellAllPoolBase:         current.SellAllPoolBase.String(),
		SellAllPoolDelta:        current.SellAllPoolDelta.String(),
		CreateTicker3:           current.CreateTicker3.String(),
		CreateTicker4:           current.CreateTicker4.String(),
		CreateTicker5:           current.CreateTicker5.String(),
		CreateTicker6:           current.CreateTicker6.String(),
		CreateTicker7_10:        current.CreateTicker7to10.String(),
		CreateCoin:              current.CreateCoin.String(),
		CreateToken:             current.CreateToken.String(),
		RecreateCoin:            current.RecreateCoin.String(),
		RecreateToken:           current.RecreateToken.String(),
		DeclareCandidacy:        current.DeclareCandidacy.String(),
		Delegate:                current.Delegate.String(),
		Unbond:                  current.Unbond.String(),
		RedeemCheck:             current.RedeemCheck.String(),
		SetCandidateOn:          current.SetCandidateOn.String(),
		SetCandidateOff:         current.SetCandidateOff.String(),
		CreateMultisig:          current.CreateMultisig.String(),
		MultisendBase:           current.MultisendBase.String(),
		MultisendDelta:          current.MultisendDelta.String(),
		EditCandidate:           current.EditCandidate.String(),
		SetHaltBlock:            current.SetHaltBlock.String(),
		EditTickerOwner:         current.EditTickerOwner.String(),
		EditMultisig:            current.EditMultisig.String(),
		EditCandidatePublicKey:  current.EditCandidatePublicKey.String(),
		CreateSwapPool:          current.CreateSwapPool.String(),
		AddLiquidity:            current.AddLiquidity.String(),
		RemoveLiquidity:         current.RemoveLiquidity.String(),
		EditCandidateCommission: current.EditCandidateCommission.String(),
		MintToken:               current.MintToken.String(),
		BurnToken:               current.BurnToken.String(),
		VoteCommission:          current.VoteCommission.String(),
		VoteUpdate:              current.VoteUpdate.String(),
		FailedTx:                current.FailedTx.String(),
		AddLimitOrder:           current.AddLimitOrder.String(),
		RemoveLimitOrder:        current.RemoveLimitOrder.String(),
		MoveStake:               current.MoveStake.String(),
		LockStake:               current.LockStake.String(),
		Lock:                    current.Lock.String(),
	}
}

// Deprecated
func (c *Commission) ExportV1(state *types.AppState, id types.CoinID) {
	if id == 0 {
		state.Commission = types.Commission{
			Coin:                    uint64(types.GetBaseCoinID()),
			PayloadByte:             helpers.StringToBigInt("200000000000000000").String(),
			Send:                    helpers.StringToBigInt("1000000000000000000").String(),
			BuyBancor:               helpers.StringToBigInt("10000000000000000000").String(),
			SellBancor:              helpers.StringToBigInt("10000000000000000000").String(),
			SellAllBancor:           helpers.StringToBigInt("10000000000000000000").String(),
			BuyPoolBase:             helpers.StringToBigInt("10000000000000000000").String(),
			BuyPoolDelta:            helpers.StringToBigInt("5000000000000000000").String(),
			SellPoolBase:            helpers.StringToBigInt("10000000000000000000").String(),
			SellPoolDelta:           helpers.StringToBigInt("5000000000000000000").String(),
			SellAllPoolBase:         helpers.StringToBigInt("10000000000000000000").String(),
			SellAllPoolDelta:        helpers.StringToBigInt("5000000000000000000").String(),
			CreateTicker3:           helpers.StringToBigInt("100000000000000000000000000").String(),
			CreateTicker4:           helpers.StringToBigInt("10000000000000000000000000").String(),
			CreateTicker5:           helpers.StringToBigInt("1000000000000000000000000").String(),
			CreateTicker6:           helpers.StringToBigInt("100000000000000000000000").String(),
			CreateTicker7_10:        helpers.StringToBigInt("10000000000000000000000").String(),
			CreateCoin:              helpers.StringToBigInt("0").String(),
			CreateToken:             helpers.StringToBigInt("0").String(),
			RecreateCoin:            helpers.StringToBigInt("1000000000000000000000000").String(),
			RecreateToken:           helpers.StringToBigInt("1000000000000000000000000").String(),
			DeclareCandidacy:        helpers.StringToBigInt("1000000000000000000000").String(),
			Delegate:                helpers.StringToBigInt("20000000000000000000").String(),
			Unbond:                  helpers.StringToBigInt("20000000000000000000").String(),
			RedeemCheck:             helpers.StringToBigInt("3000000000000000000").String(),
			SetCandidateOn:          helpers.StringToBigInt("10000000000000000000").String(),
			SetCandidateOff:         helpers.StringToBigInt("10000000000000000000").String(),
			CreateMultisig:          helpers.StringToBigInt("10000000000000000000").String(),
			MultisendBase:           helpers.StringToBigInt("1000000000000000000").String(),
			MultisendDelta:          helpers.StringToBigInt("500000000000000000").String(),
			EditCandidate:           helpers.StringToBigInt("1000000000000000000000").String(),
			SetHaltBlock:            helpers.StringToBigInt("100000000000000000000").String(),
			EditTickerOwner:         helpers.StringToBigInt("1000000000000000000000000").String(),
			EditMultisig:            helpers.StringToBigInt("100000000000000000000").String(),
			EditCandidatePublicKey:  helpers.StringToBigInt("10000000000000000000000000").String(),
			CreateSwapPool:          helpers.StringToBigInt("100000000000000000000").String(),
			AddLiquidity:            helpers.StringToBigInt("10000000000000000000").String(),
			RemoveLiquidity:         helpers.StringToBigInt("10000000000000000000").String(),
			EditCandidateCommission: helpers.StringToBigInt("1000000000000000000000").String(),
			BurnToken:               helpers.StringToBigInt("10000000000000000000").String(),
			MintToken:               helpers.StringToBigInt("10000000000000000000").String(),
			VoteCommission:          helpers.StringToBigInt("100000000000000000000").String(),
			VoteUpdate:              helpers.StringToBigInt("100000000000000000000").String(),
		}
		return
	}
	state.Commission = types.Commission{
		Coin:                    uint64(id),
		PayloadByte:             helpers.FloatBipToPip(0.002).String(),
		Send:                    helpers.FloatBipToPip(0.01).String(),
		BuyBancor:               helpers.FloatBipToPip(0.03).String(),
		SellBancor:              helpers.FloatBipToPip(0.03).String(),
		SellAllBancor:           helpers.FloatBipToPip(0.03).String(),
		BuyPoolBase:             helpers.FloatBipToPip(0.03).String(),
		BuyPoolDelta:            helpers.FloatBipToPip(0.005).String(),
		SellPoolBase:            helpers.FloatBipToPip(0.03).String(),
		SellPoolDelta:           helpers.FloatBipToPip(0.005).String(),
		SellAllPoolBase:         helpers.FloatBipToPip(0.03).String(),
		SellAllPoolDelta:        helpers.FloatBipToPip(0.005).String(),
		CreateTicker3:           helpers.BipToPip(big.NewInt(100000)).String(),
		CreateTicker4:           helpers.FloatBipToPip(10000).String(),
		CreateTicker5:           helpers.FloatBipToPip(1000).String(),
		CreateTicker6:           helpers.FloatBipToPip(100).String(),
		CreateTicker7_10:        helpers.FloatBipToPip(10).String(),
		CreateCoin:              helpers.FloatBipToPip(0).String(),
		CreateToken:             helpers.FloatBipToPip(0).String(),
		RecreateCoin:            helpers.FloatBipToPip(100).String(),
		RecreateToken:           helpers.FloatBipToPip(100).String(),
		DeclareCandidacy:        helpers.FloatBipToPip(100).String(),
		Delegate:                helpers.FloatBipToPip(0.1).String(),
		Unbond:                  helpers.FloatBipToPip(0.1).String(),
		RedeemCheck:             helpers.FloatBipToPip(0.03).String(),
		SetCandidateOn:          helpers.FloatBipToPip(10).String(),
		SetCandidateOff:         helpers.FloatBipToPip(10).String(),
		CreateMultisig:          helpers.FloatBipToPip(0.1).String(),
		MultisendBase:           helpers.FloatBipToPip(0.01).String(),
		MultisendDelta:          helpers.FloatBipToPip(0.005).String(),
		EditCandidate:           helpers.FloatBipToPip(100).String(),
		SetHaltBlock:            helpers.FloatBipToPip(0.01).String(),
		EditTickerOwner:         helpers.FloatBipToPip(100).String(),
		EditMultisig:            helpers.FloatBipToPip(0.01).String(),
		EditCandidatePublicKey:  helpers.FloatBipToPip(10000).String(),
		CreateSwapPool:          helpers.FloatBipToPip(0.1).String(),
		AddLiquidity:            helpers.FloatBipToPip(0.03).String(),
		RemoveLiquidity:         helpers.FloatBipToPip(0.03).String(),
		EditCandidateCommission: helpers.FloatBipToPip(100).String(),
		MintToken:               helpers.FloatBipToPip(0.01).String(),
		BurnToken:               helpers.FloatBipToPip(0.01).String(),
		VoteCommission:          helpers.FloatBipToPip(1).String(),
		VoteUpdate:              helpers.FloatBipToPip(1).String(),
	}
}

func (c *Commission) Commit(db *iavl.MutableTree, version int64) error {
	c.lock.Lock()
	if c.dirtyCurrent {
		c.dirtyCurrent = false
		db.Set([]byte{mainPrefix}, c.currentPrice.Encode())
	}
	dirties := c.getOrderedDirty()
	c.lock.Unlock()
	for _, height := range dirties {
		models := c.getFromMap(height)

		c.lock.Lock()
		delete(c.dirty, height)
		c.lock.Unlock()

		data, err := rlp.EncodeToBytes(models)
		if err != nil {
			return fmt.Errorf("can't encode object at %d: %v", height, err)
		}

		db.Set(getPath(height), data)
	}

	if c.forDelete != 0 {
		path := getPath(c.forDelete)
		db.Remove(path)
		c.lock.Lock()
		delete(c.list, c.forDelete)
		c.forDelete = 0
		c.lock.Unlock()
	}

	return nil
}

func (c *Commission) GetVotes(height uint64) []*Model {
	return c.get(height)
}

func (c *Commission) GetCommissions() *Price {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.currentPrice != nil {
		return c.currentPrice
	}
	_, value := c.immutableTree().Get([]byte{mainPrefix})
	if len(value) == 0 {
		return nil
	}
	c.currentPrice = &Price{}
	err := rlp.DecodeBytes(value, c.currentPrice)
	if err != nil {
		panic(err)
	}
	return c.currentPrice
}

func (c *Commission) SetNewCommissions(prices []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.dirtyCurrent = true
	var newPrices Price
	err := rlp.DecodeBytes(prices, &newPrices)
	if err != nil {
		panic(err) // todo: if update network after price vote, clean up following blocks
	}
	c.currentPrice = &newPrices
}

func (c *Commission) getOrNew(height uint64, encode string) *Model {
	prices := c.get(height)

	for _, model := range prices {
		if encode == model.Price {
			return model
		}
	}

	price := &Model{
		Votes:     []types.Pubkey{},
		Price:     encode,
		height:    height,
		markDirty: c.markDirty(height),
	}
	c.setToMap(height, append(prices, price))

	return price
}

func (c *Commission) get(height uint64) []*Model {
	if haltBlock := c.getFromMap(height); haltBlock != nil {
		return haltBlock
	}

	_, enc := c.immutableTree().Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	var voteBlock []*Model
	if err := rlp.DecodeBytes(enc, &voteBlock); err != nil {
		panic(fmt.Sprintf("failed to decode halt blocks at height %d: %s", height, err))
	}

	for _, vote := range voteBlock {
		vote.markDirty = c.markDirty(height)
		vote.height = height
	}

	c.setToMap(height, voteBlock)

	return voteBlock
}

func (c *Commission) markDirty(height uint64) func() {
	return func() {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.dirty[height] = struct{}{}
	}
}

func (c *Commission) getOrderedDirty() []uint64 {
	keys := make([]uint64, 0, len(c.dirty))
	for k := range c.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (c *Commission) IsVoteExists(height uint64, pubkey types.Pubkey) bool {
	model := c.get(height)
	if len(model) == 0 {
		return false
	}

	for _, price := range model {
		for _, vote := range price.Votes {
			if vote == pubkey {
				return true
			}
		}
	}

	return false
}

func (c *Commission) AddVote(height uint64, pubkey types.Pubkey, encode []byte) {
	c.getOrNew(height, string(encode)).addVote(pubkey)
}

func (c *Commission) Delete(height uint64) {
	prices := c.get(height)
	if len(prices) == 0 {
		return
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	c.forDelete = height
}

func (c *Commission) getFromMap(height uint64) []*Model {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[height]
}

func (c *Commission) setToMap(height uint64, model []*Model) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[height] = model
}

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
