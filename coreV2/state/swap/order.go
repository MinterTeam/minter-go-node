package swap

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
)

const commissionOrder = 2

func (s *Swap) PairSellWithOrders(coin0, coin1 types.CoinID, amount0In, minAmount1Out *big.Int) (*big.Int, *big.Int, uint32, *ChangeDetailsWithOrders, map[types.Address]*big.Int) {
	pair := s.Pair(coin0, coin1)
	amount1Out, owners, details := pair.SellWithOrders(amount0In)
	if amount1Out.Cmp(minAmount1Out) == -1 {
		panic(fmt.Sprintf("calculatedAmount1Out %s less minAmount1Out %s", amount1Out, minAmount1Out))
	}

	for _, b := range owners {
		s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(b))
	}
	s.bus.Checker().AddCoin(coin0, amount0In)
	s.bus.Checker().AddCoin(coin1, big.NewInt(0).Neg(amount1Out))
	return amount0In, amount1Out, pair.GetID(), details, owners
}

func (s *Swap) PairBuyWithOrders(coin0, coin1 types.CoinID, maxAmount0In, amount1Out *big.Int) (*big.Int, *big.Int, uint32, *ChangeDetailsWithOrders, map[types.Address]*big.Int) {
	pair := s.Pair(coin0, coin1)
	amount0In, owners, details := pair.BuyWithOrders(amount1Out)
	if amount1Out.Cmp(maxAmount0In) == 1 {
		panic(fmt.Sprintf("calculatedAmount1Out %s less minAmount1Out %s", amount1Out, maxAmount0In))
	}

	for _, b := range owners {
		s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(b))
	}
	s.bus.Checker().AddCoin(coin0, amount0In)
	s.bus.Checker().AddCoin(coin1, big.NewInt(0).Neg(amount1Out))
	return amount0In, amount1Out, pair.GetID(), details, owners
}

type ChangeDetailsWithOrders struct {
	AmountIn            *big.Int
	AmountOut           *big.Int
	CommissionAmountIn  *big.Int
	CommissionAmountOut *big.Int
	Orders              []*Limit
}

func (c *ChangeDetailsWithOrders) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		AmountIn            string   `json:"amount_in"`
		AmountOut           string   `json:"amount_out"`
		CommissionAmountIn  string   `json:"commission_amount_in"`
		CommissionAmountOut string   `json:"commission_amount_out"`
		Orders              []*Limit `json:"orders"`
	}{
		AmountIn:            c.AmountIn.String(),
		AmountOut:           c.AmountOut.String(),
		CommissionAmountIn:  c.CommissionAmountIn.String(),
		CommissionAmountOut: c.CommissionAmountOut.String(),
		Orders:              c.Orders,
	})
}

func (p *Pair) SellWithOrders(amount0In *big.Int) (amount1Out *big.Int, owners map[types.Address]*big.Int, c *ChangeDetailsWithOrders) {
	if amount0In == nil || amount0In.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}
	amount1Out, orders := p.calculateBuyForSellWithOrders(amount0In)
	if amount1Out == nil || amount1Out.Sign() != 1 {
		panic(ErrorInsufficientOutputAmount)
	}

	commission0orders, commission1orders, amount0, amount1, owners := CalcDiffPool(amount0In, amount1Out, orders)

	log.Println("uS", commission0orders, commission1orders)

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		log.Println("a", amount0, amount1)
		p.update(amount0, big.NewInt(0).Neg(amount1))
	}

	p.update(commission0orders, commission1orders)

	p.updateOrders(orders)

	p.orderSellByIndex(0) // update list

	return amount1Out, owners, &ChangeDetailsWithOrders{
		AmountIn:            amount0,
		AmountOut:           amount1,
		CommissionAmountIn:  commission0orders,
		CommissionAmountOut: commission1orders,
		Orders:              orders,
	}
}

func CalcDiffPool(amount0In, amount1Out *big.Int, orders []*Limit) (*big.Int, *big.Int, *big.Int, *big.Int, map[types.Address]*big.Int) {
	owners := map[types.Address]*big.Int{}

	amount0orders, amount1orders := big.NewInt(0), big.NewInt(0)
	commission0orders, commission1orders := big.NewInt(0), big.NewInt(0)
	for _, order := range orders {
		amount0orders.Add(amount0orders, order.WantBuy)
		amount1orders.Add(amount1orders, order.WantSell)

		cB := calcCommission999(order.WantBuy)
		cS := calcCommission999(order.WantSell)

		commission0orders.Add(commission0orders, cB)
		commission1orders.Add(commission1orders, cS)

		if owners[order.Owner] == nil {
			owners[order.Owner] = big.NewInt(0)
		}
		owners[order.Owner].Add(owners[order.Owner], big.NewInt(0).Sub(order.WantBuy, cB))
	}

	amount1orders.Sub(amount1orders, commission1orders)

	amount0 := big.NewInt(0).Sub(amount0In, amount0orders)
	amount1 := big.NewInt(0).Sub(amount1Out, amount1orders)

	return commission0orders, commission1orders, amount0, amount1, owners
}

func (p *Pair) BuyWithOrders(amount1Out *big.Int) (amount0In *big.Int, owners map[types.Address]*big.Int, c *ChangeDetailsWithOrders) { // todo: add mutex
	if amount1Out == nil || amount1Out.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}
	amount0In, orders := p.calculateSellForBuyWithOrders(amount1Out)
	if amount0In == nil || amount0In.Sign() != 1 {
		panic(ErrorInsufficientOutputAmount)
	}

	commission0orders, commission1orders, amount0, amount1, owners := CalcDiffPool(amount0In, amount1Out, orders)

	log.Println("uB", commission0orders, commission1orders)

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		log.Println("a", amount0, amount1)
		p.update(amount0, big.NewInt(0).Neg(amount1))
	}

	p.update(commission0orders, commission1orders)

	p.updateOrders(orders)

	return amount0In, owners, &ChangeDetailsWithOrders{
		AmountIn:            amount0,
		AmountOut:           amount1,
		CommissionAmountIn:  commission0orders,
		CommissionAmountOut: commission1orders,
		Orders:              orders,
	}
}

