package swap

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math"
	"math/big"
)

func calcPriceSell(sell, buy *big.Int) *big.Float {
	return new(big.Float).SetPrec(precision).Quo(
		big.NewFloat(0).SetInt(buy),
		big.NewFloat(0).SetInt(sell),
	)
}

type Order struct {
	*Limit
	isDrop bool
}

type Limit struct {
	isBuy bool
	Coin0 *big.Int
	Coin1 *big.Int

	price *big.Float
	id    uint32
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

func (l *Limit) Price() *big.Float {
	if l.price == nil {
		l.price = calcPriceSell(l.Coin0, l.Coin1)
	}
	return l.price
}

func (l *Limit) SortPrice() *big.Float {
	if !l.isBuy {
		return calcPriceSell(l.Coin0, l.Coin1) // not l.Price()
	}
	return calcPriceSell(l.Coin1, l.Coin0)
}

func (l *Limit) reverse() *Limit {
	return &Limit{
		isBuy: !l.isBuy,
		Coin0: l.Coin1,
		Coin1: l.Coin0,
		price: nil,
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
		price: nil,
		id:    l.id,
	}
}

func (p *Pair) MarkDirtyOrders(order *Order) {
	p.markDirtyOrders()
	p.dirtyOrders.orders = append(p.dirtyOrders.orders, order)
	return
}

func (p *Pair) setSellHigherOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	price := calcPriceSell(amountSell, amountBuy)
	var index int
	orders := p.sellHigherOrders()
	for i, limit := range orders {
		if price.Cmp(limit.Price()) != -1 {
			index = i + 1
			continue
		}
		break
	}

	limit = &Limit{
		isBuy: !p.isSorted(),
		Coin0: amountSell,
		Coin1: amountBuy,
		id:    p.getLastTotalOrderID(),
	}
	defer p.MarkDirtyOrders(&Order{
		Limit:  limit,
		isDrop: false,
	})

	if index == 0 {
		p.setSellHigherOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(orders) {
		p.setSellHigherOrders(append(orders, limit))
		return
	}

	p.setSellHigherOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (p *Pair) setSellLowerOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	price := calcPriceSell(amountSell, amountBuy)
	var index int
	orders := p.SellLowerOrders()
	for i, limit := range orders {
		if price.Cmp(limit.Price()) != 1 {
			index = i + 1
			continue
		}
		break
	}

	limit = &Limit{
		isBuy: !p.isSorted(),
		Coin0: amountSell,
		Coin1: amountBuy,
		id:    p.getLastTotalOrderID(),
	}
	defer p.MarkDirtyOrders(&Order{
		Limit:  limit,
		isDrop: false,
	})

	if index == 0 {
		p.setSellLowerOrders(append([]*Limit{limit}, orders...))
		return
	}

	if index == len(p.sellOrders.lower) {
		p.setSellLowerOrders(append(orders, limit))
		return
	}

	p.setSellLowerOrders(append(orders[:index], append([]*Limit{limit}, orders[index:]...)...))
	return
}

func (p *Pair) SellLowerOrders() []*Limit {
	if p.isSorted() {
		return p.sellOrders.lower
	}
	return p.buyOrders.higher
}

func (p *Pair) sellHigherOrders() []*Limit {
	if p.isSorted() {
		return p.sellOrders.higher
	}
	return p.buyOrders.lower
}
func (p *Pair) BuyHigherOrders() []*Limit {
	if p.isSorted() {
		return p.buyOrders.higher
	}
	return p.sellOrders.lower
}
func (p *Pair) buyLowerOrders() []*Limit {
	if p.isSorted() {
		return p.buyOrders.lower
	}
	return p.sellOrders.higher
}

func (p *Pair) setSellLowerOrders(orders []*Limit) {
	if p.isSorted() {
		p.sellOrders.lower = orders
		return
	}
	p.buyOrders.higher = orders
	return
}
func (p *Pair) setSellHigherOrders(orders []*Limit) {
	if p.isSorted() {
		p.sellOrders.higher = orders
		return
	}
	p.buyOrders.lower = orders
	return
}
func (p *Pair) setBuyHigherOrders(orders []*Limit) {
	if p.isSorted() {
		p.buyOrders.higher = orders
		return
	}
	p.sellOrders.lower = orders
	return
}
func (p *Pair) setBuyLowerOrders(orders []*Limit) {
	if p.isSorted() {
		p.buyOrders.lower = orders
		return
	}
	p.sellOrders.higher = orders
	return
}

func (s *Swap) PairAddSellOrder(coin0, coin1 types.CoinID, sellAmount, buyAmount *big.Int) uint32 {
	pair := s.Pair(coin0, coin1)
	orderID := pair.SetOrder(sellAmount, buyAmount)

	s.bus.Checker().AddCoin(coin0, sellAmount)

	return orderID
}

func (p *Pair) SetOrder(sellAmount, buyAmount *big.Int) (id uint32) {
	var order *Limit
	price := calcPriceSell(sellAmount, buyAmount)
	currantPrice := p.Price()
	if currantPrice.Cmp(price) == -1 {
		order = p.setSellHigherOrder(sellAmount, buyAmount)
	} else {
		order = p.setSellLowerOrder(sellAmount, buyAmount)
	}

	return order.id
}

// loadBuyHigherOrders loads only needed orders for pair, not all
func (s *Swap) loadBuyHigherOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex
	endKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(0), byte(255)) // todo: mb more high bytes
	var startKey []byte

	if len(slice) > 0 {
		var l = slice[len(slice)-1]
		startKey = pricePath(pair.pairKey, l.SortPrice(), l.id+1, false)
	} else {
		startKey = pricePath(pair.pairKey, pair.SortPrice(), 0, false)
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
			// order = order.reverse()
		}
		slice = append(slice, order)
		i++
		return false
	})

	return slice
}

func (s *Swap) loadSellLowerOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex
	startKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(1), byte(0))
	var endKey []byte

	if len(slice) > 0 {
		var l = slice[len(slice)-1]
		endKey = pricePath(pair.pairKey, l.SortPrice(), l.id-1, true)
	} else {
		endKey = pricePath(pair.pairKey, pair.SortPrice(), math.MaxInt32, true)
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
			// order = order.reverse()
		}
		slice = append(slice, order)
		i++
		return false
	})

	return slice
}

func (p *Pair) OrderBuyHigherByIndex(index int) *Limit {
	if len(p.BuyHigherOrders()) <= index {
		p.setBuyHigherOrders(p.loadHigherOrders(p, p.BuyHigherOrders(), index))
	}
	if len(p.BuyHigherOrders())-1 < index {
		return nil
	}
	order := p.BuyHigherOrders()[index]
	if !p.isSorted() {
		return order.reverse()
	}

	return order
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
		p.setSellLowerOrders(p.loadLowerOrders(p, p.SellLowerOrders(), index))
	}
	if len(p.SellLowerOrders())-1 < index {
		return nil
	}

	order := p.SellLowerOrders()[index]
	if !p.isSorted() {
		return order.reverse()
	}

	return order
}

func (p *Pair) OrderSellLowerLast() (limit *Limit, index int) {
	for order := p.OrderSellLowerByIndex(index); order != nil; order = p.OrderSellLowerByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}
