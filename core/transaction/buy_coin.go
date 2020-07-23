package transaction

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type BuyCoinData struct {
	CoinToBuy          types.CoinID
	ValueToBuy         *big.Int
	CoinToSell         types.CoinID
	MaximumValueToSell *big.Int
}

func (data BuyCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CoinToBuy          string `json:"coin_to_buy"`
		ValueToBuy         string `json:"value_to_buy"`
		CoinToSell         string `json:"coin_to_sell"`
		MaximumValueToSell string `json:"maximum_value_to_sell"`
	}{
		CoinToBuy:          data.CoinToBuy.String(),
		ValueToBuy:         data.ValueToBuy.String(),
		CoinToSell:         data.CoinToSell.String(),
		MaximumValueToSell: data.MaximumValueToSell.String(),
	})
}

func (data BuyCoinData) String() string {
	return fmt.Sprintf("BUY COIN sell:%s buy:%s %s",
		data.CoinToSell.String(), data.ValueToBuy.String(), data.CoinToBuy.String())
}

func (data BuyCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data BuyCoinData) TotalSpend(tx *Transaction, context *state.CheckState) (TotalSpends,
	[]Conversion, *big.Int, *Response) {
	total := TotalSpends{}
	var conversions []Conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionIncluded := false

	if !data.CoinToBuy.IsBaseCoin() {
		coin := context.Coins().GetCoin(data.CoinToBuy)

		if errResp := CheckForCoinSupplyOverflow(coin.Volume(), data.ValueToBuy, coin.MaxSupply()); errResp != nil {
			return nil, nil, nil, errResp
		}
	}

	var value *big.Int

	switch {
	case data.CoinToSell.IsBaseCoin():
		coin := context.Coins().GetCoin(data.CoinToBuy)
		value = formula.CalculatePurchaseAmount(coin.Volume(), coin.Reserve(), coin.Crr(), data.ValueToBuy)

		if value.Cmp(data.MaximumValueToSell) == 1 {
			return nil, nil, nil, &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					data.MaximumValueToSell.String(), value.String()),
				Info: EncodeError(map[string]string{
					"maximum_value_to_sell": data.MaximumValueToSell.String(),
					"needed_spend_value":    value.String(),
				}),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coin.Volume())
			nVolume.Add(nVolume, data.ValueToBuy)

			nReserveBalance := big.NewInt(0).Set(coin.Reserve())
			nReserveBalance.Add(nReserveBalance, value)

			if nReserveBalance.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coin.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(map[string]string{
						"has_value":      coin.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"gas_coin":       fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coin.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, Conversion{
			FromCoin:  data.CoinToSell,
			ToCoin:    data.CoinToBuy,
			ToAmount:  data.ValueToBuy,
			ToReserve: value,
		})
	case data.CoinToBuy.IsBaseCoin():
		valueToBuy := big.NewInt(0).Set(data.ValueToBuy)

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true
			valueToBuy.Add(valueToBuy, commissionInBaseCoin)
		}

		coin := context.Coins().GetCoin(data.CoinToSell)
		value = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), valueToBuy)

		if value.Cmp(data.MaximumValueToSell) == 1 {
			return nil, nil, nil, &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					data.MaximumValueToSell.String(), value.String()),
				Info: EncodeError(map[string]string{
					"maximum_value_to_sell": data.MaximumValueToSell.String(),
					"needed_spend_value":    value.String(),
				}),
			}
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  value,
			FromReserve: valueToBuy,
			ToCoin:      data.CoinToBuy,
		})
	default:
		valueToBuy := big.NewInt(0).Set(data.ValueToBuy)

		coinFrom := context.Coins().GetCoin(data.CoinToSell)
		coinTo := context.Coins().GetCoin(data.CoinToBuy)
		baseCoinNeeded := formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToBuy)

		if coinFrom.Reserve().Cmp(baseCoinNeeded) < 0 {
			return nil, nil, nil, &Response{
				Code: code.CoinReserveNotSufficient,
				Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
					coinFrom.Reserve().String(),
					types.GetBaseCoin(),
					baseCoinNeeded.String(),
					types.GetBaseCoin()),
				Info: EncodeError(map[string]string{
					"has_value":      coinFrom.Reserve().String(),
					"required_value": commissionInBaseCoin.String(),
					"gas_coin":       fmt.Sprintf("%s", types.GetBaseCoin()),
				}),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coinTo.Volume())
			nVolume.Add(nVolume, valueToBuy)

			nReserveBalance := big.NewInt(0).Set(coinTo.Reserve())
			nReserveBalance.Add(nReserveBalance, baseCoinNeeded)

			if nReserveBalance.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coinTo.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(map[string]string{
						"has_value":      coinTo.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"gas_coin":       fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinTo.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		value = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), baseCoinNeeded)

		if value.Cmp(data.MaximumValueToSell) == 1 {
			return nil, nil, nil, &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf("You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					data.MaximumValueToSell.String(), value.String()),
				Info: EncodeError(map[string]string{
					"maximum_value_to_sell": data.MaximumValueToSell.String(),
					"needed_spend_value":    value.String(),
				}),
			}
		}

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coinFrom.Volume())
			nVolume.Sub(nVolume, value)

			nReserveBalance := big.NewInt(0).Set(coinFrom.Reserve())
			nReserveBalance.Sub(nReserveBalance, baseCoinNeeded)

			if nReserveBalance.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coinFrom.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(map[string]string{
						"has_value":      coinFrom.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"gas_coin":       fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinFrom.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})

			totalValue := big.NewInt(0).Add(value, commission)
			if totalValue.Cmp(data.MaximumValueToSell) == 1 {
				return nil, nil, nil, &Response{
					Code: code.MaximumValueToSellReached,
					Log:  fmt.Sprintf("You wanted to sell maximum %s, but currently you need to spend %s to complete tx", data.MaximumValueToSell.String(), totalValue.String()),
					Info: EncodeError(map[string]string{
						"maximum_value_to_sell": data.MaximumValueToSell.String(),
						"needed_spend_value":    value.String(),
					}),
				}
			}
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  value,
			FromReserve: baseCoinNeeded,
			ToCoin:      data.CoinToBuy,
			ToAmount:    valueToBuy,
			ToReserve:   baseCoinNeeded,
		})
	}

	if !commissionIncluded {
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if !tx.GasCoin.IsBaseCoin() {
			coin := context.Coins().GetCoin(tx.GasCoin)

			if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coin.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(map[string]string{
						"has_value":      coin.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"gas_coin":       fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		total.Add(tx.GasCoin, commission)
	}

	return total, conversions, value, nil
}

func (data BuyCoinData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.ValueToBuy == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin")}
	}

	if !context.Coins().Exists(data.CoinToSell) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.CoinToSell),
			Info: EncodeError(map[string]string{
				"coin_to_sell": fmt.Sprintf("%s", data.CoinToSell),
			}),
		}
	}

	if !context.Coins().Exists(data.CoinToBuy) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.CoinToBuy),
			Info: EncodeError(map[string]string{
				"coin_to_buy": fmt.Sprintf("%s", data.CoinToBuy),
			}),
		}
	}

	return nil
}

