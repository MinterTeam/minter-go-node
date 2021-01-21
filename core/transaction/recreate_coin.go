package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
)

type RecreateCoinData struct {
	Name                 string
	Symbol               types.CoinSymbol
	InitialAmount        *big.Int
	InitialReserve       *big.Int
	ConstantReserveRatio uint32
	MaxSupply            *big.Int
}

func (data RecreateCoinData) TxType() TxType {
	return TypeRecreateCoin
}

func (data RecreateCoinData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.InitialReserve == nil || data.InitialAmount == nil || data.MaxSupply == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if len(data.Name) > maxCoinNameBytes {
		return &Response{
			Code: code.InvalidCoinName,
			Log:  fmt.Sprintf("Coin name is invalid. Allowed up to %d bytes.", maxCoinNameBytes),
			Info: EncodeError(code.NewInvalidCoinName(strconv.Itoa(maxCoinNameBytes), strconv.Itoa(len(data.Name)))),
		}
	}

	if data.InitialAmount.Cmp(minCoinSupply) == -1 || data.InitialAmount.Cmp(data.MaxSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin supply should be between %s and %s", minCoinSupply.String(), data.MaxSupply.String()),
			Info: EncodeError(code.NewWrongCoinSupply(minCoinSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), minCoinReserve.String(), data.InitialReserve.String(), data.InitialAmount.String())),
		}
	}

	if data.MaxSupply.Cmp(maxCoinSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Max coin supply should be less than %s", maxCoinSupply),
			Info: EncodeError(code.NewWrongCoinSupply(minCoinSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), minCoinReserve.String(), data.InitialReserve.String(), data.InitialAmount.String())),
		}
	}

	if data.InitialReserve.Cmp(minCoinReserve) == -1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin reserve should be greater than or equal to %s", minCoinReserve.String()),
			Info: EncodeError(map[string]string{
				"code":                    strconv.Itoa(int(code.WrongCoinSupply)),
				"min_initial_reserve":     minCoinReserve.String(),
				"current_initial_reserve": data.InitialReserve.String(),
			})}
	}
	if data.ConstantReserveRatio < 10 || data.ConstantReserveRatio > 100 {
		return &Response{
			Code: code.WrongCrr,
			Log:  "Constant Reserve Ratio should be between 10 and 100",
			Info: EncodeError(code.NewWrongCrr("10", "100", strconv.Itoa(int(data.ConstantReserveRatio)))),
		}
	}
	if data.InitialReserve.Cmp(minCoinReserve) == -1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin reserve should be greater than or equal to %s", minCoinReserve.String()),
			Info: EncodeError(code.NewWrongCoinSupply(minCoinSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), minCoinReserve.String(), data.InitialReserve.String(), data.InitialAmount.String())),
		}
	}

	sender, _ := tx.Sender()

	coin := context.Coins().GetCoinBySymbol(data.Symbol, 0)
	if coin == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Symbol),
			Info: EncodeError(code.NewCoinNotExists(data.Symbol.String(), "")),
		}
	}

	symbolInfo := context.Coins().GetSymbolInfo(coin.Symbol())
	if symbolInfo == nil || symbolInfo.OwnerAddress() == nil || *symbolInfo.OwnerAddress() != sender {
		var owner *string
		if symbolInfo != nil && symbolInfo.OwnerAddress() != nil {
			own := symbolInfo.OwnerAddress().String()
			owner = &own
		}
		return &Response{
			Code: code.IsNotOwnerOfCoin,
			Log:  "Sender is not owner of coin",
			Info: EncodeError(code.NewIsNotOwnerOfCoin(data.Symbol.String(), owner)),
		}
	}

	return nil
}

func (data RecreateCoinData) String() string {
	return fmt.Sprintf("RECREATE COIN symbol:%s reserve:%s amount:%s crr:%d",
		data.Symbol.String(), data.InitialReserve, data.InitialAmount, data.ConstantReserveRatio)
}

func (data RecreateCoinData) CommissionData(price *commission.Price) *big.Int {
	return price.Recreate
}

func (data RecreateCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int, gas int64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.Commission(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if checkState.Accounts().GetBalance(sender, types.GetBaseCoinID()).Cmp(data.InitialReserve) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), data.InitialReserve.String(), types.GetBaseCoin()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), data.InitialReserve.String(), types.GetBaseCoin().String(), types.GetBaseCoinID().String())),
		}
	}

	if tx.GasCoin.IsBaseCoin() {
		gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

		totalTxCost := big.NewInt(0)
		totalTxCost.Add(totalTxCost, data.InitialReserve)
		totalTxCost.Add(totalTxCost, commission)

		if checkState.Accounts().GetBalance(sender, types.GetBaseCoinID()).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), gasCoin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), totalTxCost.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
			}
		}
	}
	oldCoinID := checkState.Coins().GetCoinBySymbol(data.Symbol, 0).ID()
	var coinId = checkState.App().GetNextCoinID()
	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SubBalance(sender, types.GetBaseCoinID(), data.InitialReserve)

		deliverState.Coins.Recreate(
			coinId,
			data.Name,
			data.Symbol,
			data.InitialAmount,
			data.ConstantReserveRatio,
			data.InitialReserve,
			data.MaxSupply,
		)

		deliverState.App.SetCoinsCount(coinId.Uint32())
		deliverState.Accounts.AddBalance(sender, coinId, data.InitialAmount)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.gas"), Value: []byte(strconv.Itoa(int(gas)))},
		kv.Pair{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeRecreateCoin)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_symbol"), Value: []byte(data.Symbol.String())},
		kv.Pair{Key: []byte("tx.coin_id"), Value: []byte(coinId.String())},
		kv.Pair{Key: []byte("tx.old_coin_symbol"), Value: []byte(checkState.Coins().GetCoin(oldCoinID).GetFullSymbol())},
		kv.Pair{Key: []byte("tx.old_coin_id"), Value: []byte(oldCoinID.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   gas,
		GasWanted: gas,
	}
}
