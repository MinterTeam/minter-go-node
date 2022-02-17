package swap

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/events"
	"log"
	"math"
	"math/big"
	"sort"
	"sync"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
)

var burnAddress = types.StringToAddress("Mxffffffffffffffffffffffffffffffffffffffff") // TODO

func (s *SwapV2) PairSellWithOrders(coin0, coin1 types.CoinID, amount0In, minAmount1Out *big.Int) (*big.Int, *big.Int, uint32, *ChangeDetailsWithOrders, []*OrderDetail) {
	pair := s.Pair(coin0, coin1)
	amount1Out, ownersMap, details, expiredOrders := pair.SellWithOrders(amount0In)
	if amount1Out.Cmp(minAmount1Out) == -1 {
		panic(fmt.Sprintf("calculatedAmount1Out %s less minAmount1Out %s", amount1Out, minAmount1Out))
	}

	s.hundleLittleExpiredOrders(expiredOrders)

	owners := sortOwners(ownersMap)
	for _, b := range owners {
		s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(b.ValueBigInt))
	}
	s.bus.Checker().AddCoin(coin0, amount0In)
	s.bus.Checker().AddCoin(coin1, big.NewInt(0).Neg(amount1Out))

	commission1000 := calcCommission1000(amount0In)
	s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(commission1000))
	s.bus.Accounts().AddBalance(burnAddress, coin0, commission1000)

	return amount0In, amount1Out, pair.GetID(), details, owners
}

func (s *SwapV2) PairBuyWithOrders(coin0, coin1 types.CoinID, maxAmount0In, amount1Out *big.Int) (*big.Int, *big.Int, uint32, *ChangeDetailsWithOrders, []*OrderDetail) {
	pair := s.Pair(coin0, coin1)
	amount0In, ownersMap, details, expiredOrders := pair.BuyWithOrders(amount1Out)
	if amount1Out.Cmp(maxAmount0In) == 1 {
		panic(fmt.Sprintf("calculatedAmount1Out %s less minAmount1Out %s", amount1Out, maxAmount0In))
	}

	s.hundleLittleExpiredOrders(expiredOrders)

	owners := sortOwners(ownersMap)
	for _, b := range owners {
		s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(b.ValueBigInt))
	}
	s.bus.Checker().AddCoin(coin0, amount0In)
	s.bus.Checker().AddCoin(coin1, big.NewInt(0).Neg(amount1Out))

	commission1000 := calcCommission1000(amount0In)
	s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(commission1000))
	s.bus.Accounts().AddBalance(burnAddress, coin0, commission1000)
	return amount0In, amount1Out, pair.GetID(), details, owners
}

func (s *SwapV2) hundleLittleExpiredOrders(expiredOrders []*Limit) {
	for _, limit := range expiredOrders {
		returnVolume := big.NewInt(0).Set(limit.WantSell)
		s.bus.Accounts().AddBalance(limit.Owner, limit.Coin1, returnVolume)
		s.bus.Checker().AddCoin(limit.Coin1, big.NewInt(0).Neg(returnVolume))
		s.bus.Events().AddEvent(&events.OrderExpiredEvent{
			ID:      uint64(limit.ID()),
			Address: limit.Owner,
			Coin:    uint64(limit.Coin1),
			Amount:  returnVolume.String(),
		})
	}
}

func (p *PairV2) SellWithOrders(amount0In *big.Int) (amount1Out *big.Int, owners map[types.Address]*big.Int, c *ChangeDetailsWithOrders, expiredOrders []*Limit) {
	if amount0In == nil || amount0In.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}

	if amount0In = big.NewInt(0).Sub(amount0In, calcCommission1000(amount0In)); amount0In.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	amount1Out, orders := p.calculateBuyForSellWithOrders(amount0In)
	if amount1Out == nil || amount1Out.Sign() != 1 {
		log.Println("s", amount0In, amount1Out)
		for _, limit := range orders {
			log.Println(limit)
		}
		panic(ErrorInsufficientOutputAmount)
	}

	commission0orders, commission1orders, amount0, amount1, ownersMap := CalcDiffPool(amount0In, amount1Out, orders)

	//log.Println("uS", commission0orders, commission1orders)

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		//log.Println("a", amount0, amount1)
		p.update(amount0, big.NewInt(0).Neg(amount1))
	}

	p.update(commission0orders, commission1orders)

	expiredOrders = p.updateOrders(orders)

	p.orderSellByIndex(0) // update list

	return amount1Out, ownersMap, &ChangeDetailsWithOrders{
		AmountIn:            amount0,
		AmountOut:           amount1,
		CommissionAmountIn:  commission0orders,
		CommissionAmountOut: commission1orders,
		Orders:              orders,
	}, expiredOrders
}

func (p *PairV2) BuyWithOrders(amount1Out *big.Int) (amount0In *big.Int, owners map[types.Address]*big.Int, c *ChangeDetailsWithOrders, expiredOrders []*Limit) { // todo: add mutex
	if amount1Out == nil || amount1Out.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	amount0In, orders := p.calculateSellForBuyWithOrders(amount1Out)
	if amount0In == nil || amount0In.Sign() != 1 {
		log.Println("b", amount0In, amount1Out)
		for _, limit := range orders {
			log.Println(limit)
		}
		panic(ErrorInsufficientOutputAmount)
	}

	commission0orders, commission1orders, amount0, amount1, ownersMap := CalcDiffPool(amount0In, amount1Out, orders)

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		p.update(amount0, big.NewInt(0).Neg(amount1))
	}

	p.update(commission0orders, commission1orders)

	expiredOrders = p.updateOrders(orders)

	p.orderSellByIndex(0) // update list

	amount0In = big.NewInt(0).Add(amount0In, calcCommission0999(amount0In))

	return amount0In, ownersMap, &ChangeDetailsWithOrders{
		AmountIn:            amount0,
		AmountOut:           amount1,
		CommissionAmountIn:  commission0orders,
		CommissionAmountOut: commission1orders,
		Orders:              orders,
	}, expiredOrders
}

