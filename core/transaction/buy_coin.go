package transaction

import (
	"encoding/hex"
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

func (data BuyCoinData) String() string {
	return fmt.Sprintf("BUY COIN sell:%s buy:%s %s",
		data.CoinToSell.String(), data.ValueToBuy.String(), data.CoinToBuy.String())
}

func (data BuyCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data BuyCoinData) totalSpend(tx *Transaction, context *state.CheckState) (totalSpends, []conversion, *big.Int, *Response) {
	total := totalSpends{}
	var conversions []conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionIncluded := false

	if !data.CoinToBuy.IsBaseCoin() {
		coin := context.Coins().GetCoin(data.CoinToBuy)

		if errResp := CheckForCoinSupplyOverflow(coin, data.ValueToBuy); errResp != nil {
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
				Info: EncodeError(code.NewMaximumValueToSellReached(data.MaximumValueToSell.String(), value.String(), coin.GetFullSymbol(), coin.ID().String())),
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
					Info: EncodeError(code.NewCoinReserveNotSufficient(
						coin.GetFullSymbol(),
						coin.ID().String(),
						coin.Reserve().String(),
						commissionInBaseCoin.String(),
					)),
				}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coin.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, conversion{
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
				Info: EncodeError(code.NewMaximumValueToSellReached(data.MaximumValueToSell.String(), value.String(), coin.GetFullSymbol(), coin.ID().String())),
			}
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, conversion{
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
				Info: EncodeError(code.NewCoinReserveNotSufficient(
					coinFrom.GetFullSymbol(),
					coinFrom.ID().String(),
					coinFrom.Reserve().String(),
					commissionInBaseCoin.String(),
				)),
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
					Info: EncodeError(code.NewCoinReserveNotSufficient(
						coinTo.GetFullSymbol(),
						coinTo.ID().String(),
						coinTo.Reserve().String(),
						commissionInBaseCoin.String(),
					)),
				}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinTo.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, conversion{
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
				Info: EncodeError(code.NewMaximumValueToSellReached(data.MaximumValueToSell.String(), value.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
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
					Info: EncodeError(code.NewCoinReserveNotSufficient(
						coinFrom.GetFullSymbol(),
						coinFrom.ID().String(),
						coinFrom.Reserve().String(),
						commissionInBaseCoin.String(),
					)),
				}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinFrom.Crr(), commissionInBaseCoin) // todo CalculateCommission

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, conversion{
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
					Info: EncodeError(code.NewMaximumValueToSellReached(data.MaximumValueToSell.String(), value.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
				}
			}
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, conversion{
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
					Info: EncodeError(code.NewCoinReserveNotSufficient(
						coin.GetFullSymbol(),
						coin.ID().String(),
						coin.Reserve().String(),
						commissionInBaseCoin.String(),
					)),
				}
			}

			commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
			conversions = append(conversions, conversion{
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

func (data BuyCoinData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.ValueToBuy == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}

	}

	coinToSell := context.Coins().GetCoin(data.CoinToSell)
	if coinToSell == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin to sell not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.CoinToSell.String())),
		}
	}

	if !coinToSell.BaseOrHasReserve() {
		return &Response{
			Code: code.CoinHasNotReserve,
			Log:  "sell coin has not reserve",
			Info: EncodeError(code.NewCoinHasNotReserve(
				coinToSell.GetFullSymbol(),
				coinToSell.ID().String(),
			)),
		}
	}

	coinToBuy := context.Coins().GetCoin(data.CoinToBuy)
	if coinToBuy == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin to buy not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.CoinToBuy.String())),
		}
	}

	if !coinToBuy.BaseOrHasReserve() {
		return &Response{
			Code: code.CoinHasNotReserve,
			Log:  "buy coin has not reserve",
			Info: EncodeError(code.NewCoinHasNotReserve(
				coinToBuy.GetFullSymbol(),
				coinToBuy.ID().String(),
			)),
		}
	}

	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.CoinToSell.String(),
				coinToSell.GetFullSymbol(),
				data.CoinToBuy.String(),
				coinToBuy.GetFullSymbol()),
			),
		}
	}

	return nil
}