func (p *Pair) updateOrders(orders []*Limit) {
	var editedOrders []*Limit
	for _, order := range orders {
		editedOrders = append(editedOrders, p.updateSellOrder(order.id, order.WantBuy, order.WantSell))
	} // todo: FIXME need tests
	//for _, editedOrder := range editedOrders {
	//	p.resortSellOrderList(0, editedOrder)
	//}

}

func (p *Pair) updateSellOrder(id uint32, amount0, amount1 *big.Int) *Limit {
	limit := p.getOrder(id)

	newLimit := limit.sort()
	newLimit.OldSortPrice()

	fmt.Println(limit.WantBuy, amount0, limit.WantSell, amount1)
	limit.WantBuy.Sub(limit.WantBuy, amount0)
	limit.WantSell.Sub(limit.WantSell, amount1)

	fmt.Println(limit.WantBuy, limit.WantSell)

	p.MarkDirtyOrders(newLimit) // need before resort

	return newLimit
}

func (p *Pair) resortSellOrderList(i int, limit *Limit) {
	if limit.isEmpty() {
		if !(limit.WantBuy.Sign() == 0 && limit.WantSell.Sign() == 0) {
			panic(fmt.Sprintf("zero value of %#v", limit))
		}
		p.unsetOrderSellByIndex(i)
		return
	}

	if limit.CmpOldRate() == 0 {
		return
	}

	log.Println("set new order position", p.isSorted(), // true
		limit.CmpOldRate(), limit.SortPrice().Text('f', 18), // 0.333333333333333315
		limit.OldSortPrice().Text('f', 18), // 0.333333333333333343
		p.Price().Text('f', 18),            // 0.333333333333333259
	)

	// cmp := p.DirectionSortPrice()

	// if !(limit.SortPrice().Cmp(p.SortPrice()) == cmp) {
	// 	p.unsetOrderSellByIndex(i)
	// 	return
	// }

	p.unsetOrderSellByIndex(i)

	// if limit.CmpOldRate() != cmp {
	// 	loadedLen := len(p.SellLowerOrders())
	// 	newIndex := p.setSellOrder(limit)
	// 	if newIndex == loadedLen {
	// 		p.unsetOrderSellByIndex(newIndex)
	//p.addOrder(limit)
	// }
	// }
}

func (l *Limit) isEmpty() bool {
	return l == nil || l.WantBuy.Sign() == 0 || l.WantSell.Sign() == 0
}

func (l *Limit) isKeepRate() bool {
	if l == nil {
		return false
	}
	fmt.Println("is keep", l.oldSortPrice, l.SortPrice())
	return l.CmpOldRate() == 0
}

func (l *Limit) CmpOldRate() int {
	return l.SortPrice().Cmp(l.OldSortPrice())
}

func (p *Pair) CalculateBuyForSellWithOrders(amount0In *big.Int) (amount1Out *big.Int) {
	amount1Out, _ = p.calculateBuyForSellWithOrders(amount0In)
	return amount1Out
}

func (p *Pair) calculateBuyForSellWithOrders(amount0In *big.Int) (amountOut *big.Int, orders []*Limit) {
	amountOut = big.NewInt(0)
	amountIn := big.NewInt(0).Set(amount0In)
	var pair EditableChecker = p
	for i := 0; true; i++ {
		if amountIn.Sign() == 0 {
			return amountOut, orders
		}

		limit := p.orderSellByIndex(i)
		if limit == nil {
			break
		}

		price := limit.Price()
		if price.Cmp(pair.Price()) == -1 {
			reserve0diff, reserve1diff := pair.CalculateAddAmountsForPrice(price)
			if reserve0diff != nil && reserve1diff != nil {
				if amountIn.Cmp(reserve0diff) != 1 {
					break
				}

				amountIn.Sub(amountIn, reserve0diff)
				amountOut.Add(amountOut, reserve1diff)

				if err := pair.CheckSwap(reserve0diff, reserve1diff); err != nil {
					panic(err) // todo: for test
				}
				pair = pair.AddLastSwapStep(reserve0diff, reserve1diff)

				log.Println("rS", reserve0diff, reserve1diff)

			}
		}

		// хотим продать 9009
		// проверяем есть ли столько на продажу
		log.Println("i", amountIn)
		rest := big.NewInt(0).Sub(amountIn, limit.WantBuy)
		if rest.Sign() != 1 {
			// 9009
			amount0 := big.NewInt(0).Set(amountIn)
			// считаем сколько сможем купить -- 3003
			amount1, acc := big.NewFloat(0).Mul(price, big.NewFloat(0).SetInt(amount0)).Int(nil)
			if acc != big.Exact {
				log.Println("acc", acc)
				// if acc == big.Below { // todo
				// 	amount1.Add(amount1,big.NewInt(1))
				// }
			}
			log.Println("m", amount1)

			orders = append(orders, &Limit{
				IsBuy:        limit.IsBuy,
				PairKey:      p.PairKey,
				WantBuy:      amount0, // 9009, 9 заберем в пул
				WantSell:     amount1, // 3003, 3 пул
				Owner:        limit.Owner,
				Height:       limit.Height,
				oldSortPrice: limit.SortPrice(),
				id:           limit.id,
				RWMutex:      new(sync.RWMutex),
			})

			comB := calcCommission999(amount1)
			log.Println("n", comB)
			amountOut.Add(amountOut, big.NewInt(0).Sub(amount1, comB))
			return amountOut, orders
		}

		orders = append(orders, &Limit{
			IsBuy:        limit.IsBuy,
			WantBuy:      big.NewInt(0).Set(limit.WantBuy),
			WantSell:     big.NewInt(0).Set(limit.WantSell),
			Owner:        limit.Owner,
			Height:       limit.Height,
			PairKey:      limit.PairKey,
			oldSortPrice: limit.SortPrice(),
			id:           limit.id,
			RWMutex:      limit.RWMutex,
		})

		comS := calcCommission999(limit.WantBuy)
		comB := calcCommission999(limit.WantSell)
		log.Println("ccc1", comS, big.NewInt(0).Neg(comB))
		pair = pair.AddLastSwapStep(comS, big.NewInt(0).Neg(comB))
		amountOut.Add(amountOut, big.NewInt(0).Sub(limit.WantSell, comB))

		amountIn = rest
	}

	amount1diff := pair.CalculateBuyForSell(amountIn)
	if amount1diff != nil {
		if err := pair.CheckSwap(amountIn, amount1diff); err != nil {
			panic(err) // todo: for test
		}
		amountOut.Add(amountOut, amount1diff)
	}
	return amountOut, orders
}