func (p *PairV2) updateOrders(orders []*Limit) (littles []*Limit) {
	for _, order := range orders {
		limit := p.updateSellOrder(order.id, order.WantBuy, order.WantSell)
		//fmt.Println(limit.WantBuy, limit.WantSell)
		if !limit.isEmpty() &&
			(limit.WantBuy.Cmp(big.NewInt(minimumOrderVolume)) == -1 ||
				limit.WantSell.Cmp(big.NewInt(minimumOrderVolume)) == -1) {

			littles = append(littles, limit.clone())

			limit.WantSell.SetInt64(0)
			limit.WantBuy.SetInt64(0)
		}
		//fmt.Println(limit.WantBuy, limit.WantSell)
		p.MarkDirtyOrders(limit)
	}
	return littles
}

func (p *PairV2) updateSellOrder(id uint32, amount0, amount1 *big.Int) *Limit {
	limit := p.getOrder(id)
	limit.OldSortPrice()

	limit.WantBuy.Sub(limit.WantBuy, amount0)
	limit.WantSell.Sub(limit.WantSell, amount1)

	return limit
}

func (p *PairV2) CalculateBuyForSellWithOrders(amount0In *big.Int) (amount1Out *big.Int, orders []*Limit) {
	if amount0In != nil && amount0In.Sign() == 1 {
		amount0In = big.NewInt(0).Sub(amount0In, calcCommission1000(amount0In))
	}
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	amount1Out, orders = p.calculateBuyForSellWithOrders(amount0In)
	return amount1Out, orders
}

func (p *PairV2) calculateBuyForSellWithOrders(amount0In *big.Int) (amountOut *big.Int, orders []*Limit) {
	amountOut = big.NewInt(0)
	amountIn := big.NewInt(0).Set(amount0In)
	var pair EditableChecker = p
	for i := 0; true; i++ {
		if amountIn.Sign() == -1 {
			log.Println(fmt.Sprint(amountIn, amountOut))
		}
		if amountIn.Sign() == 0 {
			return amountOut, orders
		}

		limit := p.orderSellByIndex(i)
		if limit == nil {
			break
		}

		price := limit.Price()
		if pair.PriceRatCmp(limit.PriceRat()) == 1 {
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
			}
		}

		// хотим продать 1001 (1 пойдет в пул)
		// проверяем есть ли 1000 на продажу
		amount0 := big.NewInt(0).Sub(amountIn, calcCommission1001(amountIn))
		if amount0.Cmp(limit.WantBuy) != 1 {
			// считаем сколько сможем купить -- 1000
			amount1, acc := new(big.Float).SetRat(new(big.Rat).Mul(limit.PriceRat(), new(big.Rat).SetFrac(amount0, big.NewInt(1)))).Int(nil)
			if acc != big.Exact {
				//log.Println("acc", acc)
			}

			if amount1.Cmp(limit.WantSell) == 0 && amount0.Cmp(limit.WantBuy) != 0 {
				log.Println(limit.WantBuy, limit.WantSell, limit.id)
				log.Println(amount0, amount1)
				log.Panicln("neg BFS 0")
			}

			if amount1.Cmp(limit.WantSell) == 1 {
				log.Println(limit.WantBuy, limit.WantSell, limit.id)
				log.Println(amount0, amount1)
				log.Println("b")
				amount1.Set(limit.WantSell)
				if amount0.Cmp(limit.WantBuy) == -1 {
					log.Println("bc")
					amount1.Sub(amount1, big.NewInt(1))
				}
			}

			if amount1.Cmp(limit.WantSell) == -1 && amount0.Cmp(limit.WantBuy) == 0 {
				log.Println(limit.WantBuy, limit.WantSell, limit.id)
				log.Println(amount0, amount1)
				log.Println("a")
				amount1.Set(limit.WantSell)
			}

			orders = append(orders, &Limit{
				IsBuy:        limit.IsBuy,
				PairKey:      p.PairKey,
				WantBuy:      amount0, // 1000
				WantSell:     amount1, // 1000, 1 в пул и 999 тейкеру
				Owner:        limit.Owner,
				Height:       limit.Height,
				oldSortPrice: limit.sortPrice(),
				id:           limit.id,
				mu:           new(sync.RWMutex),
			})

			comB := calcCommission1000(amount1)
			amountOut.Add(amountOut, big.NewInt(0).Sub(amount1, comB)) // 999
			return amountOut, orders
		}

		orders = append(orders, &Limit{
			IsBuy:        limit.IsBuy,
			WantBuy:      big.NewInt(0).Set(limit.WantBuy),
			WantSell:     big.NewInt(0).Set(limit.WantSell),
			Owner:        limit.Owner,
			Height:       limit.Height,
			PairKey:      limit.PairKey,
			oldSortPrice: limit.sortPrice(),
			id:           limit.id,
			mu:           limit.mu,
		})

		comS := calcCommission1000(limit.WantBuy)
		comB := calcCommission1000(limit.WantSell)
		pair = pair.AddLastSwapStep(comS, big.NewInt(0).Neg(comB))

		amountOut.Add(amountOut, big.NewInt(0).Sub(limit.WantSell, comB))
		amountIn = big.NewInt(0).Sub(amountIn, big.NewInt(0).Add(limit.WantBuy, comS))
	}

	amount1diff := pair.CalculateBuyForSell(amountIn)
	if amount1diff != nil {
		if err := pair.CheckSwap(amountIn, amount1diff); err != nil {
			fmt.Println(amountIn, amount1diff)
			panic(err)
		}
		amountOut.Add(amountOut, amount1diff)
	}
	return amountOut, orders
}

