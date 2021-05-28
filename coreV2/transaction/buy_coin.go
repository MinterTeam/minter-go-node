package transaction

import (
	"fmt"
	"math/big"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/formula"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type BuyCoinData struct {
	CoinToBuy          types.CoinID
	ValueToBuy         *big.Int
	CoinToSell         types.CoinID
	MaximumValueToSell *big.Int
}

func (data BuyCoinData) Gas() int64 {
	return gasBuyCoin
}
func (data BuyCoinData) TxType() TxType {
	return TypeBuyCoin
}

func (data BuyCoinData) String() string {
	return fmt.Sprintf("BUY COIN sell:%s buy:%s %s",
		data.CoinToSell.String(), data.ValueToBuy.String(), data.CoinToBuy.String())
}

func (data BuyCoinData) CommissionData(price *commission.Price) *big.Int {
	return price.BuyBancor
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
			Log:  "sell coin has no reserve",
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
			Log:  "coin to buy has no reserve",
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

func (data BuyCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := tx.Commission(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	coinToSell := data.CoinToSell
	coinToBuy := data.CoinToBuy
	var coinFrom, coinTo CalculateCoin
	coinFrom = checkState.Coins().GetCoin(coinToSell)
	coinTo = checkState.Coins().GetCoin(coinToBuy)

	value := big.NewInt(0).Set(data.ValueToBuy)

	if isGasCommissionFromPoolSwap == false && !tx.GasCoin.IsBaseCoin() {
		if tx.GasCoin == data.CoinToSell {
			coinFrom = DummyCoin{
				id:         gasCoin.ID(),
				volume:     big.NewInt(0).Sub(gasCoin.Volume(), commission),
				reserve:    big.NewInt(0).Sub(gasCoin.Reserve(), commissionInBaseCoin),
				crr:        gasCoin.Crr(),
				fullSymbol: gasCoin.GetFullSymbol(),
				maxSupply:  gasCoin.MaxSupply(),
			}
		} else if tx.GasCoin == data.CoinToBuy {
			coinTo = DummyCoin{
				id:         gasCoin.ID(),
				volume:     big.NewInt(0).Sub(gasCoin.Volume(), commission),
				reserve:    big.NewInt(0).Sub(gasCoin.Reserve(), commissionInBaseCoin),
				crr:        gasCoin.Crr(),
				fullSymbol: gasCoin.GetFullSymbol(),
				maxSupply:  gasCoin.MaxSupply(),
			}
		}
	}

	if !coinToBuy.IsBaseCoin() {
		if errResp = CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return *errResp
		}
		value = formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
	}
	diffBipReserve := big.NewInt(0).Set(value)
	if !coinToSell.IsBaseCoin() {
		value, errResp = CalculateSaleAmountAndCheck(coinFrom, value)
		if errResp != nil {
			return *errResp
		}
	}

	valueToSell := big.NewInt(0).Set(value)
	if valueToSell.Cmp(data.MaximumValueToSell) == 1 {
		return Response{
			Code: code.MaximumValueToSellReached,
			Log: fmt.Sprintf(
				"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
				data.MaximumValueToSell.String(), valueToSell.String()),
			Info: EncodeError(code.NewMaximumValueToSellReached(data.MaximumValueToSell.String(), valueToSell.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
		}
	}

	spendInGasCoin := big.NewInt(0).Set(commission)
	if tx.GasCoin == coinToSell {
		spendInGasCoin.Add(spendInGasCoin, value)
	} else {
		if checkState.Accounts().GetBalance(sender, data.CoinToSell).Cmp(value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value.String(), coinFrom.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), value.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
			}
		}
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(spendInGasCoin) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), spendInGasCoin.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), spendInGasCoin.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}
	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _, _, _ = deliverState.Swap.PairSellWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
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

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String()), Index: true},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String()), Index: true},
			{Key: []byte("tx.return"), Value: []byte(value.String())},
			{Key: []byte("tx.reserve"), Value: []byte(diffBipReserve.String())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

func CalculateSaleAmountAndCheck(coinFrom CalculateCoin, value *big.Int) (*big.Int, *Response) {
	if coinFrom.Reserve().Cmp(value) == -1 {
		return nil, &Response{
			Code: code.CoinReserveNotSufficient,
			Log: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
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

func CalculateSaleReturnAndCheck(coinFrom CalculateCoin, value *big.Int) (*big.Int, *Response) {
	if coinFrom.Volume().Cmp(value) == -1 {
		return nil, &Response{
			Code: code.CoinReserveNotSufficient,
			Log: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
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
	value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), value)
	if errResp := CheckReserveUnderflow(coinFrom, value); errResp != nil {
		return nil, errResp
	}
	return value, nil
}