func (data BuyCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.BasicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	totalSpends, conversions, value, response := data.TotalSpend(tx, checkState)
	if response != nil {
		return *response
	}

	for _, ts := range totalSpends {
		if checkState.Accounts().GetBalance(sender, ts.Coin).Cmp(ts.Value) < 0 {
			coin := checkState.Coins().GetCoin(ts.Coin)

			return Response{
				Code: code.InsufficientFunds,
				Log: fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s.",
					sender.String(),
					ts.Value.String(),
					coin.GetFullSymbol()),
				Info: EncodeError(map[string]string{
					"sender":       sender.String(),
					"needed_value": ts.Value.String(),
					"coin":         coin.GetFullSymbol(),
				}),
			}
		}
	}

	errResp := checkConversionsReserveUnderflow(conversions, checkState)
	if errResp != nil {
		return *errResp
	}

	if deliverState, ok := context.(*state.State); ok {
		deliverState.Lock()
		defer deliverState.Unlock()
		for _, ts := range totalSpends {
			deliverState.Accounts.SubBalance(sender, ts.Coin, ts.Value)
		}

		for _, conversion := range conversions {
			deliverState.Coins.SubVolume(conversion.FromCoin, conversion.FromAmount)
			deliverState.Coins.SubReserve(conversion.FromCoin, conversion.FromReserve)

			deliverState.Coins.AddVolume(conversion.ToCoin, conversion.ToAmount)
			deliverState.Coins.AddReserve(conversion.ToCoin, conversion.ToReserve)
		}

		rewardPool.Add(rewardPool, tx.CommissionInBaseCoin())
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, data.ValueToBuy)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeBuyCoin)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		kv.Pair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		kv.Pair{Key: []byte("tx.return"), Value: []byte(value.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}

func checkConversionsReserveUnderflow(conversions []Conversion, context *state.CheckState) *Response {
	var totalReserveCoins []types.CoinID
	totalReserveSub := make(map[types.CoinID]*big.Int)
	for _, conversion := range conversions {
		if conversion.FromCoin.IsBaseCoin() {
			continue
		}

		if totalReserveSub[conversion.FromCoin] == nil {
			totalReserveCoins = append(totalReserveCoins, conversion.FromCoin)
			totalReserveSub[conversion.FromCoin] = big.NewInt(0)
		}

		totalReserveSub[conversion.FromCoin].Add(totalReserveSub[conversion.FromCoin], conversion.FromReserve)
	}

	for _, coinSymbol := range totalReserveCoins {
		errResp := CheckReserveUnderflow(context.Coins().GetCoin(coinSymbol), totalReserveSub[coinSymbol])
		if errResp != nil {
			return errResp
		}
	}

	return nil
}
