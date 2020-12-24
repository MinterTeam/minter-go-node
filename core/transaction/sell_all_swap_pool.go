package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type SellAllSwapPoolData struct {
	CoinToSell        types.CoinID
	CoinToBuy         types.CoinID
	MinimumValueToBuy *big.Int
}

func (data SellAllSwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if !context.Swap().SwapPoolExist(data.CoinToSell, data.CoinToBuy) {
		return &Response{
			Code: code.PairNotExists,
			Log:  "swap pool not found",
		}
	}
	return nil
}

func (data SellAllSwapPoolData) String() string {
	return fmt.Sprintf("SWAP POOL SELL ALL")
}

func (data SellAllSwapPoolData) Gas() int64 {
	return commissions.ConvertTx
}

func (data SellAllSwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	balance := checkState.Accounts().GetBalance(sender, data.CoinToSell)
	if tx.GasCoin == data.CoinToSell {
		balance.Sub(balance, commission)
	} else if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}
	if balance.Sign() != 1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	errResp = CheckSwap(checkState, data.CoinToSell, balance, data.CoinToBuy, data.MinimumValueToBuy, false)
	if errResp != nil {
		return *errResp
	}

	amountIn, amountOut := balance, data.MinimumValueToBuy
	if deliverState, ok := context.(*state.State); ok {
		amountIn, amountOut = deliverState.Swap.PairSell(data.CoinToSell, data.CoinToBuy, balance, data.MinimumValueToBuy)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, amountIn)
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, amountOut)

		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSellAllSwapPool)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		kv.Pair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		kv.Pair{Key: []byte("tx.return"), Value: []byte(amountOut.String())},
		kv.Pair{Key: []byte("tx.sell_amount"), Value: []byte(balance.String())},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}

func CalculateCommission(checkState *state.CheckState, gasCoin *coins.Model, commissionInBaseCoin *big.Int) (commission *big.Int, poolSwap bool, errResp *Response) {
	if gasCoin.ID().IsBaseCoin() {
		return new(big.Int).Set(commissionInBaseCoin), false, nil
	}
	commissionFromPool, responseFromPool := commissionFromPool(checkState, gasCoin.ID(), commissionInBaseCoin)
	commissionFromReserve, responseFromReserve := commissionFromReserve(gasCoin, commissionInBaseCoin)

	if responseFromPool != nil && responseFromReserve != nil {
		return nil, false, &Response{
			Code: code.CommissionCoinNotSufficient,
			Log:  fmt.Sprintf("Not possible to pay commission in coin %s", gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewCommissionCoinNotSufficient(responseFromPool.Log, responseFromReserve.Log)),
		}
	}

	if responseFromPool == responseFromReserve {
		if commissionFromReserve.Cmp(commissionFromPool) == -1 {
			return commissionFromReserve, false, nil
		}
		return commissionFromPool, true, nil
	}

	if responseFromPool == nil {
		return commissionFromPool, true, nil
	}

	return commissionFromReserve, false, nil
}

func commissionFromPool(checkState *state.CheckState, id types.CoinID, commissionInBaseCoin *big.Int) (*big.Int, *Response) {
	if !checkState.Swap().SwapPoolExist(id, types.GetBaseCoinID()) {
		return nil, &Response{
			Code: code.PairNotExists,
			Log:  fmt.Sprintf("swap pair beetwen coins %s and %s not exists in pool", checkState.Coins().GetCoin(id).GetFullSymbol(), types.GetBaseCoin()),
			Info: EncodeError(code.NewPairNotExists(id.String(), types.GetBaseCoinID().String())),
		}
	}
	commission, _ := checkState.Swap().PairCalculateSellForBuy(id, types.GetBaseCoinID(), commissionInBaseCoin)
	if errResp := CheckSwap(checkState, id, commission, types.GetBaseCoinID(), commissionInBaseCoin, true); errResp != nil {
		return nil, errResp
	}
	return commission, nil
}

func commissionFromReserve(gasCoin *coins.Model, commissionInBaseCoin *big.Int) (*big.Int, *Response) {
	if !gasCoin.BaseOrHasReserve() {
		return nil, &Response{
			Code: code.CoinHasNotReserve,
			Log:  "gas coin has not reserve",
		}
	}
	errResp := CheckReserveUnderflow(gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return nil, errResp
	}

	return formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin), nil
}