func calcCommission001(amount0 *big.Int) *big.Int {
	mul := big.NewInt(0).Mul(amount0, big.NewInt(commissionOrder/2))
	quo := big.NewInt(0).Quo(mul, big.NewInt(1000))
	remainder := big.NewInt(0)
	if big.NewInt(0).Rem(mul, big.NewInt(1000)).Sign() == 1 {
		remainder = big.NewInt(1)
	}
	quo.Add(quo, remainder)
	return quo
}

func calcCommission999(amount1 *big.Int) *big.Int {
	quo := big.NewInt(0).Quo(amount1, big.NewInt(1000+commissionOrder/2))
	remainder := big.NewInt(0)
	if big.NewInt(0).Rem(amount1, big.NewInt(1000+commissionOrder/2)).Sign() == 1 {
		remainder = big.NewInt(1)
	}
	quo.Add(quo, remainder)
	return quo
}

func (p *Pair) CalculateAddAmountsForPrice(price *big.Float) (amount0In, amount1Out *big.Int) {
	if price.Cmp(p.Price()) == 1 {
		fmt.Println("price cur and first ord", price, p.Price())
		return nil, nil
	}
	return p.calculateAddAmountsForPrice(price)
}

// calculateAddAmountsForPrice returns a0 and a1 to reach the price
//		{ (r0 + 0.998a0) * (r1 - a1) = r0 * r1
//	   {
//		{ (r0 + a0) / (r1 - a1) = price
//
func (p *Pair) calculateAddAmountsForPrice(price *big.Float) (amount0 *big.Int, amount1 *big.Int) {
	reserve0, reserve1 := p.Reserves()
	r0 := big.NewFloat(0).SetInt(reserve0)
	r1 := big.NewFloat(0).SetInt(reserve1)
	k := big.NewFloat(0).Mul(r0, r1)
	r0Qrt := big.NewFloat(0).Mul(r0, r0)
	b := big.NewFloat(0).Mul(big.NewFloat((2000.0-commission)/2), r0)
	kMulPrice := big.NewFloat(0).Mul(k, big.NewFloat(0).Quo(big.NewFloat(1), price))
	r0QrtSubKMulPrice := big.NewFloat(0).Sub(r0Qrt, kMulPrice)
	d := big.NewFloat(0).Sub(big.NewFloat(0).Mul(big.NewFloat((2000-commission)*(2000-commission)/4), r0Qrt), big.NewFloat(0).Mul(big.NewFloat(2000*(1000-commission)/2), r0QrtSubKMulPrice))
	x1 := big.NewFloat(0).Quo(big.NewFloat(0).Add(big.NewFloat(0).Neg(b), big.NewFloat(0).Sqrt(d)), big.NewFloat(1000-commission))
	amount0, acc := x1.Int(nil)
	log.Println("acc0", acc)

	if amount0 == nil || amount0.Sign() != 1 {
		return nil, nil
	}

	return amount0, p.CalculateBuyForSell(amount0)
}

func (p *Pair) CalculateSellForBuyWithOrders(amount1Out *big.Int) (amount0In *big.Int) {
	amount0In, _ = p.calculateSellForBuyWithOrders(amount1Out)
	return amount0In
}