func (p *PairV2) CalculateAddAmountsForPrice(price *big.Float) (amount0In, amount1Out *big.Int) {
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
func (p *PairV2) calculateAddAmountsForPrice(price *big.Float) (amount0 *big.Int, amount1 *big.Int) {
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
	var acc big.Accuracy
	amount0, acc = x1.Int(nil)
	if acc != big.Exact {

	}

	if amount0 == nil || amount0.Sign() != 1 {
		return nil, nil
	}

	amount1Out := p.CalculateBuyForSell(amount0)
	if amount1Out == nil {
		return nil, nil
	}

	return amount0, amount1Out
}

func (p *PairV2) CalculateSellForBuyWithOrders(amount1Out *big.Int) (amount0In *big.Int, orders []*Limit) {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	amount0In, orders = p.calculateSellForBuyWithOrders(amount1Out)

	if amount0In != nil && amount0In.Sign() == 1 {
		amount0In = big.NewInt(0).Add(amount0In, calcCommission0999(amount0In))
	}

	return amount0In, orders
}

func (p *PairV2) calculateSellForBuyWithOrders(amount1Out *big.Int) (amountIn *big.Int, orders []*Limit) {
	amountIn = big.NewInt(0)
	amountOut := big.NewInt(0).Set(amount1Out)
	var pair EditableChecker = p
	for i := 0; true; i++ {
		if amountOut.Sign() == -1 {
			log.Println(fmt.Sprint(amountIn, amountOut))
		}
		// todo: move check minAmountIn
		if amountOut.Sign() == 0 {
			return amountIn, orders
		}

		limit := p.orderSellByIndex(i)
		if limit == nil {
			break
		}

		price := limit.Price()
		if pair.PriceRatCmp(limit.PriceRat()) == 1 {
			reserve0diff, reserve1diff := pair.CalculateAddAmountsForPrice(price)
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
			}
		}

		amount1 := big.NewInt(0).Add(amountOut, calcCommission0999(amountOut))
		if amount1.Cmp(limit.WantSell) != 1 {
			amount0, acc := new(big.Float).SetRat(new(big.Rat).Quo(new(big.Rat).SetFrac(amount1, big.NewInt(1)), limit.PriceRat())).Int(nil)
			if acc != big.Exact {
			}

			if amount1.Cmp(limit.WantSell) == 0 && amount0.Cmp(limit.WantBuy) != 0 {

				if amount0.Cmp(limit.WantBuy) == -1 {
					log.Println(limit.WantBuy, limit.WantSell, limit.id)
					log.Println(amount0, amount1)
					log.Println("d")
					amount0.Set(limit.WantBuy)
				} else {
					log.Println(limit.WantBuy, limit.WantSell, limit.id)
					log.Println(amount0, amount1)
					log.Panicln("neg SFB 0")
				}
			}
			if amount1.Cmp(limit.WantSell) == -1 && amount0.Cmp(limit.WantBuy) == 0 {
				log.Println(limit.WantBuy, limit.WantSell, limit.id)
				log.Println(amount0, amount1)
				log.Println("f")
				amount1.Set(limit.WantSell)
			}

			orders = append(orders, &Limit{
				IsBuy:        limit.IsBuy,
				PairKey:      p.PairKey,
				WantBuy:      big.NewInt(0).Set(amount0),
				WantSell:     amount1,
				Owner:        limit.Owner,
				Height:       limit.Height,
				oldSortPrice: limit.sortPrice(),
				id:           limit.id,
				mu:           new(sync.RWMutex),
			})

			com := calcCommission1000(amount0)
			amountIn.Add(amountIn, amount0)
			amountIn.Add(amountIn, com)
			return amountIn, orders
		}

		orders = append(orders, &Limit{
			IsBuy:        limit.IsBuy,
			WantBuy:      big.NewInt(0).Set(limit.WantBuy),
			WantSell:     big.NewInt(0).Set(limit.WantSell),
			Owner:        limit.Owner,
			Height:       limit.Height,
			PairKey:      limit.PairKey,
			oldSortPrice: limit.sortPrice(),
			id:           limit.id,
			mu:           limit.mu,
		})

		comB := calcCommission1000(limit.WantSell)
		comS := calcCommission1000(limit.WantBuy)

		pair = pair.AddLastSwapStep(comS, big.NewInt(0).Neg(comB))

		amountOut = big.NewInt(0).Sub(amountOut, big.NewInt(0).Sub(limit.WantSell, comB))
		amountIn.Add(amountIn, big.NewInt(0).Add(limit.WantBuy, comS))
	}

	amount0diff := pair.CalculateSellForBuy(amountOut)
	if amount0diff != nil {
		if err := pair.CheckSwap(amount0diff, amountOut); err != nil {
			fmt.Println(amount0diff, amountOut)
			panic(err)
		}
		amountIn.Add(amountIn, amount0diff)
	}
	return amountIn, orders
}

