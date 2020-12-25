package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
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
	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.CoinToSell.String(), "",
				data.CoinToBuy.String(), "")),
		}
	}
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
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	balance := checkState.Accounts().GetBalance(sender, data.CoinToSell)
	if tx.GasCoin == data.CoinToSell {
		balance.Sub(balance, commission)
	}
	if balance.Cmp(commission) != 1 {
		symbol := checkState.Coins().GetCoin(data.CoinToSell).GetFullSymbol()
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), balance.String(), symbol),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), balance.String(), symbol, data.CoinToSell.String())),
		}
	}
	swapper := checkState.Swap().GetSwapper(data.CoinToSell, data.CoinToBuy)
	if isGasCommissionFromPoolSwap && (tx.GasCoin == data.CoinToSell && data.CoinToBuy.IsBaseCoin()) {
		swapper = commissionPoolSwapper.AddLastSwapStep(commission, commissionInBaseCoin)
	}

	errResp = CheckSwap(swapper, checkState.Coins().GetCoin(data.CoinToSell), checkState.Coins().GetCoin(data.CoinToBuy), balance, data.MinimumValueToBuy, false)
	if errResp != nil {
		return *errResp
	}

	amountIn, amountOut := balance, data.MinimumValueToBuy
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		amountIn, amountOut = deliverState.Swap.PairSell(data.CoinToSell, data.CoinToBuy, balance, data.MinimumValueToBuy)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, amountIn)
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, amountOut)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
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

type dummyCoin struct {
	id         types.CoinID
	volume     *big.Int
	reserve    *big.Int
	crr        uint32
	fullSymbol string
}

func (m dummyCoin) ID() types.CoinID {
	return m.id
}

func (m dummyCoin) BaseOrHasReserve() bool {
	return m.ID().IsBaseCoin() || (m.Crr() > 0 && m.Reserve().Sign() == 1)
}

func (m dummyCoin) Volume() *big.Int {
	return m.volume
}

func (m dummyCoin) Reserve() *big.Int {
	return m.reserve
}

func (m dummyCoin) Crr() uint32 {
	return m.crr
}

func (m dummyCoin) GetFullSymbol() string {
	return m.fullSymbol
}

type calculateCoin interface {
	ID() types.CoinID
	BaseOrHasReserve() bool
	Volume() *big.Int
	Reserve() *big.Int
	Crr() uint32
	GetFullSymbol() string
}

func CalculateCommission(checkState *state.CheckState, swapper swap.SwapChecker, gasCoin calculateCoin, commissionInBaseCoin *big.Int) (commission *big.Int, poolSwap bool, errResp *Response) {
	if gasCoin.ID().IsBaseCoin() {
		return new(big.Int).Set(commissionInBaseCoin), false, nil
	}
	commissionFromPool, responseFromPool := commissionFromPool(swapper, gasCoin, checkState.Coins().GetCoin(types.BasecoinID), commissionInBaseCoin)
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

func commissionFromPool(swapChecker swap.SwapChecker, coin calculateCoin, baseCoin calculateCoin, commissionInBaseCoin *big.Int) (*big.Int, *Response) {
	if !swapChecker.IsExist() {
		return nil, &Response{
			Code: code.PairNotExists,
			Log:  fmt.Sprintf("swap pair beetwen coins %s and %s not exists in pool", coin.GetFullSymbol(), types.GetBaseCoin()),
			Info: EncodeError(code.NewPairNotExists(coin.ID().String(), types.GetBaseCoinID().String())),
		}
	}
	commission := swapChecker.CalculateSellForBuy(commissionInBaseCoin)
	if errResp := CheckSwap(swapChecker, coin, baseCoin, commission, commissionInBaseCoin, true); errResp != nil {
		return nil, errResp
	}
	return commission, nil
}

func commissionFromReserve(gasCoin calculateCoin, commissionInBaseCoin *big.Int) (*big.Int, *Response) {
	if !gasCoin.BaseOrHasReserve() {
		return nil, &Response{
			Code: code.CoinHasNotReserve,
			Log:  "Gas coin has not reserve",
		}
	}
	errResp := CheckReserveUnderflow(gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return nil, errResp
	}

	return formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin), nil
}