func (p *Pair) calculateSellForBuyWithOrders(amount1Out *big.Int) (amountIn *big.Int, orders []*Limit) {
	amountIn = big.NewInt(0)
	amountOut := big.NewInt(0).Set(amount1Out)
	var pair EditableChecker = p
	for i := 0; true; i++ {
		if amountOut.Sign() == 0 {
			return amountIn, orders
		}

		limit := p.orderSellByIndex(i)
		if limit == nil {
			break
		}
		log.Println("ow", limit.Owner.String())

		price := limit.Price()
		if price.Cmp(pair.Price()) == -1 {
			reserve0diff, reserve1diff := pair.CalculateAddAmountsForPrice(price)
			log.Println(reserve1diff, "ooo")
			if reserve1diff != nil && reserve0diff != nil {
				if amountOut.Cmp(reserve1diff) != 1 {
					break
				}
				amountOut.Sub(amountOut, reserve1diff)
				amountIn.Add(amountIn, reserve0diff)

				if err := pair.CheckSwap(reserve0diff, reserve1diff); err != nil {
					panic(err) // todo: for test
				}
				pair = pair.AddLastSwapStep(reserve0diff, reserve1diff)

				log.Println("rB", reserve0diff, reserve1diff)
			}
		}

		// хочу купить amountOut = 3000
		// тк мы 0.1% положим в пул, то надо купить 3003
		// проверим что в пуле есть X * 0.999 == 3000
		// на продажу есть 5000
		// что бы в пул пошел 0.1%, мне надо купить 3003 из которых 3 положить в пул
		comB := calcCommission999(limit.WantSell)
		log.Println("amountOut", amountOut)
		log.Println("comB", comB)
		rest := big.NewInt(0).Sub(amountOut, big.NewInt(0).Sub(limit.WantSell, comB))
		if rest.Sign() != 1 {
			log.Println("order")
			amount1 := big.NewInt(0).Add(amountOut, calcCommission001(amountOut))
			log.Println("amount1", amount1)
			// считаем сколько монет надо продать что бы купить 3003
			amount0, acc := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(amount1), price).Int(nil)
			if acc != big.Exact {
				log.Println("acc", acc) // todo
			}
			log.Println("amount0", amount0)

			orders = append(orders, &Limit{
				IsBuy:        limit.IsBuy,
				PairKey:      p.PairKey,
				WantBuy:      big.NewInt(0).Set(amount0), // и того продам по ордеру 9009, из них 9000 продавцу и 9 в пул
				WantSell:     amount1,                    // 3003, позже вычтем 3 и положим в пул
				Owner:        limit.Owner,
				Height:       limit.Height,
				oldSortPrice: limit.SortPrice(),
				id:           limit.id,
				RWMutex:      new(sync.RWMutex),
			})

			amountIn.Add(amountIn, amount0)
			return amountIn, orders
		}

		log.Println("order more")
		orders = append(orders, &Limit{
			IsBuy:        limit.IsBuy,
			WantBuy:      big.NewInt(0).Set(limit.WantBuy),
			WantSell:     big.NewInt(0).Set(limit.WantSell),
			Owner:        limit.Owner,
			Height:       limit.Height,
			PairKey:      limit.PairKey,
			oldSortPrice: limit.SortPrice(),
			id:           limit.id,
			RWMutex:      limit.RWMutex,
		})

		comS := calcCommission999(limit.WantBuy)

		pair = pair.AddLastSwapStep(comS, big.NewInt(0).Neg(comB))
		amountOut = rest

		amountIn.Add(amountIn, limit.WantBuy)
		// amountIn.Add(amountIn, comS)
	}

	amount0diff := pair.CalculateSellForBuy(amountOut)
	if amount0diff != nil {
		if err := pair.CheckSwap(amount0diff, amountOut); err != nil {
			panic(err) // todo: for test
		}
		amountIn.Add(amountIn, amount0diff)
	}
	return amountIn, orders
}

func CalcPriceSell(sell, buy *big.Int) *big.Float {
	return new(big.Float).SetPrec(precision).Quo(
		big.NewFloat(0).SetInt(buy),
		big.NewFloat(0).SetInt(sell),
	)
}

type Limit struct {
	WantBuy  *big.Int
	WantSell *big.Int
	Owner    types.Address
	IsBuy    bool
	Height   uint64

	PairKey
	oldSortPrice *big.Float
	id           uint32

	*sync.RWMutex
}

func (l *Limit) ID() uint32 {
	if l == nil {
		return 0 // todo: test
	}

	return l.id
}

func (l *Limit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		WantBuy  string `json:"buy"`
		WantSell string `json:"sell"`
		Owner    string `json:"seller"`
		ID       uint32 `json:"id"`
	}{
		WantBuy:  l.WantBuy.String(),
		WantSell: l.WantSell.String(),
		Owner:    l.Owner.String(),
		ID:       l.id,
	})
}

type limits struct {
	ids []uint32
}

type orderList struct {
	mu   sync.RWMutex
	list map[uint32]*Limit // list sorted dirties Order
}

type orderDirties struct {
	mu   sync.RWMutex
	list map[uint32]struct{}
}

const (
	precision = 54 // supported precision
)

func (l *Limit) Price() *big.Float {
	if l.isEmpty() {
		return big.NewFloat(0)
	}
	l.RLock()
	defer l.RUnlock()
	return CalcPriceSell(l.WantBuy, l.WantSell)
}

func (p *Pair) Price() *big.Float {
	return p.pairData.Price()
}

func (p *Pair) getOrder(id uint32) *Limit {
	p.orders.mu.Lock() // todo: test edition
	defer p.orders.mu.Unlock()

	l, ok := p.orders.list[id]
	if ok {
		if p.isSorted() {
			return l
		}

		return l.Reverse() // todo: need reverse?
	}

	l = p.loadOrder(id)
	if l == nil {
		return nil
	}

	p.orders.list[id] = l

	if p.isSorted() {
		return l
	}

	return l.Reverse() // todo: need reverse?
}

func (p *Pair) setOrder(l *Limit) {
	p.orders.mu.Lock()
	defer p.orders.mu.Unlock()
	p.orders.list[l.id] = l
}

func (p *Pair) SortPrice() *big.Float {
	if p.isSorted() {
		return p.pairData.Price()
	}
	return p.pairData.reverse().Price()
}

func (p *Pair) DirectionSortPrice() int {
	if !p.isSorted() {
		return 1
	}
	return -1
}

func (l *Limit) SortPrice() *big.Float {
	if l.isSorted() {
		return l.Price()
	}
	return l.Reverse().Price()
}

func (l *Limit) OldSortPrice() *big.Float {
	if l.oldSortPrice == nil {
		l.oldSortPrice = new(big.Float).SetPrec(precision).Set(l.SortPrice())
	}
	return l.oldSortPrice
}