func (p *PairV2) Price() *big.Float {
	return p.pairData.Price()
}
func (p *PairV2) PriceRat() *big.Rat {
	return p.pairData.PriceRat()
}
func (p *PairV2) PriceRatCmp(rat *big.Rat) int {
	return p.pairData.PriceRat().Cmp(rat)
}

func (p *PairV2) getOrder(id uint32) *Limit {
	p.orders.mu.Lock()
	defer p.orders.mu.Unlock()

	return p.order(id)
}

func (p *PairV2) order(id uint32) *Limit {
	l, ok := p.orders.list[id]
	if ok {
		if l == nil {
			return nil
		}

		if p.isSorted() {
			return l
		}

		return l.Reverse()
	}

	l = p.loadOrder(id)
	if l == nil {
		return nil
	}

	p.orders.list[id] = l

	if p.isSorted() {
		return l
	}

	return l.Reverse()
}

func (p *PairV2) GetOrders(ids []uint32) []*Limit {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	p.orders.mu.Lock()
	defer p.orders.mu.Unlock()

	return p.getOrders(ids)
}

func (p *PairV2) getOrders(ids []uint32) []*Limit {
	var result []*Limit
	for _, id := range ids {
		result = append(result, p.order(id))
	}
	return result
}

func (p *PairV2) setOrder(l *Limit) {
	p.orders.mu.Lock()
	defer p.orders.mu.Unlock()

	p.orders.list[l.id] = l
}

func (p *PairV2) DirectionSortPrice() int {
	if !p.isSorted() {
		return 1
	}
	return -1
}

func (p *PairV2) MarkDirtyOrders(order *Limit) {
	p.markDirtyOrders()

	if order.isEmpty() {
		p.setDeletedSellOrderIDs(order.id)
	} else if !order.isKeepRate() {
		p.setUnsortedSellOrder(order.id)
	}

	p.dirtyOrders.mu.Lock()
	p.dirtyOrders.list[order.id] = struct{}{}
	p.dirtyOrders.mu.Unlock()
	return
}

func (p *PairV2) sellOrderIDs() []uint32 {
	if p.isSorted() {
		return p.sellOrders.ids
	}
	return p.buyOrders.ids
}

func (p *PairV2) isUnsortedSellOrder(id uint32) bool {
	ds := p.unsortedSellOrderIDs()
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	_, ok := ds.list[id]
	return ok
}

func (p *PairV2) hasUnsortedSellOrders() bool {
	return len(p.unsortedSellOrderIDs().list) > 0
}

func (p *PairV2) unsortedSellOrderIDs() *orderDirties {
	if p.isSorted() {
		return p.unsortedDirtySellOrders
	}
	return p.unsortedDirtyBuyOrders
}

func (p *PairV2) setUnsortedSellOrder(id uint32) {
	us := p.unsortedSellOrderIDs()
	us.mu.Lock()
	defer us.mu.Unlock()
	us.list[id] = struct{}{}

	ds := p.deletedSellOrderIDs()
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.list, id)
}

func (p *PairV2) hasDeletedSellOrders() bool {
	return len(p.deletedSellOrderIDs().list) > 0
}

func (p *PairV2) setDeletedSellOrderIDs(id uint32) {
	us := p.unsortedSellOrderIDs()
	us.mu.Lock()
	defer us.mu.Unlock()
	delete(us.list, id)

	ds := p.deletedSellOrderIDs()
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.list[id] = struct{}{}
}
func (p *PairV2) isDeletedSellOrder(id uint32) bool {
	ds := p.deletedSellOrderIDs()
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	_, ok := ds.list[id]
	return ok
}
func (p *PairV2) isDeletedBuyOrder(id uint32) bool {
	ds := p.deletedBuyOrderIDs()
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	_, ok := ds.list[id]
	return ok
}
func (p *PairV2) deletedSellOrderIDs() *orderDirties {
	if p.isSorted() {
		return p.deletedSellOrders
	}
	return p.deletedBuyOrders
}
func (p *PairV2) deletedBuyOrderIDs() *orderDirties {
	if p.isSorted() {
		return p.deletedBuyOrders
	}
	return p.deletedSellOrders
}

func (p *PairV2) loadedSellOrderIDs() []uint32 {
	return p.loadedSellOrders.ids
}

func (p *PairV2) loadedBuyOrderIDs() []uint32 {
	return p.loadedBuyOrders.ids
}

func (p *PairV2) setSellOrders(orders []uint32) {
	if p.isSorted() {
		p.sellOrders.ids = orders
		return
	}
	p.buyOrders.ids = orders
	return
}
func (p *PairV2) setLoadedSellOrders(orders []uint32) {
	p.loadedSellOrders.ids = orders
	return
}
func (p *PairV2) setLoadedBuyOrders(orders []uint32) {
	p.loadedBuyOrders.ids = orders
	return
}

func (p *PairV2) isDirtyOrder(id uint32) bool {
	p.dirtyOrders.mu.RLock()
	defer p.dirtyOrders.mu.RUnlock()

	_, ok := p.dirtyOrders.list[id]
	return ok
}

