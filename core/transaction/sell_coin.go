package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
	"strconv"
)

type SellCoinData struct {
	CoinToSell        types.CoinID
	ValueToSell       *big.Int
	CoinToBuy         types.CoinID
	MinimumValueToBuy *big.Int
}

func (data SellCoinData) TxType() TxType {
	return TypeSellCoin
}

func (data SellCoinData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.ValueToSell == nil {
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

func (data SellCoinData) String() string {
	return fmt.Sprintf("SELL COIN sell:%s %s buy:%s",
		data.ValueToSell.String(), data.CoinToBuy.String(), data.CoinToSell.String())
}

func (data SellCoinData) CommissionData(price *commission.Price) *big.Int {
	return price.Convert
}

func (data SellCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int, gas int64) Response {
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
	gasCoinUpdated := dummyCoin{
		id:         gasCoin.ID(),
		volume:     gasCoin.Volume(),
		reserve:    gasCoin.Reserve(),
		crr:        gasCoin.Crr(),
		fullSymbol: gasCoin.GetFullSymbol(),
		maxSupply:  gasCoin.MaxSupply(),
	}
	coinToSell := data.CoinToSell
	coinToBuy := data.CoinToBuy
	coinFrom := checkState.Coins().GetCoin(coinToSell)
	coinTo := checkState.Coins().GetCoin(coinToBuy)

	value := big.NewInt(0).Set(data.ValueToSell)
	if !coinToSell.IsBaseCoin() {
		value, errResp = CalculateSaleReturnAndCheck(coinFrom, value)
		if errResp != nil {
			return *errResp
		}
		if coinToSell == gasCoinUpdated.ID() {
			gasCoinUpdated.volume.Sub(gasCoinUpdated.volume, data.ValueToSell)
			gasCoinUpdated.reserve.Sub(gasCoinUpdated.reserve, value)
		}
	}
	diffBipReserve := big.NewInt(0).Set(value)
	if !coinToBuy.IsBaseCoin() {
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
		if errResp := CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return *errResp
		}
		if coinToBuy == gasCoinUpdated.ID() {
			gasCoinUpdated.volume.Add(gasCoinUpdated.volume, value)
			gasCoinUpdated.reserve.Add(gasCoinUpdated.reserve, diffBipReserve)
		}
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoinUpdated, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	// if !isGasCommissionFromPoolSwap && gasCoin.ID() == coinToSell && !coinToSell.IsBaseCoin() {
	// 	// commission = formula.CalculateSaleAmount(gasCoinUpdated.Volume(), gasCoinUpdated.Reserve(), coinFrom.Crr(), commissionInBaseCoin)
	// 	value, errResp = CalculateSaleReturnAndCheck(coinFrom, big.NewInt(0).Add(data.ValueToSell, commission))
	// 	if errResp != nil {
	// 		return *errResp
	// 	}
	//
	// 	if !coinToBuy.IsBaseCoin() {
	// 		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
	// 		if errResp := CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
	// 			return *errResp
	// 		}
	// 	}
	// 	commission.Sub(commission, value)
	// }

	spendInGasCoin := big.NewInt(0).Set(commission)
	if tx.GasCoin != coinToSell {
		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return Response{
				Code: code.MinimumValueToBuyReached,
				Log: fmt.Sprintf(
					"You wanted to buy minimum %s, but currently you need to spend %s to complete tx",
					data.MinimumValueToBuy.String(), value.String()),
				Info: EncodeError(code.NewMaximumValueToSellReached(data.MinimumValueToBuy.String(), value.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
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
		spendInGasCoin.Add(spendInGasCoin, data.ValueToSell)
	}
	if spendInGasCoin.Cmp(data.MinimumValueToBuy) == -1 {
		return Response{
			Code: code.MinimumValueToBuyReached,
			Log: fmt.Sprintf(
				"You wanted to buy minimum %s, but currently you need to spend %s to complete tx",
				data.MinimumValueToBuy.String(), spendInGasCoin.String()),
			Info: EncodeError(code.NewMaximumValueToSellReached(data.MinimumValueToBuy.String(), spendInGasCoin.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
		}
	}
	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(spendInGasCoin) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), spendInGasCoin.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), spendInGasCoin.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
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
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, data.ValueToSell)
		if !data.CoinToSell.IsBaseCoin() {
			deliverState.Coins.SubVolume(data.CoinToSell, data.ValueToSell)
			deliverState.Coins.SubReserve(data.CoinToSell, diffBipReserve)
		}
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, value)
		if !data.CoinToBuy.IsBaseCoin() {
			deliverState.Coins.AddVolume(data.CoinToBuy, value)
			deliverState.Coins.AddReserve(data.CoinToBuy, diffBipReserve)
		}
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.gas"), Value: []byte(strconv.Itoa(int(gas)))},
		kv.Pair{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSellCoin)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		kv.Pair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		kv.Pair{Key: []byte("tx.return"), Value: []byte(value.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   gas,
		GasWanted: gas,
	}
}