func (l *Limit) isSell() bool {
	return !l.IsBuy
}

// ReCalcOldSortPrice saves before change, need for update on disk
func (l *Limit) ReCalcOldSortPrice() *big.Float {
	l.oldSortPrice.Set(l.SortPrice())
	return l.oldSortPrice
}

func (l *Limit) Reverse() *Limit {
	l.RLock()
	defer l.RUnlock()
	return &Limit{
		PairKey:      l.PairKey.reverse(),
		IsBuy:        !l.IsBuy,
		WantBuy:      l.WantSell,
		WantSell:     l.WantBuy,
		Owner:        l.Owner,
		Height:       l.Height,
		oldSortPrice: l.oldSortPrice,
		id:           l.id,
		RWMutex:      l.RWMutex,
	}
}

func (l *Limit) sort() *Limit {
	if l.isSorted() {
		return l
	}

	return l.Reverse()
}

func (l *Limit) isSorted() bool {
	return l.PairKey.isSorted()
}

func (l *Limit) clone() *Limit {
	return &Limit{
		PairKey:      l.PairKey,
		IsBuy:        l.IsBuy,
		WantBuy:      big.NewInt(0).Set(l.WantBuy),
		WantSell:     big.NewInt(0).Set(l.WantSell),
		Owner:        l.Owner,
		Height:       l.Height,
		oldSortPrice: big.NewFloat(0).Set(l.oldSortPrice),
		id:           l.id,
		RWMutex:      &sync.RWMutex{},
	}
}

func (p *Pair) MarkDirtyOrders(order *Limit) {
	p.markDirtyOrders()

	if order.isEmpty() {
		p.deletedOrders.mu.Lock()
		p.deletedOrders.list[order.id] = struct{}{}
		p.deletedOrders.mu.Unlock()
	} else if !order.isKeepRate() {
		p.unsortedSellOrderIDs().mu.Lock()
		p.unsortedSellOrderIDs().list[order.id] = struct{}{}
		p.unsortedSellOrderIDs().mu.Unlock()
	}

	p.dirtyOrders.mu.Lock()
	p.dirtyOrders.list[order.id] = struct{}{}
	p.dirtyOrders.mu.Unlock()
	return
}

func (p *Pair) setSellOrder(new *Limit) (index int) {
	cmp := p.DirectionSortPrice()

	//p.setOrder(new)
	p.orderSellByIndex(0)
	orders := p.SellOrderIDs()
	for i, limitID := range orders {
		limit := p.getOrder(limitID)
		if !(new.SortPrice().Cmp(limit.SortPrice()) != cmp) {
			index = i + 1
			continue
		}
		break
	}

	if index == 0 {
		p.setSellOrders(append([]uint32{new.id}, orders...))
		return
	}

	if index == len(orders) {
		p.setSellOrders(append(orders, new.id))
		return
	}

	p.setSellOrders(append(orders[:index], append([]uint32{new.id}, orders[index:]...)...))
	return
}

func (p *Pair) SellOrderIDs() []uint32 {
	if p.isSorted() {
		return p.sellOrders.ids
	}
	return p.buyOrders.ids
}

func (p *Pair) BuyOrderIDs() []uint32 {
	if p.isSorted() {
		return p.buyOrders.ids
	}
	return p.sellOrders.ids
}

func (p *Pair) isUnsortedSellOrder(id uint32) bool {
	ds := p.unsortedSellOrderIDs()
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	_, ok := ds.list[id]
	return ok
}
func (p *Pair) isUnsortedBuyOrder(id uint32) bool {
	ds := p.unsortedBuyOrderIDs()
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	_, ok := ds.list[id]
	return ok
}
func (p *Pair) unsortedSellOrderIDs() *orderDirties {
	if p.isSorted() {
		return p.unsortedDirtySellOrders
	}
	return p.unsortedDirtyBuyOrders
}
func (p *Pair) unsortedBuyOrderIDs() *orderDirties {
	if p.isSorted() {
		return p.unsortedDirtyBuyOrders
	}
	return p.unsortedDirtySellOrders
}

func (p *Pair) loadedSellOrderIDs() []uint32 {
	if p.isSorted() {
		return p.loadedSellOrders.ids
	}
	return p.loadedBuyOrders.ids
}

func (p *Pair) loadedBuyOrderIDs() []uint32 {
	if p.isSorted() {
		return p.loadedBuyOrders.ids
	}
	return p.loadedBuyOrders.ids
}

func (p *Pair) setSellOrders(orders []uint32) {
	if p.isSorted() {
		p.sellOrders.ids = orders
		return
	}
	p.buyOrders.ids = orders
	return
}
func (p *Pair) setLoadedSellOrders(orders []uint32) {
	if p.isSorted() {
		p.loadedSellOrders.ids = orders
		return
	}
	p.loadedBuyOrders.ids = orders
	return
}
func (p *Pair) setLoadedBuyOrders(orders []uint32) {
	if p.isSorted() {
		p.loadedBuyOrders.ids = orders
		return
	}
	p.loadedSellOrders.ids = orders
	return
}

func (p *Pair) setBuyOrders(orders []uint32) {
	if p.isSorted() {
		p.buyOrders.ids = orders
		return
	}
	p.sellOrders.ids = orders
	return
}

func (s *Swap) PairAddOrder(coinWantBuy, coinWantSell types.CoinID, wantBuyAmount, wantSellAmount *big.Int, sender types.Address, block uint64) (uint32, uint32) {
	pair := s.Pair(coinWantBuy, coinWantSell)
	order := pair.AddOrder(wantBuyAmount, wantSellAmount, sender, block)

	s.bus.Checker().AddCoin(coinWantSell, wantSellAmount)

	return order.id, pair.GetID()
}