func (p *PairV2) getDirtyOrdersList() []uint32 {
	dirtiesOrders := make([]uint32, 0, len(p.dirtyOrders.list))

	p.dirtyOrders.mu.RLock()
	for id := range p.dirtyOrders.list {
		dirtiesOrders = append(dirtiesOrders, id)
	}
	p.dirtyOrders.mu.RUnlock()

	sort.SliceStable(dirtiesOrders, func(i, j int) bool {
		return dirtiesOrders[i] < dirtiesOrders[j]
	})
	return dirtiesOrders
}

func (s *SwapV2) PairAddOrder(coinWantBuy, coinWantSell types.CoinID, wantBuyAmount, wantSellAmount *big.Int, sender types.Address, block uint64) (uint32, uint32) {
	pair := s.Pair(coinWantBuy, coinWantSell)
	order := pair.AddOrder(wantBuyAmount, wantSellAmount, sender, block)

	s.bus.Checker().AddCoin(coinWantSell, wantSellAmount)

	return order.id, pair.GetID()
}

func (s *SwapV2) PairRemoveLimitOrder(id uint32) (types.CoinID, *big.Int) {
	order := s.loadOrder(id)
	if order == nil {
		return 0, big.NewInt(0)
	}

	return s.removeLimitOrder(order)
}

func (s *SwapV2) removeLimitOrder(order *Limit) (types.CoinID, *big.Int) {
	if !order.isSell() {
		order = order.Reverse()
	}

	pair := s.Pair(order.Coin0, order.Coin1)

	pair.lockOrders.Lock()
	defer pair.lockOrders.Unlock()

	if pair.isDirtyOrder(order.ID()) {
		if pair.isOrderAlreadyUsed(order.ID()) {
			return 0, big.NewInt(0)
		}

		order = pair.getOrder(order.ID())
		if order == nil || order.isEmpty() {
			return 0, big.NewInt(0)
		}
	} else {
		order.reCalcOldSortPrice()
	}

	returnVolume := big.NewInt(0).Set(order.WantSell)

	s.bus.Checker().AddCoin(order.Coin1, big.NewInt(0).Neg(returnVolume))

	pair.updateOrders([]*Limit{order})
	pair.orderSellByIndex(0)

	return order.Coin1, returnVolume
}

func (s *SwapV2) pairAddOrderWithID(coinWantBuy, coinWantSell types.CoinID, wantBuyAmount, wantSellAmount *big.Int, sender types.Address, id uint32, height uint64) (uint32, uint32) {
	pair := s.Pair(coinWantBuy, coinWantSell)
	order := pair.addOrderWithID(wantBuyAmount, wantSellAmount, sender, id, height)

	s.bus.Checker().AddCoin(coinWantSell, wantSellAmount)

	return order.id, pair.GetID()
}

func (p *PairV2) GetOrder(id uint32) *Limit {
	return p.getOrder(id)
}
func (p *PairV2) IsOrderAlreadyUsed(id uint32) bool {
	if p.isOrderAlreadyUsed(id) {
		return true
	}

	order := p.getOrder(id)
	return order == nil || order.isEmpty()
}

func (p *PairV2) isOrderAlreadyUsed(id uint32) bool {
	if p.isDeletedBuyOrder(id) || p.isDeletedSellOrder(id) {
		return true
	}
	return false
}

func (p *PairV2) AddOrder(wantBuyAmount0, wantSellAmount1 *big.Int, sender types.Address, block uint64) (order *Limit) {
	order = &Limit{
		PairKey:      p.PairKey,
		IsBuy:        false,
		WantBuy:      wantBuyAmount0,
		WantSell:     wantSellAmount1,
		id:           p.getLastTotalOrderID(),
		oldSortPrice: new(big.Float).SetPrec(Precision),
		Owner:        sender,
		mu:           new(sync.RWMutex),
		Height:       block,
	}
	sortedOrder := order.sort()

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	p.MarkDirtyOrders(sortedOrder)

	p.setOrder(sortedOrder)
	p.orderSellByIndex(0)
	return order
}

func (p *PairV2) addOrderWithID(wantBuyAmount0, wantSellAmount1 *big.Int, sender types.Address, id uint32, height uint64) (order *Limit) {
	order = &Limit{
		PairKey:      p.PairKey,
		IsBuy:        false,
		WantBuy:      wantBuyAmount0,
		WantSell:     wantSellAmount1,
		id:           id,
		oldSortPrice: new(big.Float).SetPrec(Precision),
		Owner:        sender,
		Height:       height,
		mu:           new(sync.RWMutex),
	}
	sortedOrder := order.sort()

	p.MarkDirtyOrders(sortedOrder)

	p.setOrder(sortedOrder)
	return order
}

func (p *PairV2) loadAllOrders(immutableTree *iavl.ImmutableTree) (orders []*Limit) {
	const countFirstBytes = 10

	startKey := append(append([]byte{mainPrefix}, p.pathOrders()...), byte(0), byte(0))
	endKey := append(append(append([]byte{mainPrefix}, p.pathOrders()...), byte(1), byte(255)), id2Bytes(math.MaxUint32)...)

	immutableTree.IterateRange(startKey, endKey, true, func(key []byte, value []byte) bool {
		var isSell = true
		if key[countFirstBytes : countFirstBytes+1][0] == 0 {
			isSell = false
		}

		id := binary.BigEndian.Uint32(key[len(key)-4:])
		if isSell {
			id = math.MaxUint32 - id
		}
		order := p.getOrder(id)
		if order != nil {
			orders = append(orders, order)
		}

		return false
	})

	return orders
}

