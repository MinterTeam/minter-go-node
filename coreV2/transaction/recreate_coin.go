package transaction

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type RecreateCoinData struct {
	Name                 string
	Symbol               types.CoinSymbol
	InitialAmount        *big.Int
	InitialReserve       *big.Int
	ConstantReserveRatio uint32
	MaxSupply            *big.Int
}

func (data RecreateCoinData) Gas() int64 {
	return gasRecreateCoin
}
func (data RecreateCoinData) TxType() TxType {
	return TypeRecreateCoin
}

func (data RecreateCoinData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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
	return price.RecreateCoin
}

func (data RecreateCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  map[types.Address]*big.Int
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.GasCoin,
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  make([]*OrderDetail, 0, len(ownersCom)),
			}
			for address, value := range ownersCom {
				deliverState.Accounts.AddBalance(address, tx.GasCoin, value)
				tagsCom.Sellers = append(tagsCom.Sellers, &OrderDetail{Owner: address, Value: value.String()})
			}
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SubBalance(sender, types.GetBaseCoinID(), data.InitialReserve)

		oldCoinID := checkState.Coins().GetCoinBySymbol(data.Symbol, 0).ID()
		coinId := checkState.App().GetNextCoinID()

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

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.coin_symbol"), Value: []byte(data.Symbol.String()), Index: true},
			{Key: []byte("tx.coin_id"), Value: []byte(coinId.String()), Index: true},
			{Key: []byte("tx.old_coin_symbol"), Value: []byte(checkState.Coins().GetCoin(oldCoinID).GetFullSymbol())},
			{Key: []byte("tx.old_coin_id"), Value: []byte(oldCoinID.String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
