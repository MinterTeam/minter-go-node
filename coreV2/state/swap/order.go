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

func (p *Pair) setBuyHigherOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := calcRate(amountSell, amountBuy)
	var index int
	orders := p.BuyHigherOrders()
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
		p.SetBuyHigherOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(orders) {
		p.SetBuyHigherOrders(append(orders, limit))
		return
	}

	p.SetBuyHigherOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (p *Pair) setBuyLowerOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := calcRate(amountSell, amountBuy)
	var index int
	orders := p.BuyLowerOrders()
	for i, limit := range orders {
		if rate.Cmp(limit.Rate()) != -1 { // todo
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
		p.SetBuyLowerOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(orders) {
		p.SetBuyLowerOrders(append(orders, limit))
		return
	}

	p.SetBuyLowerOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (p *Pair) setSellHigherOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := calcRate(amountSell, amountBuy)
	var index int
	orders := p.SellHigherOrders()
	for i, limit := range orders {
		if rate.Cmp(limit.Rate()) != -1 { // todo
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
		p.SetSellHigherOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(orders) {
		p.SetSellHigherOrders(append(orders, limit))
		return
	}

	p.SetSellHigherOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (p *Pair) setSellLowerOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := calcRate(amountSell, amountBuy)
	var index int
	orders := p.SellLowerOrders()
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
		p.SetSellLowerOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(p.sellOrders.lower) {
		p.SetSellLowerOrders(append(orders, limit))
		return
	}

	p.SetSellLowerOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (p *Pair) SellLowerOrders() []*Limit {
	// if p.isSorted() {
	return p.sellOrders.lower
	// }
	// return p.buyOrders.higher
}

func (p *Pair) SellHigherOrders() []*Limit {
	if p.isSorted() {
		return p.sellOrders.higher
	}
	return p.buyOrders.lower
}
func (p *Pair) BuyHigherOrders() []*Limit {
	// if p.isSorted() {
	return p.buyOrders.higher
	// }
	// return p.sellOrders.lower
}
func (p *Pair) BuyLowerOrders() []*Limit {
	// if p.isSorted() {
	return p.buyOrders.lower
	// }
	// return p.sellOrders.higher
}

func (p *Pair) SetSellLowerOrders(orders []*Limit) {
	// if p.isSorted() {
	p.sellOrders.lower = orders
	return
	// }
	// p.buyOrders.higher = orders
	// return
}
func (p *Pair) SetSellHigherOrders(orders []*Limit) {
	// if p.isSorted() {
	p.sellOrders.higher = orders
	// return
	// }
	// p.buyOrders.lower = orders
	// return
}
func (p *Pair) SetBuyHigherOrders(orders []*Limit) {
	// if p.isSorted() {
	p.buyOrders.higher = orders
	// return
	// }
	// p.sellOrders.lower = orders
	// return
}
func (p *Pair) SetBuyLowerOrders(orders []*Limit) {
	// if p.isSorted() {
	p.buyOrders.lower = orders
	// return
	// }
	// p.sellOrders.higher = orders
	// return
}

func (s *Swap) PairAddSellOrder(coin0, coin1 types.CoinID, sellAmount, buyAmount *big.Int) uint32 {
	pair := s.Pair(coin0, coin1)
	orderID := pair.SetOrder(sellAmount, buyAmount)

	s.bus.Checker().AddCoin(coin0, sellAmount)

	return orderID
}

func (p *Pair) SetOrder(sellAmount, buyAmount *big.Int) (id uint32) {
	var order *Limit
	rate := calcRate(sellAmount, buyAmount)
	// if p.isSorted() {
	if p.Rate().Cmp(rate) == -1 {
		order = p.setSellHigherOrder(sellAmount, buyAmount)
	} else {
		order = p.setSellLowerOrder(sellAmount, buyAmount)
	}
	// } else {
	// 	if p.SortRate().Cmp(rate) == 1 {
	// 		order = p.setBuyHigherOrder(sellAmount, buyAmount)
	// 	} else {
	// 		order = p.setBuyLowerOrder(sellAmount, buyAmount)
	// 	}
	// }

	return order.id
}

// loadBuyHigherOrders loads only needed orders for pair, not all
func (s *Swap) loadBuyHigherOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex

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

func (s *Swap) loadSellLowerOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex

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

func (p *Pair) OrderBuyHigherByIndex(index int) *Limit {
	if len(p.BuyHigherOrders()) <= index {
		p.SetBuyHigherOrders(p.loadHigherOrders(p, p.BuyHigherOrders(), index))
	}
	if len(p.BuyHigherOrders())-1 < index {
		return nil
	}
	return p.BuyHigherOrders()[index]
}

func (p *Pair) OrderBuyHigherLast() (limit *Limit, index int) {
	for order := p.OrderBuyHigherByIndex(index); order != nil; order = p.OrderBuyHigherByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}

func (p *Pair) OrderSellLowerByIndex(index int) *Limit {
	if len(p.SellLowerOrders()) <= index {
		p.SetSellLowerOrders(p.loadLowerOrders(p, p.SellLowerOrders(), index))
	}
	if len(p.SellLowerOrders())-1 < index {
		return nil
	}
	return p.SellLowerOrders()[index]
}

func (p *Pair) OrderSellLowerLast() (limit *Limit, index int) {
	for order := p.OrderSellLowerByIndex(index); order != nil; order = p.OrderSellLowerByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}