func (s *SwapV2) loadBuyOrders(pair *PairV2, fromOrder *Limit, limit int) []uint32 {
	endKey := append(append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(0), byte(255)), id2Bytes(math.MaxUint32)...)
	var startKey = append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(0), byte(0))

	var loadedAll bool
	ids := pair.loadedBuyOrderIDs()
	if len(ids) != 0 && ids[len(ids)-1] == 0 {
		loadedAll = true
		ids = ids[: len(ids)-1 : len(ids)-1]
	}

	if fromOrder == nil && len(ids) >= limit {
		return ids[:limit:limit]
	}

	k := 1
	var slice []uint32
	for i, id := range ids {
		if id == fromOrder.ID() {
			if len(ids[i+1:]) < limit {
				slice = append(slice, ids[i+1:]...)
				k += len(ids[i+1:])
				fromOrder = pair.order(ids[len(ids)-1])
				break
			}

			return ids[i+1 : i+limit+1 : i+limit+1]
		}
	}

	if loadedAll {
		return append(slice, 0)
	}

	if fromOrder != nil {
		startKey = pricePath(pair.PairKey, fromOrder.OldSortPrice(), fromOrder.id+1, false) // todo: tests OldSortPrice
	}

	var has bool
	s.immutableTree().IterateRange(startKey, endKey, true, func(key []byte, _ []byte) bool {
		id := binary.BigEndian.Uint32(key[len(key)-4:])

		l, ok := pair.orders.list[id]
		if ok && l == nil {
			return false
		}

		has = true
		if k > limit {
			return true
		}

		slice = append(slice, id)
		k++
		return false
	})

	if !has {
		slice = append(slice, 0)
	}

	pair.setLoadedBuyOrders(append(ids, slice...))
	return slice
}

func (s *SwapV2) GetOrder(id uint32) *Limit {
	order := s.loadOrder(id)
	if order == nil {
		return nil
	}

	pair := s.Pair(order.Coin0, order.Coin1)

	pair.lockOrders.Lock()
	defer pair.lockOrders.Unlock()

	if pair.IsOrderAlreadyUsed(id) {
		return nil
	}

	pair.orders.mu.Lock()
	defer pair.orders.mu.Unlock()

	if o, ok := pair.orders.list[id]; ok {
		return o
	}

	pair.orders.list[id] = order
	return order
}

func (s *SwapV2) loadOrder(id uint32) *Limit {
	_, value := s.immutableTree().Get(pathOrder(id))
	if value == nil {
		return nil
	}

	order := &Limit{
		id:           id,
		oldSortPrice: new(big.Float).SetPrec(Precision),
		mu:           new(sync.RWMutex),
	}
	err := rlp.DecodeBytes(value, order)
	if err != nil {
		panic(err)
	}

	order.reCalcOldSortPrice()

	return order
}

func (s *SwapV2) loadSellOrders(pair *PairV2, fromOrder *Limit, limit int) []uint32 {
	startKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(1), byte(0))
	var endKey = append(append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(1), byte(255)), id2Bytes(math.MaxUint32)...)

	var loadedAll bool
	ids := pair.loadedSellOrderIDs()
	if len(ids) != 0 && ids[len(ids)-1] == 0 {
		loadedAll = true
		ids = ids[: len(ids)-1 : len(ids)-1]
	}

	if fromOrder == nil && len(ids) >= limit {
		return ids[:limit:limit]
	}
	k := 1
	var slice []uint32
	for i, id := range ids {
		if id == fromOrder.ID() {
			if len(ids[i+1:]) < limit {
				slice = append(slice, ids[i+1:]...)
				k += len(ids[i+1:])
				fromOrder = pair.order(ids[len(ids)-1])
				break
			}

			return ids[i+1 : i+limit+1 : i+limit+1]
		}
	}

	if loadedAll {
		return append(slice, 0)
	}

	if fromOrder != nil {
		endKey = pricePath(pair.PairKey, fromOrder.OldSortPrice(), fromOrder.id, true)
	}

	var has bool
	s.immutableTree().IterateRange(startKey, endKey, false, func(key []byte, value []byte) bool {
		id := math.MaxUint32 - binary.BigEndian.Uint32(key[len(key)-4:])

		l, ok := pair.orders.list[id]
		if ok && l == nil {
			return false
		}

		has = true
		if k > limit {
			return true
		}

		slice = append(slice, id)
		k++
		return false
	})

	if !has || len(slice) < limit {
		slice = append(slice, 0)
	}

	pair.setLoadedSellOrders(append(ids, slice...))
	return slice
}

func (p *PairV2) updateDirtyOrders(list []uint32, lower bool) (orders []uint32, delCount int) {
	var limits []*Limit
	for _, orderID := range list {
		if _, ok := p.deletedSellOrderIDs().list[orderID]; ok {
			delCount++
			continue
		}
		if _, ok := p.unsortedSellOrderIDs().list[orderID]; ok {
			delCount++
			continue
		}

		order := p.order(orderID)
		limits = append(limits, order)
		if order == nil {
			break
		}
	}

	cmp := p.DirectionSortPrice()
	if !lower {
		cmp *= -1
	}

	var dirties []*Limit
	for orderID := range p.unsortedSellOrderIDs().list {
		dirty := p.order(orderID)
		dirties = append(dirties, dirty)
	}

	sort.Slice(dirties, func(i, j int) bool {
		a := dirties[j]
		b := dirties[i]
		switch a.sortPrice().Cmp(b.sortPrice()) {
		case cmp:
			return true
		case 0:
			return a.id < b.id
		default:
			return false
		}
	})

	var pos int
	for _, dirty := range dirties {
		var isSet bool
		limits, isSet, pos = addToList(limits, dirty, cmp, pos)
		if isSet {
			delCount--
			delete(p.unsortedSellOrderIDs().list, dirty.id)
		} else {
			break
		}
	}

	for _, order := range limits {
		if order == nil {
			return append(orders, 0), delCount
		}
		orders = append(orders, order.id)
	}

	return orders, delCount
}