func (s *Swap) PairRemoveLimitOrder(id uint32) (types.CoinID, *big.Int) {
	order := s.GetOrder(id)
	if order == nil {
		panic("order not exist")
	}

	if !order.isSell() {
		order = order.Reverse()
	}

	returnVolume := big.NewInt(0).Set(order.WantSell)

	s.bus.Checker().AddCoin(order.Coin1, big.NewInt(0).Neg(returnVolume))

	pair := s.Pair(order.Coin0, order.Coin1)
	pair.updateOrders([]*Limit{order})

	return order.Coin1, returnVolume
}

func (s *Swap) PairAddOrderWithID(coinWantBuy, coinWantSell types.CoinID, wantBuyAmount, wantSellAmount *big.Int, sender types.Address, id uint32, height uint64) (uint32, uint32) {
	pair := s.Pair(coinWantBuy, coinWantSell)
	order := pair.AddOrderWithID(wantBuyAmount, wantSellAmount, sender, id, height)

	s.bus.Checker().AddCoin(coinWantSell, wantSellAmount)

	return order.id, pair.GetID()
}

func (p *Pair) GetOrder(id uint32) *Limit {
	return p.getOrder(id)
}

func (p *Pair) AddOrder(wantBuyAmount0, wantSellAmount1 *big.Int, sender types.Address, block uint64) (order *Limit) {
	order = &Limit{
		PairKey:      p.PairKey,
		IsBuy:        false,
		WantBuy:      wantBuyAmount0,
		WantSell:     wantSellAmount1,
		id:           p.getLastTotalOrderID(),
		oldSortPrice: new(big.Float).SetPrec(precision),
		Owner:        sender,
		RWMutex:      new(sync.RWMutex),
		Height:       block,
	}
	sort := order.sort()
	p.MarkDirtyOrders(sort)

	p.lockOrders.Lock() // todo: tests
	defer p.lockOrders.Unlock()

	p.setOrder(sort)
	p.orderSellByIndex(0)
	return order
}

func (p *Pair) AddOrderWithID(wantBuyAmount0, wantSellAmount1 *big.Int, sender types.Address, id uint32, height uint64) (order *Limit) {
	order = &Limit{
		PairKey:      p.PairKey,
		IsBuy:        false,
		WantBuy:      wantBuyAmount0,
		WantSell:     wantSellAmount1,
		id:           id,
		oldSortPrice: big.NewFloat(0),
		Owner:        sender,
		Height:       height,
		RWMutex:      new(sync.RWMutex),
	}
	sort := order.sort()
	p.MarkDirtyOrders(sort)

	p.lockOrders.Lock() // todo: tests
	defer p.lockOrders.Unlock()

	p.setOrder(order)
	p.orderSellByIndex(0)
	return order
}

func (p *Pair) addOrder(limit *Limit) {
	if p.Price().Cmp(limit.Price()) == -1 {
		log.Println("Higher")
	}

	firstSavedOrder := p.orderSellByIndex(0)
	if firstSavedOrder != nil {
		return
	}

	// todo: remove
	orders, count := p.updateDirtyOrders([]uint32{}, true)
	log.Println("addOrder: del", count)
	p.setSellOrders(orders)
	return
}

func (p *Pair) loadAllOrders(immutableTree *iavl.ImmutableTree) (orders []*Limit) {
	const countFirstBytes = 10

	startKey := append(append([]byte{mainPrefix}, p.pathOrders()...), byte(0), byte(0))
	endKey := append(append([]byte{mainPrefix}, p.pathOrders()...), byte(1), byte(255))

	immutableTree.IterateRange(startKey, endKey, true, func(key []byte, value []byte) bool {
		var isSell = true
		if key[countFirstBytes : countFirstBytes+1][0] == 0 {
			isSell = false
		}

		order := p.getOrder(binary.BigEndian.Uint32(key[len(key)-4:]))
		if !isSell {
			// order = order.reverse() // todo: need test
		}
		orders = append(orders, order)

		return false
	})

	return orders
}

func (s *Swap) loadBuyOrders(pair *Pair, fromOrder *Limit, limit int) []uint32 {
	endKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(0), byte(255))
	var startKey = append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(0), byte(0))

	ids := pair.loadedBuyOrderIDs()

	if fromOrder == nil && len(ids) >= limit {
		return ids[:limit]
	}

	k := 1
	var slice []uint32
	for i, id := range ids {
		if id == fromOrder.ID() {
			if len(ids[i+1:]) < limit {
				slice = append(slice, ids[i+1:]...)
				k += len(ids[i+1:])
				fromOrder = pair.getOrder(ids[len(ids)-1])
				break
			}

			return ids[i+1 : i+limit+1]
		}
	}

	if fromOrder != nil { //  todo .isKeepRate()
		startKey = pricePath(pair.PairKey, fromOrder.OldSortPrice(), fromOrder.id+1, false) // todo: tests OldSortPrice
	}

	s.immutableTree().IterateRange(startKey, endKey, true, func(key []byte, _ []byte) bool {
		if k > limit {
			return true
		}

		id := binary.BigEndian.Uint32(key[len(key)-4:])

		slice = append(slice, id)
		k++
		return false
	})

	pair.setLoadedBuyOrders(append(ids, slice...))
	return slice
}

func (s *Swap) GetOrder(id uint32) *Limit {
	order := s.loadOrder(id)
	if order == nil {
		return nil
	}

	list := s.Pair(order.Coin0, order.Coin1).orders
	list.mu.RLock()
	defer list.mu.RUnlock()

	if o, ok := list.list[id]; ok {
		return o
	}

	list.list[id] = order
	return order
}

