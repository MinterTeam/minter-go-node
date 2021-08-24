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

type RecreateTokenData struct {
	Name          string
	Symbol        types.CoinSymbol
	InitialAmount *big.Int
	MaxSupply     *big.Int
	Mintable      bool
	Burnable      bool
}

func (data RecreateTokenData) Gas() int64 {
	return gasRecreateToken
}
func (data RecreateTokenData) TxType() TxType {
	return TypeRecreateToken
}

func (data RecreateTokenData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if len(data.Name) > maxCoinNameBytes {
		return &Response{
			Code: code.InvalidCoinName,
			Log:  fmt.Sprintf("Coin name is invalid. Allowed up to %d bytes.", maxCoinNameBytes),
			Info: EncodeError(code.NewInvalidCoinName(strconv.Itoa(maxCoinNameBytes), strconv.Itoa(len(data.Name)))),
		}
	}

	if !data.Mintable && data.InitialAmount.Cmp(data.MaxSupply) != 0 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Maximum supply cannot be more than the initial amount, if the token is not mintable"),
			Info: EncodeError(code.NewWrongCoinSupply(minTokenSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), "", "", data.InitialAmount.String())),
		}
	}

	if data.InitialAmount.Cmp(minTokenSupply) == -1 || data.InitialAmount.Cmp(data.MaxSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin amount should be between %s and %s", minTokenSupply.String(), data.MaxSupply.String()),
			Info: EncodeError(code.NewWrongCoinSupply(minTokenSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), "", "", data.InitialAmount.String())),
		}
	}

	if data.MaxSupply.Cmp(maxCoinSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Max coin supply should be less %s", maxCoinSupply.String()),
			Info: EncodeError(code.NewWrongCoinSupply(minTokenSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), "", "", data.InitialAmount.String())),
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

func (data RecreateTokenData) String() string {
	return fmt.Sprintf("RECREATE TOKEN symbol:%s emission:%s",
		data.Symbol.String(), data.MaxSupply)
}

func (data RecreateTokenData) CommissionData(price *commission.Price) *big.Int {
	return price.RecreateToken
}

func (data RecreateTokenData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := price
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {

		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  []*swap.OrderDetail
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.CommissionCoin(),
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  ownersCom,
			}
			for _, value := range ownersCom {
				deliverState.Accounts.AddBalance(value.Owner, tx.CommissionCoin(), value.ValueBigInt)
			}
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.CommissionCoin(), commission)
			deliverState.Coins.SubReserve(tx.CommissionCoin(), commissionInBaseCoin)
		}
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		oldCoinID := checkState.Coins().GetCoinBySymbol(data.Symbol, 0).ID()
		coinId := checkState.App().GetNextCoinID()
		deliverState.Coins.RecreateToken(
			coinId,
			data.Name,
			data.Symbol,
			data.Mintable,
			data.Burnable,
			data.InitialAmount,
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
