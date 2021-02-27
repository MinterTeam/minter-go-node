package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/formula"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type SellCoinData struct {
	CoinToSell        types.CoinID
	ValueToSell       *big.Int
	CoinToBuy         types.CoinID
	MinimumValueToBuy *big.Int
}

func (data SellCoinData) Gas() int {
	return gasSellCoin
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

func (data SellCoinData) String() string {
	return fmt.Sprintf("SELL COIN sell:%s %s buy:%s",
		data.ValueToSell.String(), data.CoinToBuy.String(), data.CoinToSell.String())
}

func (data SellCoinData) CommissionData(price *commission.Price) *big.Int {
	return price.SellBancor
}

func (data SellCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	value := big.NewInt(0).Set(data.ValueToSell)

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

	if !coinToSell.IsBaseCoin() {
		value, errResp = CalculateSaleReturnAndCheck(coinFrom, value)
		if errResp != nil {
			return *errResp
		}
	}
	diffBipReserve := big.NewInt(0).Set(value)
	if !coinToBuy.IsBaseCoin() {
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
		if errResp := CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return *errResp
		}
	}

	if value.Cmp(data.MinimumValueToBuy) == -1 {
		return Response{
			Code: code.MinimumValueToBuyReached,
			Log: fmt.Sprintf(
				"You wanted to buy minimum %s, but currently you need to spend %s to complete tx",
				data.MinimumValueToBuy.String(), value.String()),
			Info: EncodeError(code.NewMaximumValueToSellReached(data.MinimumValueToBuy.String(), value.String(), coinFrom.GetFullSymbol(), coinFrom.ID().String())),
		}
	}

	spendInGasCoin := big.NewInt(0).Set(commission)
	if tx.GasCoin == coinToSell {
		spendInGasCoin.Add(spendInGasCoin, data.ValueToSell)
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
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
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

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String()), Index: true},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String()), Index: true},
			{Key: []byte("tx.return"), Value: []byte(value.String())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