func (p *PairV2) OrderSellByIndex(index int) *Limit {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	return p.orderSellByIndex(index)
}

func (p *PairV2) orderSellLoadToIndex(index int) *Limit {
	p.unsortedSellOrderIDs().mu.Lock()
	defer p.unsortedSellOrderIDs().mu.Unlock()

	p.deletedSellOrderIDs().mu.Lock()
	defer p.deletedSellOrderIDs().mu.Unlock()

	orders := p.sellOrderIDs()

	var fromOrder *Limit
	// если массив не пустой, то пересортировать, если есть грязные!
	if len(orders) != 0 {
		// если есть грязные.
		if p.hasUnsortedSellOrders() || p.hasDeletedSellOrders() {
			// пересортируем, что бы лист почистился и пересортировался
			orders, _ = p.updateDirtyOrders(orders, true)
			lastI := len(orders) - 1
			// если загружены не все
			if lastI >= 0 && orders[lastI] != 0 {
				// проверяем есть ли среди этого массива, элемент с нужным индексом
				if index > lastI {
					//log.Println("b")
					// загрузим с последнего нужное количество и отсортируем
					fromOrder = p.order(orders[lastI])
					loadedNextOrders := p.loadSellOrders(p, fromOrder, index-lastI)
					resortedOrders, unsets := p.updateDirtyOrders(append(orders, loadedNextOrders...), true)
					//resortedOrders, unsets := p.updateDirtyOrders(append(orders, loadedNextOrders...), true)
					// проверим загружены ли все
					lastJ := len(resortedOrders) - 1
					if resortedOrders[lastJ] != 0 {
						//log.Println("c")
						// среди них не может быть использованных иначе бы они были загружены ранее,
						// но могут быть удаленные удаленных, проверим
						for ; index > lastJ && lastJ >= 0 && resortedOrders[lastJ] != 0 && p.hasDeletedSellOrders() && unsets > 0; lastJ = len(resortedOrders) - 1 {
							//log.Println("d")
							fromOrder = p.order(resortedOrders[lastI])
							loadedNextOrders := p.loadSellOrders(p, fromOrder, index-lastI+unsets)
							var resortLoadedNextOrders []uint32
							resortLoadedNextOrders, unsets = p.updateDirtyOrders(loadedNextOrders, true)
							resortedOrders = append(resortedOrders, resortLoadedNextOrders...)
						}
					}
					orders = resortedOrders
				}
			}
		} else {
			// проверим количество
			lastI := len(orders) - 1
			// если загружены не все и их не достаточно, то подгрузить
			if orders[lastI] != 0 && index > lastI {
				//log.Println("e")
				fromOrder = p.order(orders[lastI])
				loadedNextOrders := p.loadSellOrders(p, fromOrder, index-lastI)
				// тк нет грязных, то просто складываем
				orders = append(orders, loadedNextOrders...)
			}
		}
	} else {
		num := index
		for {
			orders = append(orders, p.loadSellOrders(p, fromOrder, num+1)...)
			num = 0
			if p.hasUnsortedSellOrders() || p.hasDeletedSellOrders() {
				orders, num = p.updateDirtyOrders(orders, true)
			}
			if num <= 0 {
				break
			}
			lenOrders := len(orders)
			if lenOrders != 0 && orders[lenOrders-1] != 0 {
				fromOrder = p.order(orders[lenOrders-1])
			} else {
				break
			}
		}
	}

	p.setSellOrders(orders)
	i := len(orders) - 1
	if i >= 0 && orders[i] == 0 {
		i--
	}
	if i < index {
		return nil
	}

	order := p.order(orders[index])

	return order
}

func (p *PairV2) orderSellByIndex(index int) *Limit {
	p.orders.mu.Lock()
	defer p.orders.mu.Unlock()

	return p.orderSellLoadToIndex(index)
}

func (p *PairV2) ordersSellToIndex(index int) []*Limit {
	p.orders.mu.Lock()
	defer p.orders.mu.Unlock()

	p.orderSellLoadToIndex(index)

	orderIDs := p.sellOrderIDs()
	if len(orderIDs) > index {
		return p.getOrders(orderIDs[: index+1 : index+1])
	}

	return p.getOrders(orderIDs)
}

func (p *PairV2) OrdersSell(limit uint32) []*Limit {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	return p.ordersSell(limit)
}

func (p *PairV2) ordersSell(limit uint32) []*Limit {
	index := int(limit - 1)

	return p.ordersSellToIndex(index)

}

// Deprecated
func (p *PairV2) OrderSellLast() (limit *Limit, index int) {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	return p.orderSellLast()
}

// Deprecated
func (p *PairV2) orderSellLast() (limit *Limit, index int) {
	for order := p.orderSellByIndex(index); order != nil; order = p.orderSellByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}