func (data BuyCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()
	var errResp *Response
	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	gasCoinEdited := dummyCoin{
		id:         gasCoin.ID(),
		volume:     gasCoin.Volume(),
		reserve:    gasCoin.Reserve(),
		crr:        gasCoin.Crr(),
		fullSymbol: gasCoin.GetFullSymbol(),
	}
	coinToSell := data.CoinToSell
	coinToBuy := data.CoinToBuy
	coinFrom := checkState.Coins().GetCoin(coinToSell)
	coinTo := checkState.Coins().GetCoin(coinToBuy)
	value := big.NewInt(0).Set(data.ValueToBuy)
	if !coinToBuy.IsBaseCoin() {
		if errResp = CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return *errResp
		}
		value = formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
		if coinToBuy == gasCoinEdited.ID() {
			gasCoinEdited.volume.Add(gasCoinEdited.volume, data.ValueToBuy)
			gasCoinEdited.reserve.Add(gasCoinEdited.reserve, value)
		}
	}
	diffBipReserve := big.NewInt(0).Set(value)
	if !coinToSell.IsBaseCoin() {
		value, errResp = CalculateSaleAmountAndCheck(coinFrom, value)
		if errResp != nil {
			return *errResp
		}
		if coinToSell == gasCoinEdited.ID() {
			gasCoinEdited.volume.Sub(gasCoinEdited.volume, value)
			gasCoinEdited.reserve.Sub(gasCoinEdited.reserve, diffBipReserve)
		}
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, gasCoinEdited, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	spendInGasCoin := big.NewInt(0).Set(commission)
	if tx.GasCoin != coinToSell {
		if value.Cmp(data.MaximumValueToSell) == 1 {
			return Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					data.MaximumValueToSell.String(), value.String()),
				Info: EncodeError(code.NewMaximumValueToSellReached(data.MaximumValueToSell.String(), value.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
			}
		}
		if checkState.Accounts().GetBalance(sender, data.CoinToSell).Cmp(value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value.String(), coinFrom.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), value.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
			}
		}
	} else {
		spendInGasCoin.Add(spendInGasCoin, value)
	}
	if spendInGasCoin.Cmp(data.MaximumValueToSell) == 1 {
		return Response{
			Code: code.MaximumValueToSellReached,
			Log: fmt.Sprintf(
				"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
				data.MaximumValueToSell.String(), spendInGasCoin.String()),
			Info: EncodeError(code.NewMaximumValueToSellReached(data.MaximumValueToSell.String(), spendInGasCoin.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
		}
	}
	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(spendInGasCoin) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), spendInGasCoin.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), spendInGasCoin.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if !isGasCommissionFromPoolSwap {
		if gasCoin.ID() == coinToSell {
			// todo check reserve
		}
		if gasCoin.ID() == coinToBuy {

		}
	}

	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, value)
		if !data.CoinToSell.IsBaseCoin() {
			deliverState.Coins.SubVolume(data.CoinToSell, value)
			deliverState.Coins.SubReserve(data.CoinToSell, diffBipReserve)
		}
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, data.ValueToBuy)
		if !data.CoinToBuy.IsBaseCoin() {
			deliverState.Coins.AddVolume(data.CoinToBuy, data.ValueToBuy)
			deliverState.Coins.AddReserve(data.CoinToBuy, diffBipReserve)
		}
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

func CalculateSaleAmountAndCheck(coinFrom calculateCoin, value *big.Int) (*big.Int, *Response) {
	if coinFrom.Reserve().Cmp(value) < 0 {
		return nil, &Response{
			Code: code.CoinReserveNotSufficient,
			Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
				coinFrom.Reserve().String(),
				types.GetBaseCoin(),
				value.String(),
				types.GetBaseCoin()),
			Info: EncodeError(code.NewCoinReserveNotSufficient(
				coinFrom.GetFullSymbol(),
				coinFrom.ID().String(),
				coinFrom.Reserve().String(),
				value.String(),
			)),
		}
	}
	value = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), value)
	if coinFrom.ID().IsBaseCoin() {
		return value, nil
	}

	if errResp := CheckReserveUnderflow(coinFrom, value); errResp != nil {
		return nil, errResp
	}

	return value, nil
}

func checkConversionsReserveUnderflow(conversions []conversion, context *state.CheckState) *Response {
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
