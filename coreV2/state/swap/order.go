package swap

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math"
	"math/big"
)

type Order struct {
	*Limit
	isDrop bool
}

type Limit struct {
	isBuy bool
	Coin0 *big.Int
	Coin1 *big.Int

	rate *big.Float
	id   uint32
}

type limits struct {
	higher []*Limit
	lower  []*Limit
	// todo: add mutex
}

type dirtyOrders struct {
	orders []*Order
}

const (
	precision = 54 // supported precision
)

func (l *Limit) Rate() *big.Float {
	if l.rate == nil {
		l.rate = big.NewFloat(0).SetPrec(precision).Quo(
			big.NewFloat(0).SetInt(l.Coin0),
			big.NewFloat(0).SetInt(l.Coin1))
	}
	return l.rate
}

func (l *Limit) SortRate() *big.Float {
	rate := big.NewFloat(0).SetPrec(precision)
	if !l.isBuy {
		rate.Quo(
			big.NewFloat(0).SetInt(l.Coin0),
			big.NewFloat(0).SetInt(l.Coin1))
	} else {
		rate.Quo(
			big.NewFloat(0).SetInt(l.Coin1),
			big.NewFloat(0).SetInt(l.Coin0))
	}
	return rate
}

func (l *Limit) reverse() *Limit {
	return &Limit{
		isBuy: !l.isBuy,
		Coin0: l.Coin1,
		Coin1: l.Coin0,
		rate:  nil,
		id:    l.id,
	}
}

func (l *Limit) sort() *Limit {
	if !l.isBuy {
		return l
	}
	return &Limit{
		isBuy: false,
		Coin0: l.Coin1,
		Coin1: l.Coin0,
		rate:  nil,
		id:    l.id,
	}
}

func (p *Pair) MarkDirtyOrders(order *Order) {
	p.markDirtyOrders()
	p.dirtyOrders.orders = append(p.dirtyOrders.orders, order)
	return
}

func (p *Pair) setHigherOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := big.NewFloat(0).SetPrec(precision).Quo(big.NewFloat(0).SetInt(amountSell), big.NewFloat(0).SetInt(amountBuy))
	var index int
	orders := p.HigherOrders()
	for i, limit := range orders {
		if rate.Cmp(limit.Rate()) != -1 {
			index = i + 1
			continue
		}
		break
	}

	limit = &Limit{
		isBuy: !p.isSorted(),
		Coin0: amountSell,
		Coin1: amountBuy,
		rate:  rate,
		id:    p.getLastTotalOrderID(),
	}
	defer p.MarkDirtyOrders(&Order{
		Limit:  limit,
		isDrop: false,
	})

	if index == 0 {
		p.SetHigherOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(orders) {
		p.SetHigherOrders(append(orders, limit))
		return
	}

	p.SetHigherOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (p *Pair) LowerOrders() []*Limit {
	if p.isSorted() {
		return p.sellOrders.lower
	}
	return p.buyOrders.lower
}
func (p *Pair) HigherOrders() []*Limit {
	if p.isSorted() {
		return p.sellOrders.higher
	}
	return p.buyOrders.higher
}

func (p *Pair) SetLowerOrders(orders []*Limit) {
	if p.isSorted() {
		p.sellOrders.lower = orders
		return
	}
	p.buyOrders.lower = orders
	return
}
func (p *Pair) SetHigherOrders(orders []*Limit) {
	if p.isSorted() {
		p.sellOrders.higher = orders
		return
	}
	p.buyOrders.higher = orders
	return
}

func (p *Pair) setLowerOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := big.NewFloat(0).SetPrec(precision).Quo(big.NewFloat(0).SetInt(amountSell), big.NewFloat(0).SetInt(amountBuy))
	var index int
	orders := p.LowerOrders()
	for i, limit := range orders {
		if rate.Cmp(limit.Rate()) != -1 {
			index = i + 1
			continue
		}
		break
	}

	limit = &Limit{
		isBuy: !p.isSorted(),
		Coin0: amountSell,
		Coin1: amountBuy,
		rate:  rate,
		id:    p.getLastTotalOrderID(),
	}
	defer p.MarkDirtyOrders(&Order{
		Limit:  limit,
		isDrop: false,
	})

	if index == 0 {
		p.SetLowerOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(p.sellOrders.lower) {
		p.SetLowerOrders(append(orders, limit))
		return
	}

	p.SetLowerOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (s *Swap) PairAddSellLimit(coin0, coin1 types.CoinID, sellAmount, buyAmount *big.Int) uint32 {
	pair := s.Pair(coin0, coin1)
	orderID := pair.SetOrder(sellAmount, buyAmount)

	s.bus.Checker().AddCoin(coin0, sellAmount)

	return orderID
}

func (p *Pair) SetOrder(sellAmount, buyAmount *big.Int) (id uint32) {
	var order *Limit
	if p.Rate().Cmp(calcRate(sellAmount, buyAmount)) == -1 {
		order = p.setHigherOrder(sellAmount, buyAmount)
	} else {
		order = p.setLowerOrder(sellAmount, buyAmount)
	}

	return order.id
}

// loadHigherOrders loads only needed orders for pair, not all
func (s *Swap) loadHigherOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex

	var isSale byte = 0
	if pair.isSorted() {
		isSale = 1
	}

	var startKey []byte
	endKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(isSale), byte(255)) // todo: mb more high bytes

	if len(slice) > 0 {
		var l = slice[len(slice)-1]
		startKey = ratePath(pair.pairKey, l.SortRate(), l.id+1, pair.isSorted())
	} else {
		startKey = ratePath(pair.pairKey, pair.SortRate(), 0, pair.isSorted())
	}

	i := limit - len(slice)
	s.immutableTree().IterateRange(startKey, endKey, true, func(key []byte, value []byte) bool {
		if i > limit {
			return true
		}

		order := &Limit{id: binary.BigEndian.Uint32(key[len(key)-4:])}
		err := rlp.DecodeBytes(value, order)
		if err != nil {
			panic(err)
		}
		if !pair.pairKey.isSorted() {
			order = order.reverse()
		}
		slice = append(slice, order)
		i++
		return false
	})

	return slice
}