func (p *PairV2) AddLastSwapStepWithOrders(amount0In, amount1Out *big.Int, buy bool) EditableChecker {
	if amount0In.Sign() == -1 || amount1Out.Sign() == -1 {
		return p.reverse().AddLastSwapStepWithOrders(big.NewInt(0).Neg(amount1Out), big.NewInt(0).Neg(amount0In), !buy).Reverse()
	}

	p.lockOrders.Lock()

	var orders []*Limit
	if buy {
		amount0InCalc, ordrs := p.calculateSellForBuyWithOrders(amount1Out)
		if amount0InCalc.Cmp(amount0In) != 0 {
			log.Println("AddLastSwapStepWithOrders calculateSellForBuyWithOrders error", amount0InCalc, amount0In)
		}
		orders = ordrs
	} else {
		amount1OutCalc, ordrs := p.calculateBuyForSellWithOrders(amount0In)
		if amount1OutCalc.Cmp(amount1Out) != 0 {
			log.Println("AddLastSwapStepWithOrders calculateBuyForSellWithOrders error", amount1OutCalc, amount1Out)
		}
		orders = ordrs
	}

	reserve0, reserve1 := p.Reserves()

	ordrs := make(map[uint32]*Limit, len(p.orders.list))
	dirtyOrdrs := make(map[uint32]struct{}, len(p.dirtyOrders.list))

	for k, v := range p.orders.list {
		if v == nil {
			ordrs[k] = nil
			continue
		}
		ordrs[k] = v.clone()
	}
	for k, v := range p.dirtyOrders.list {
		dirtyOrdrs[k] = v
	}

	unsortedDirtySellOrders := map[uint32]struct{}{}
	p.unsortedDirtySellOrders.mu.Lock()
	for k, v := range p.unsortedDirtySellOrders.list {
		unsortedDirtySellOrders[k] = v
	}
	p.unsortedDirtySellOrders.mu.Unlock()
	unsortedDirtyBuyOrders := map[uint32]struct{}{}
	p.unsortedDirtyBuyOrders.mu.Lock()
	for k, v := range p.unsortedDirtyBuyOrders.list {
		unsortedDirtyBuyOrders[k] = v
	}
	p.unsortedDirtyBuyOrders.mu.Unlock()
	deletedSellOrders := map[uint32]struct{}{}
	p.deletedSellOrders.mu.Lock()
	for k, v := range p.deletedSellOrders.list {
		deletedSellOrders[k] = v
	}
	p.deletedSellOrders.mu.Unlock()

	deletedBuyOrders := map[uint32]struct{}{}
	p.deletedBuyOrders.mu.Lock()
	for k, v := range p.deletedBuyOrders.list {
		deletedBuyOrders[k] = v
	}
	p.deletedBuyOrders.mu.Unlock()

	pair := &PairV2{
		lockOrders: &sync.Mutex{},
		PairKey:    p.PairKey,
		pairData: &pairData{
			RWMutex:   &sync.RWMutex{},
			Reserve0:  reserve0,
			Reserve1:  reserve1,
			ID:        p.ID,
			markDirty: func() {},
		},
		sellOrders: &limits{
			ids: p.sellOrders.ids[:len(p.sellOrders.ids):len(p.sellOrders.ids)],
		},
		buyOrders: &limits{
			ids: p.buyOrders.ids[:len(p.buyOrders.ids):len(p.buyOrders.ids)],
		},
		orders: &orderList{
			mu:   sync.RWMutex{},
			list: ordrs,
		},
		dirtyOrders: &orderDirties{
			mu:   sync.RWMutex{},
			list: dirtyOrdrs,
		},
		deletedSellOrders: &orderDirties{
			mu:   sync.RWMutex{},
			list: deletedSellOrders,
		},
		deletedBuyOrders: &orderDirties{
			mu:   sync.RWMutex{},
			list: deletedBuyOrders,
		},
		markDirtyOrders: p.markDirtyOrders,
		loadBuyOrders:   p.loadBuyOrders,
		loadSellOrders:  p.loadSellOrders,
		loadedSellOrders: &limits{
			ids: p.loadedSellOrders.ids[:len(p.loadedSellOrders.ids):len(p.loadedSellOrders.ids)],
		},
		loadedBuyOrders: &limits{
			ids: p.loadedBuyOrders.ids[:len(p.loadedBuyOrders.ids):len(p.loadedBuyOrders.ids)],
		},
		unsortedDirtyBuyOrders: &orderDirties{
			mu:   sync.RWMutex{},
			list: unsortedDirtyBuyOrders,
		},
		unsortedDirtySellOrders: &orderDirties{
			mu:   sync.RWMutex{},
			list: unsortedDirtySellOrders,
		},
		getLastTotalOrderID: nil,
		loadOrder:           p.loadOrder,
	}
	commission0orders, commission1orders, amount0, amount1, _ := CalcDiffPool(amount0In, amount1Out, orders)

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		pair.update(amount0, big.NewInt(0).Neg(amount1))
	}

	pair.update(commission0orders, commission1orders)

	oo := make([]*Limit, 0, len(orders))
	for _, order := range orders {
		oo = append(oo, &Limit{
			IsBuy:        order.IsBuy,
			WantBuy:      big.NewInt(0).Set(order.WantBuy),
			WantSell:     big.NewInt(0).Set(order.WantSell),
			Owner:        order.Owner,
			PairKey:      order.PairKey,
			oldSortPrice: new(big.Float).SetPrec(Precision).Set(order.oldSortPrice),
			id:           order.id,
			mu:           &sync.RWMutex{},
		})
	}

	p.lockOrders.Unlock()

	pair.updateOrders(oo)

	pair.orderSellByIndex(0)

	return pair
}
