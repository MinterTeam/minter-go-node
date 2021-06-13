package transaction

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

const maxCoinNameBytes = 64
const allowedCoinSymbols = "^[A-Z0-9]{3,10}$"

var (
	minCoinSupply                      = helpers.BipToPip(big.NewInt(1))
	minTokenSupply                     = big.NewInt(1)
	minCoinReserve                     = helpers.BipToPip(big.NewInt(10000))
	maxCoinSupply                      = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(15+18), nil)
	allowedCoinSymbolsRegexpCompile, _ = regexp.Compile(allowedCoinSymbols)
)

func checkAllowSymbol(symbol string) bool {
	if match := allowedCoinSymbolsRegexpCompile.MatchString(symbol); !match {
		return false
	}
	if _, err := strconv.Atoi(symbol); err == nil {
		return false
	}
	return true
}

type CreateCoinData struct {
	Name                 string
	Symbol               types.CoinSymbol
	InitialAmount        *big.Int
	InitialReserve       *big.Int
	ConstantReserveRatio uint32
	MaxSupply            *big.Int
}

func (data CreateCoinData) Gas() int64 {
	return gasCreateCoin
}
func (data CreateCoinData) TxType() TxType {
	return TypeCreateCoin
}

func (data CreateCoinData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if len(data.Name) > maxCoinNameBytes {
		return &Response{
			Code: code.InvalidCoinName,
			Log:  fmt.Sprintf("Coin name is invalid. Allowed up to %d bytes.", maxCoinNameBytes),
			Info: EncodeError(code.NewInvalidCoinName(strconv.Itoa(maxCoinNameBytes), strconv.Itoa(len(data.Name)))),
		}
	}

	if !checkAllowSymbol(data.Symbol.String()) {
		return &Response{
			Code: code.InvalidCoinSymbol,
			Log:  fmt.Sprintf("Invalid coin symbol. Should be %s and must contain characters", allowedCoinSymbols),
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

	if data.MaxSupply.Cmp(maxCoinSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Max coin supply should be less than %s", maxCoinSupply),
			Info: EncodeError(code.NewWrongCoinSupply(minCoinSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), minCoinReserve.String(), data.InitialReserve.String(), data.InitialAmount.String())),
		}
	}

	if data.InitialAmount.Cmp(minCoinSupply) == -1 || data.InitialAmount.Cmp(data.MaxSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin supply should be between %s and %s", minCoinSupply.String(), data.MaxSupply.String()),
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

	return nil
}

func (data CreateCoinData) String() string {
	return fmt.Sprintf("CREATE COIN symbol:%s reserve:%s amount:%s crr:%d",
		data.Symbol.String(), data.InitialReserve, data.InitialAmount, data.ConstantReserveRatio)
}

func (data CreateCoinData) CommissionData(price *commission.Price) *big.Int {
	var createTicker *big.Int
	switch len(data.Symbol.String()) {
	case 3:
		createTicker = price.CreateTicker3
	case 4:
		createTicker = price.CreateTicker4
	case 5:
		createTicker = price.CreateTicker5
	case 6:
		createTicker = price.CreateTicker6
	default:
		createTicker = price.CreateTicker7to10
	}

	return big.NewInt(0).Add(createTicker, price.CreateCoin)
}

func (data CreateCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) == -1 {
		gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	totalTxCost := big.NewInt(0).Set(data.InitialReserve)
	if tx.GasCoin == types.GetBaseCoinID() {
		totalTxCost.Add(totalTxCost, commissionInBaseCoin)
	}
	if checkState.Accounts().GetBalance(sender, types.GetBaseCoinID()).Cmp(totalTxCost) == -1 {
		coin := checkState.Coins().GetCoin(types.GetBaseCoinID())
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), coin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), totalTxCost.String(), coin.GetFullSymbol(), coin.ID().String())),
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
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SubBalance(sender, types.GetBaseCoinID(), data.InitialReserve)

		coinId := checkState.App().GetNextCoinID()
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

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_symbol"), Value: []byte(data.Symbol.String()), Index: true},
			{Key: []byte("tx.coin_id"), Value: []byte(coinId.String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