func (s *Swap) loadLowerOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex

	var isSale byte = 0
	if pair.isSorted() {
		isSale = 1
	}

	startKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(isSale), byte(0))
	var endKey []byte

	if len(slice) > 0 {
		var l = slice[len(slice)-1]
		endKey = ratePath(pair.pairKey, l.SortRate(), l.id-1, pair.isSorted())
	} else {
		endKey = ratePath(pair.pairKey, pair.SortRate(), math.MaxInt32, pair.isSorted())
	}

	i := limit - len(slice)
	s.immutableTree().IterateRange(startKey, endKey, false, func(key []byte, value []byte) bool {
		if i > limit {
			return true
		}

		order := &Limit{id: binary.BigEndian.Uint32(key[len(key)-4:])}
		err := rlp.DecodeBytes(value, order)
		if err != nil {
			panic(err)
		}
		if !pair.isSorted() {
			order = order.reverse()
		}
		slice = append(slice, order)
		i++
		return false
	})

	return slice
}

func (p *Pair) OrderHigherByIndex(index int) *Limit {
	if len(p.HigherOrders()) <= index {
		p.SetHigherOrders(p.loadHigherOrders(p, p.HigherOrders(), index))
	}
	if len(p.HigherOrders())-1 < index {
		return nil
	}
	return p.HigherOrders()[index]
}

func (p *Pair) OrderHigherLast() (limit *Limit, index int) {
	for order := p.OrderHigherByIndex(index); order != nil; order = p.OrderHigherByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}

func (p *Pair) OrderLowerByIndex(index int) *Limit {
	if len(p.LowerOrders()) <= index {
		p.SetLowerOrders(p.loadLowerOrders(p, p.LowerOrders(), index))
	}
	if len(p.LowerOrders())-1 < index {
		return nil
	}
	return p.LowerOrders()[index]
}

func (p *Pair) OrderLowerLast() (limit *Limit, index int) {
	for order := p.OrderLowerByIndex(index); order != nil; order = p.OrderLowerByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}
