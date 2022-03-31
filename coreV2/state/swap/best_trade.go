package swap

import "math/big"

func (s *Swap) GetBestTradeExactOut(fromId, toId uint64, amount *big.Int, maxNumResults, maxHops int) ([]*Trade, error) {
	s.loadPools()

	var pairs []*PairTrade

	s.muPairs.RLock()
	for key, pair := range s.pairs {
		if pair == nil {
			continue
		}
		reserve0, reserve1 := pair.Reserves()

		pairs = append(pairs, NewPair(
			NewTokenAmount(NewToken(uint64(key.Coin0)), reserve0),
			NewTokenAmount(NewToken(uint64(key.Coin1)), reserve1),
		))
	}
	s.muPairs.RUnlock()

	trades, err := GetBestTradeExactOut(
		pairs,
		NewToken(toId),
		NewTokenAmount(NewToken(fromId), amount),
		TradeOptions{MaxNumResults: maxNumResults, MaxHops: maxHops},
	)
	if err != nil {
		return nil, err
	}

	return trades, nil
}
func (s *Swap) GetBestTradeExactIn(fromId, toId uint64, amount *big.Int, maxNumResults, maxHops int) ([]*Trade, error) {
	s.loadPools()

	var pairs []*PairTrade

	s.muPairs.RLock()
	for key, pair := range s.pairs {
		if pair == nil {
			continue
		}
		reserve0, reserve1 := pair.Reserves()

		pairs = append(pairs, NewPair(
			NewTokenAmount(NewToken(uint64(key.Coin0)), reserve0),
			NewTokenAmount(NewToken(uint64(key.Coin1)), reserve1),
		))
	}
	s.muPairs.RUnlock()

	trades, err := GetBestTradeExactIn(
		pairs,
		NewToken(fromId),
		NewTokenAmount(NewToken(toId), amount),
		TradeOptions{MaxNumResults: maxNumResults, MaxHops: maxHops},
	)
	if err != nil {
		return nil, err
	}

	return trades, nil
}
