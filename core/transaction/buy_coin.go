package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type BuyCoinData struct {
	CoinToBuy          types.CoinID
	ValueToBuy         *big.Int
	CoinToSell         types.CoinID
	MaximumValueToSell *big.Int
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
	value := big.NewInt(0).Set(data.ValueToBuy)
	if !coinToBuy.IsBaseCoin() {
		if errResp = CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return *errResp
		}
		value = formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
		if coinToBuy == gasCoinUpdated.ID() {
			gasCoinUpdated.volume.Add(gasCoinUpdated.volume, data.ValueToBuy)
			gasCoinUpdated.reserve.Add(gasCoinUpdated.reserve, value)
		}
	}
	diffBipReserve := big.NewInt(0).Set(value)
	if !coinToSell.IsBaseCoin() {
		value, errResp = CalculateSaleAmountAndCheck(coinFrom, value)
		if errResp != nil {
			return *errResp
		}
		if coinToSell == gasCoinUpdated.ID() {
			gasCoinUpdated.volume.Sub(gasCoinUpdated.volume, value)
			gasCoinUpdated.reserve.Sub(gasCoinUpdated.reserve, diffBipReserve)
		}
	}

	commissionInBaseCoin := tx.Commission(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoinUpdated, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if isGasCommissionFromPoolSwap != true && gasCoin.ID() == coinToSell && !coinToSell.IsBaseCoin() {
		commission, errResp = CalculateSaleAmountAndCheck(coinFrom, big.NewInt(0).Add(diffBipReserve, commissionInBaseCoin))
		if errResp != nil {
			return *errResp
		}
		commission.Sub(commission, value)
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
	var tags []abcTypes.EventAttribute
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

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
			{Key: []byte("tx.return"), Value: []byte(value.String())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

func CalculateSaleAmountAndCheck(coinFrom calculateCoin, value *big.Int) (*big.Int, *Response) {
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

func CalculateSaleReturnAndCheck(coinFrom calculateCoin, value *big.Int) (*big.Int, *Response) {
	value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), value)
	if coinFrom.Reserve().Cmp(value) != 1 {
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