func (s *Swap) loadOrder(id uint32) *Limit {
	_, value := s.immutableTree().Get(pathOrder(id))
	if value == nil {
		return nil
	}

	order := &Limit{
		id: id,
		//oldSortPrice: new(big.Float).SetPrec(precision),
		RWMutex: new(sync.RWMutex),
	}
	err := rlp.DecodeBytes(value, order)
	if err != nil {
		panic(err)
	}

	return order
}

func (s *Swap) loadSellOrders(pair *Pair, fromOrder *Limit, limit int) []uint32 {
	startKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(1), byte(0))
	var endKey = append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(1), byte(255))

	ids := pair.loadedSellOrderIDs()

	if fromOrder == nil && len(ids) >= limit {
		return ids[:limit]
	}

	k := 1
	var slice []uint32
	for i, id := range ids {
		if id == fromOrder.ID() {
			if len(ids[i+1:]) < limit {
				slice = append(slice, ids[i+1:]...)
				k += len(ids[i+1:])
				fromOrder = pair.getOrder(ids[len(ids)-1])
				break
			}

			return ids[i+1 : i+limit+1]
		}
	}

	if fromOrder != nil { //  todo .isKeepRate()
		endKey = pricePath(pair.PairKey, fromOrder.OldSortPrice(), fromOrder.id-1, true) // todo: tests OldSortPrice

	}

	s.immutableTree().IterateRange(startKey, endKey, false, func(key []byte, value []byte) bool {
		if k > limit {
			return true
		}

		id := binary.BigEndian.Uint32(key[len(key)-4:])

		slice = append(slice, id)
		k++
		return false
	})

	pair.setLoadedSellOrders(append(ids, slice...))
	return slice
}

func (p *Pair) updateDirtyOrders(list []uint32, lower bool) (orders []uint32, delCount int) {
	unsortedDirtySellOrders := p.unsortedSellOrderIDs()
	unsortedDirtySellOrders.mu.Lock()
	defer unsortedDirtySellOrders.mu.Unlock()

	deletedDirtySellOrders := p.deletedOrders
	deletedDirtySellOrders.mu.Lock()
	defer deletedDirtySellOrders.mu.Unlock()

	for _, orderID := range list {
		if _, ok := deletedDirtySellOrders.list[orderID]; ok {
			delCount++
			continue
		}
		if _, ok := unsortedDirtySellOrders.list[orderID]; ok {
			delCount++
			continue
		}

		orders = append(orders, orderID)
	}

	cmp := p.DirectionSortPrice() // todo: tests
	if !lower {
		cmp *= -1
	}

	var addToEnd = len(p.loadedSellOrderIDs()) == 0

	for orderID := range unsortedDirtySellOrders.list {
		dirty := p.getOrder(orderID)
		var isSet bool
		orders, isSet = p.addToList(orders, dirty, cmp, addToEnd)
		if isSet {
			delCount--
			delete(unsortedDirtySellOrders.list, orderID)
		} else {

		}
	}

	return orders, delCount
}

// Deprecated
func (p *Pair) resortOrders(list []uint32, lower bool) (orders []uint32, countDirties int) {
	for _, orderID := range list {
		if _, ok := p.dirtyOrders.list[orderID]; ok {
			dirtyOrder := p.getOrder(orderID)
			if dirtyOrder.isKeepRate() {
				orders = append(orders, orderID)
			} else {
				countDirties++
			}
			continue
		}
		orders = append(orders, orderID)
	}

	cmp := 1
	if !p.isSorted() && lower || p.isSorted() && !lower {
		cmp *= -1
	}
	for _, dirtyOrderID := range p.getDirtyOrdersList() {
		dirtyOrder := p.getOrder(dirtyOrderID)
		if dirtyOrder.isKeepRate() {
			continue
		} else if dirtyOrder.isEmpty() {
			continue
		} else {
			// if dirtyOrder.SortPrice().Cmp(p.SortPrice()) != p.DirectionSortPrice() { // Deprecated: logic with highest price orders
			// 	continue
			// } else {
			var isSet bool
			orders, isSet = p.addToList(orders, dirtyOrder, cmp, false)
			if isSet {
				countDirties--
			}
			// }
		}
	}

	return orders, countDirties
}

func (p *Pair) addToList(orders []uint32, dirtyOrder *Limit, cmp int, end bool) (list []uint32, includedInInterval bool) {
	var index int
	for i, limitID := range orders {
		if limitID == dirtyOrder.id {
			return orders, true
		}
		limit := p.getOrder(limitID)

		if dirtyOrder.SortPrice().Cmp(limit.SortPrice()) == cmp {
			index = i + 1
			continue
		}

		if dirtyOrder.SortPrice().Cmp(limit.SortPrice()) == 0 && dirtyOrder.id < limit.id { // todo: need tests
			index = i + 1
			break
		}

		break
	}

	if index == 0 {
		return append([]uint32{dirtyOrder.id}, orders...), true
	}

	if index == len(orders) {
		if end {
			return append(orders, dirtyOrder.id), true
		}
		return orders, false
	}

	return append(orders[:index], append([]uint32{dirtyOrder.id}, orders[index:]...)...), true
}

func (p *Pair) OrderIDBuyByIndex(index int) uint32 {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	return p.orderIDBuyByIndex(index)
}

func (p *Pair) orderIDBuyByIndex(index int) uint32 {
	orders := p.BuyOrderIDs()
	var count int
	var deleteCount int
	for firstIterate := true; (firstIterate && len(orders) <= index) || deleteCount != 0; firstIterate = false {
		//orders, deleteCount = p.updateDirtyOrders(nil, p.loadBuyOrders(p, orders, index+count), false)
		count += deleteCount
	}

	p.setBuyOrders(orders)

	if len(orders)-1 < index {
		return 0
	}

	return orders[index]
}

