package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type CreateSwapPoolData struct {
	Coin0   types.CoinID
	Coin1   types.CoinID
	Volume0 *big.Int
	Volume1 *big.Int
}

func (data CreateSwapPoolData) Gas() int64 {
	return gasCreateSwapPool
}
func (data CreateSwapPoolData) TxType() TxType {
	return TypeCreateSwapPool
}

func (data CreateSwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Coin1 == data.Coin0 {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.Coin0.String(),
				data.Coin1.String(), "", "")),
		}
	}

	if context.Swap().SwapPoolExist(data.Coin0, data.Coin1) {
		return &Response{
			Code: code.PairAlreadyExists,
			Log:  "swap pool already exist",
			Info: EncodeError(code.NewPairAlreadyExists(
				data.Coin0.String(),
				data.Coin1.String())),
		}
	}

	coin0 := context.Coins().GetCoin(data.Coin0)
	if coin0 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin0.String())),
		}
	}

	coin1 := context.Coins().GetCoin(data.Coin1)
	if coin1 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin1.String())),
		}
	}

	return nil
}

func (data CreateSwapPoolData) String() string {
	return fmt.Sprintf("CREATE SWAP POOL")
}

func (data CreateSwapPoolData) CommissionData(price *commission.Price) *big.Int {
	return price.CreateSwapPool
}

func (data CreateSwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	if err := checkState.Swap().GetSwapper(data.Coin0, data.Coin1).CheckCreate(data.Volume0, data.Volume1); err != nil {
		if err == swap.ErrorInsufficientLiquidityMinted {
			return Response{
				Code: code.InsufficientLiquidityMinted,
				Log: fmt.Sprintf("You wanted to add less than minimum liquidity, you should add %s %s and %s or more %s",
					"10", checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), "10", checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientLiquidityMinted(data.Coin0.String(), "10", data.Coin1.String(), "10")),
			}
		}
	}
	{
		amount0 := new(big.Int).Set(data.Volume0)
		if tx.GasCoin == data.Coin0 {
			amount0.Add(amount0, commission)
		}
		if checkState.Accounts().GetBalance(sender, data.Coin0).Cmp(amount0) == -1 {
			symbol := checkState.Coins().GetCoin(data.Coin0).GetFullSymbol()
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amount0.String(), symbol),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), amount0.String(), symbol, data.Coin0.String())),
			}
		}
	}

	{
		totalAmount1 := new(big.Int).Set(data.Volume1)
		if tx.GasCoin == data.Coin1 {
			totalAmount1.Add(totalAmount1, commission)
		}
		if checkState.Accounts().GetBalance(sender, data.Coin1).Cmp(totalAmount1) == -1 {
			symbol := checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalAmount1.String(), symbol),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), totalAmount1.String(), symbol, data.Coin1.String())),
			}
		}
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) == -1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
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

		amount0, amount1, liquidity, id := deliverState.Swap.PairCreate(data.Coin0, data.Coin1, data.Volume0, data.Volume1)

		deliverState.Accounts.SubBalance(sender, data.Coin0, amount0)
		deliverState.Accounts.SubBalance(sender, data.Coin1, amount1)

		coins := liquidityCoinName(data.Coin0, data.Coin1)
		coinID := checkState.App().GetNextCoinID()

		liquidityCoinSymbol := LiquidityCoinSymbol(id)
		deliverState.Coins.CreateToken(coinID, liquidityCoinSymbol, "Liquidity Pool "+coins, true, true, big.NewInt(0).Set(liquidity), maxCoinSupply, nil)
		deliverState.Accounts.AddBalance(sender, coinID, liquidity.Sub(liquidity, swap.Bound))
		deliverState.Accounts.AddBalance(types.Address{}, coinID, swap.Bound)

		deliverState.App.SetCoinsCount(coinID.Uint32())

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.volume1"), Value: []byte(data.Volume1.String())},
			{Key: []byte("tx.liquidity"), Value: []byte(liquidity.String())},
			{Key: []byte("tx.pool_token"), Value: []byte(liquidityCoinSymbol.String()), Index: true},
			{Key: []byte("tx.pool_token_id"), Value: []byte(coinID.String()), Index: true},
			{Key: []byte("tx.pair_ids"), Value: []byte(coins), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

func LiquidityCoinSymbol(id uint32) types.CoinSymbol {
	return types.StrToCoinSymbol(fmt.Sprintf("LP-%d", id))
}
