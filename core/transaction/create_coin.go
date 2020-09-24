package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tendermint/libs/kv"
)

const maxCoinNameBytes = 64
const allowedCoinSymbols = "^[A-Z0-9]{3,10}$"

var (
	minCoinSupply                      = helpers.BipToPip(big.NewInt(1))
	minCoinReserve                     = helpers.BipToPip(big.NewInt(10000))
	maxCoinSupply                      = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(15+18), nil)
	allowedCoinSymbolsRegexpCompile, _ = regexp.Compile(allowedCoinSymbols)
)

type CreateCoinData struct {
	Name                 string
	Symbol               types.CoinSymbol
	InitialAmount        *big.Int
	InitialReserve       *big.Int
	ConstantReserveRatio uint32
	MaxSupply            *big.Int
}

func (data CreateCoinData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
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

	if match := allowedCoinSymbolsRegexpCompile.MatchString(data.Symbol.String()); !match {
		return &Response{
			Code: code.InvalidCoinSymbol,
			Log:  fmt.Sprintf("Invalid coin symbol. Should be %s", allowedCoinSymbols),
			Info: EncodeError(code.NewInvalidCoinSymbol(allowedCoinSymbols, data.Symbol.String())),
		}
	}

	if context.Coins().ExistsBySymbol(data.Symbol) {
		return &Response{
			Code: code.CoinAlreadyExists,
			Log:  "Coin already exists",
			Info: EncodeError(code.NewCoinAlreadyExists(types.StrToCoinSymbol(data.Symbol.String()).String(), context.Coins().GetCoinBySymbol(data.Symbol, 0).ID().String())),
		}
	}

	if data.ConstantReserveRatio < 10 || data.ConstantReserveRatio > 100 {
		return &Response{
			Code: code.WrongCrr,
			Log:  "Constant Reserve Ratio should be between 10 and 100",
			Info: EncodeError(code.NewWrongCrr("10", "100", strconv.Itoa(int(data.ConstantReserveRatio)))),
		}
	}

	if data.MaxSupply.Cmp(maxCoinSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Max coin supply should be less than %s", maxCoinSupply),
			Info: EncodeError(code.NewWrongCoinSupply(maxCoinSupply.String(), data.MaxSupply.String(), minCoinReserve.String(), data.InitialReserve.String(), minCoinSupply.String(), data.MaxSupply.String(), data.InitialAmount.String())),
		}
	}

	if data.InitialAmount.Cmp(minCoinSupply) == -1 || data.InitialAmount.Cmp(data.MaxSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin supply should be between %s and %s", minCoinSupply.String(), data.MaxSupply.String()),
			Info: EncodeError(code.NewWrongCoinSupply(maxCoinSupply.String(), data.MaxSupply.String(), minCoinReserve.String(), data.InitialReserve.String(), minCoinSupply.String(), data.MaxSupply.String(), data.InitialAmount.String())),
		}
	}

	if data.InitialReserve.Cmp(minCoinReserve) == -1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin reserve should be greater than or equal to %s", minCoinReserve.String()),
			Info: EncodeError(code.NewWrongCoinSupply(maxCoinSupply.String(), data.MaxSupply.String(), minCoinReserve.String(), data.InitialReserve.String(), minCoinSupply.String(), data.MaxSupply.String(), data.InitialAmount.String())),
		}
	}

	return nil
}

func (data CreateCoinData) String() string {
	return fmt.Sprintf("CREATE COIN symbol:%s reserve:%s amount:%s crr:%d",
		data.Symbol.String(), data.InitialReserve, data.InitialAmount, data.ConstantReserveRatio)
}

func (data CreateCoinData) Gas() int64 {
	switch len(data.Symbol.String()) {
	case 3:
		return 1000000000 // 1mln bips
	case 4:
		return 100000000 // 100k bips
	case 5:
		return 10000000 // 10k bips
	case 6:
		return 1000000 // 1k bips
	}

	return 100000 // 100 bips
}

func (data CreateCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}
	response := data.BasicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if tx.GasCoin != types.GetBaseCoinID() {
		coin := checkState.Coins().GetCoin(tx.GasCoin)

		errResp := CheckReserveUnderflow(coin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
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
		totalTxCost := big.NewInt(0)
		totalTxCost.Add(totalTxCost, data.InitialReserve)
		totalTxCost.Add(totalTxCost, commission)

		if checkState.Accounts().GetBalance(sender, types.GetBaseCoinID()).Cmp(totalTxCost) < 0 {
			gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), gasCoin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), totalTxCost.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
			}
		}
	}

	var coinId = checkState.App().GetNextCoinID()
	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		deliverState.Coins.SubVolume(tx.GasCoin, commission)

		deliverState.Accounts.SubBalance(sender, types.GetBaseCoinID(), data.InitialReserve)
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		deliverState.Coins.Create(
			coinId,
			data.Symbol,
			data.Name,
			data.InitialAmount,
			data.ConstantReserveRatio,
			data.InitialReserve,
			data.MaxSupply,
			&sender,
		)

		deliverState.App.SetCoinsCount(coinId.Uint32())
		deliverState.Accounts.AddBalance(sender, coinId, data.InitialAmount)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeCreateCoin)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_symbol"), Value: []byte(data.Symbol.String())},
		kv.Pair{Key: []byte("tx.coin_id"), Value: []byte(coinId.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