// Deprecated
func (p *Pair) OrderBuyByIndex(index int) *Limit {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	return p.orderBuyByIndex(index)
}

// Deprecated
func (p *Pair) orderBuyByIndex(index int) *Limit {
	//if !p.isSorted() {
	//	return p.orderSellByIndex(index)
	//}

	orders := p.BuyOrderIDs()

	var fromOrder *Limit
	if len(orders) != 0 {
		fromOrder = p.getOrder(orders[0])
		if p.isUnsortedBuyOrder(fromOrder.id) {
			loadedOrders := p.loadBuyOrders(p, fromOrder, index+1)
			resortedOrders, unsets := p.updateDirtyOrders(loadedOrders, false)
			for unsets > 0 {
				id := resortedOrders[len(resortedOrders)-1]
				fromOrder = p.getOrder(id)
				loadedNextOrders := p.loadBuyOrders(p, fromOrder, index+1)
				resortedOrders, unsets = p.updateDirtyOrders(append(orders, loadedNextOrders...), false)
			}
			orders = resortedOrders
		} else {
			if index > len(orders)-1 {
				fromOrder = p.getOrder(orders[len(orders)-1])
				loadedNextOrders := p.loadBuyOrders(p, fromOrder, index+1)
				orders, _ = p.updateDirtyOrders(append(orders, loadedNextOrders...), false)
			} else {
				orders, _ = p.updateDirtyOrders(orders, false)
			}
		}
	} else {
		orders = p.loadBuyOrders(p, nil, index+1)
		orders, _ = p.updateDirtyOrders(orders, false)
	}

	p.setBuyOrders(orders)

	if len(orders)-1 < index {
		return nil
	}

	order := p.getOrder(orders[index])
	if !p.isSorted() { // todo: tests
		return order.Reverse()
	}

	return order
}

// Deprecated
func (p *Pair) OrderBuyLast() (limit *Limit, index int) {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()
	return p.orderBuyLast()
}

// Deprecated
func (p *Pair) orderBuyLast() (limit *Limit, index int) {
	for order := p.orderBuyByIndex(index); order != nil; order = p.orderBuyByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}

func (p *Pair) unsetOrderSellByIndex(index int) {
	slice := p.SellOrderIDs()
	length := len(slice)

	if length <= index {
		panic(fmt.Sprintf("slice len %d, want index %d", length, index))
	}

	if length == 1 {
		p.setSellOrders(nil)
		return
	}

	switch index {
	case 0:
		slice = slice[index+1:]
	case length - 1:
		slice = slice[:index]
	default:
		slice = append(slice[:index], slice[index+1:]...)
	}

	p.setSellOrders(slice)
	return
}

func (p *Pair) OrderSellByIndex(index int) *Limit {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()
	return p.orderSellByIndex(index)
}

func (p *Pair) orderSellByIndex(index int) *Limit {
	orders := p.SellOrderIDs()

	var fromOrder *Limit
	if len(orders) != 0 {
		fromOrder = p.getOrder(orders[0])
		if p.isUnsortedSellOrder(fromOrder.id) {
			loadedOrders := p.loadSellOrders(p, fromOrder, index+1)
			var resortedOrders []uint32
			var unsets int
			if len(loadedOrders) == 0 && len(orders) != 0 {
				resortedOrders, unsets = p.updateDirtyOrders(orders, false)
			} else {
				resortedOrders, unsets = p.updateDirtyOrders(loadedOrders, false)
			}
			for unsets > 0 {
				id := resortedOrders[len(resortedOrders)-1]
				fromOrder = p.getOrder(id)
				loadedNextOrders := p.loadSellOrders(p, fromOrder, index+1)
				resortedOrders, unsets = p.updateDirtyOrders(append(orders, loadedNextOrders...), true)
			}
			orders = resortedOrders
		} else {
			if index > len(orders)-1 {
				fromOrder = p.getOrder(orders[len(orders)-1])
				loadedNextOrders := p.loadSellOrders(p, fromOrder, index+1)
				orders, _ = p.updateDirtyOrders(append(orders, loadedNextOrders...), true)
			} else {
				orders, _ = p.updateDirtyOrders(orders, true)
			}
		}
	} else {
		orders = p.loadSellOrders(p, nil, index+1)
		orders, _ = p.updateDirtyOrders(orders, true)
	}

	p.setSellOrders(orders)

	if len(orders)-1 < index {
		return nil
	}

	order := p.getOrder(orders[index])
	//if !p.isSorted() { // todo: tests
	//	return order.Reverse()
	//}

	return order
}

func (p *Pair) OrderIDSellByIndex(index int) uint32 {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()
	return p.orderIDSellByIndex(index)
}

func (p *Pair) orderIDSellByIndex(index int) uint32 {
	orders := p.SellOrderIDs()
	var count int
	var deleteCount int
	for firstIterate := true; (firstIterate && len(orders) <= index) || deleteCount != 0; firstIterate = false {
		//orders, deleteCount = p.updateDirtyOrders(nil, p.loadSellOrders(p, orders, index+count), true)
		count += deleteCount
	}

	p.setSellOrders(orders)

	if len(orders)-1 < index {
		return 0
	}

	return orders[index]
}

func (p *Pair) OrderSellLast() (limit *Limit, index int) {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	return p.orderSellLast()
}

func (p *Pair) orderSellLast() (limit *Limit, index int) {
	for order := p.orderSellByIndex(index); order != nil; order = p.orderSellByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}
