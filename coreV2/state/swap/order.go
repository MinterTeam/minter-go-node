package swap

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math"
	"math/big"
)

type Order struct {
	*Limit
	isDrop bool
}

type Limit struct {
	reversed bool
	Coin0    *big.Int
	Coin1    *big.Int

	rate *big.Float
	id   uint32
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
	if !l.reversed {
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
		reversed: !l.reversed,
		Coin0:    l.Coin1,
		Coin1:    l.Coin0,
		rate:     nil,
		id:       l.id,
	}
}

func (l *Limit) sort() *Limit {
	if !l.reversed {
		return l
	}
	return &Limit{
		reversed: false,
		Coin0:    l.Coin1,
		Coin1:    l.Coin0,
		rate:     nil,
		id:       l.id,
	}
}

func (p *Pair) MarkDirtyOrders(order *Order) {
	p.markDirtyOrders()
	p.dirtyOrders = append(p.dirtyOrders, order)
	return
}

func (p *Pair) SetHigherOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := big.NewFloat(0).SetPrec(precision).Quo(big.NewFloat(0).SetInt(amountSell), big.NewFloat(0).SetInt(amountBuy))
	var index int
	for i, limit := range p.ordersHigher {
		if rate.Cmp(limit.Rate()) != -1 {
			index = i + 1
			continue
		}
		break
	}

	limit = &Limit{
		reversed: !p.isSorted(),
		Coin0:    amountSell,
		Coin1:    amountBuy,
		rate:     rate,
		id:       p.getLastTotalOrderID(),
	}
	defer p.MarkDirtyOrders(&Order{
		Limit:  limit,
		isDrop: false,
	})

	if index == 0 {
		p.ordersHigher = append([]*Limit{limit}, p.ordersHigher...)
		return
	}

	if index == len(p.ordersHigher) {
		p.ordersHigher = append(p.ordersHigher, limit)
		return
	}

	p.ordersHigher = append(p.ordersHigher[:index], append([]*Limit{limit}, p.ordersHigher[index:]...)...)
	return
}

func (p *Pair) SetLowerOrder(amountSell, amountBuy *big.Int) (limit *Limit) {
	rate := big.NewFloat(0).SetPrec(precision).Quo(big.NewFloat(0).SetInt(amountSell), big.NewFloat(0).SetInt(amountBuy))
	var index int
	for i, limit := range p.ordersLower {
		if rate.Cmp(limit.Rate()) != -1 {
			index = i + 1
			continue
		}
		break
	}

	limit = &Limit{
		reversed: p.isSorted(),
		Coin0:    amountSell,
		Coin1:    amountBuy,
		rate:     rate,
		id:       p.getLastTotalOrderID(),
	}
	defer p.MarkDirtyOrders(&Order{
		Limit:  limit,
		isDrop: false,
	})

	if index == 0 {
		p.ordersLower = append([]*Limit{limit}, p.ordersLower...)
		return
	}

	if index == len(p.ordersLower) {
		p.ordersLower = append(p.ordersLower, limit)
		return
	}

	p.ordersLower = append(p.ordersLower[:index], append([]*Limit{limit}, p.ordersLower[index:]...)...)
	return
}

func (p *Pair) SetOrder(amount0, amount1 *big.Int, sell bool) (id uint32) {
	var r *big.Float
	var order *Limit
	if p.isSorted() {
		if sell {
			r = calcRate(amount0, amount1)
			if p.Rate().Cmp(r) != -1 {
				order = p.SetLowerOrder(amount0, amount1)
			} else {
				order = p.SetHigherOrder(amount0, amount1)
			}
		} else {
			r = calcRate(amount1, amount0)
			if p.Rate().Cmp(r) != -1 {
				order = p.SetLowerOrder(amount1, amount0)
			} else {
				order = p.SetHigherOrder(amount1, amount0)
			}
		}
	} else {
		if !sell {
			r = calcRate(amount0, amount1)
			if p.Rate().Cmp(r) != -1 {
				order = p.SetLowerOrder(amount0, amount1)
			} else {
				order = p.SetHigherOrder(amount0, amount1)
			}
		} else {
			r = calcRate(amount1, amount0)
			if p.Rate().Cmp(r) != -1 {
				order = p.SetLowerOrder(amount1, amount0)
			} else {
				order = p.SetHigherOrder(amount1, amount0)
			}
		}
	}
	return order.id
}

// loadHigherOrders loads only needed orders for pair, not all
func (s *Swap) loadHigherOrders(pair *Pair, limit int) { // todo: add mutex
	var startKey []byte
	endKey := append(append([]byte{mainPrefix}, pair.pairKey.pathOrders()...), byte(255))

	var slice []*Limit
	if pair.pairKey.isSorted() {
		slice = pair.ordersHigher
	} else {
		slice = pair.ordersLower
	}

	if len(slice) > 0 {
		var l = slice[len(slice)-1]
		startKey = ratePath(pair.pairKey, l.SortRate(), l.id+1)
	} else {
		startKey = ratePath(pair.pairKey, pair.SortRate(), 0)
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

	if pair.pairKey.isSorted() {
		pair.ordersHigher = slice
	} else {
		pair.ordersLower = slice
	}
}

func (s *Swap) loadLowerOrders(pair *Pair, limit int) {
	startKey := append(append([]byte{mainPrefix}, pair.pairKey.pathOrders()...), byte(0))
	var endKey []byte

	var slice []*Limit
	if pair.pairKey.isSorted() {
		slice = pair.ordersLower
	} else {
		slice = pair.ordersHigher
	}

	if len(slice) > 0 {
		var l = slice[len(slice)-1]
		endKey = ratePath(pair.pairKey, l.SortRate(), l.id-1)
	} else {
		endKey = ratePath(pair.pairKey, pair.SortRate(), math.MaxInt32)
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
		if !pair.pairKey.isSorted() {
			order = order.reverse()
		}
		slice = append(slice, order)
		i++
		return false
	})

	if pair.pairKey.isSorted() {
		pair.ordersLower = slice
	} else {
		pair.ordersHigher = slice
	}
}

func (p *Pair) OrderHigherByIndex(index int) *Limit {
	if len(p.ordersHigher) <= index { // todo
		p.loadHigherOrders(p, index)
	}
	if len(p.ordersHigher)-1 < index {
		return nil
	}
	return p.ordersHigher[index]
}

func (p *Pair) OrderHigherLast() (limit *Limit, index int) {
	for order := p.OrderHigherByIndex(index); order != nil; order = p.OrderHigherByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}

func (p *Pair) OrderLowerByIndex(index int) *Limit {
	if len(p.ordersLower) <= index { // todo
		p.loadLowerOrders(p, index)
	}
	if len(p.ordersLower)-1 < index {
		return nil
	}
	return p.ordersLower[index]
}

func (p *Pair) OrderLowerLast() (limit *Limit, index int) {
	for order := p.OrderLowerByIndex(index); order != nil; order = p.OrderLowerByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}
