package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
	"regexp"
)

const maxCoinNameBytes = 64
const allowedCoinSymbols = "^[A-Z0-9]{3,10}$"

var (
	minCoinSupply  = helpers.BipToPip(big.NewInt(1))
	minCoinReserve = helpers.BipToPip(big.NewInt(1))
)

type CreateCoinData struct {
	Name                 string           `json:"name"`
	Symbol               types.CoinSymbol `json:"symbol"`
	InitialAmount        *big.Int         `json:"initial_amount"`
	InitialReserve       *big.Int         `json:"initial_reserve"`
	ConstantReserveRatio uint             `json:"constant_reserve_ratio"`
}

func (data CreateCoinData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data CreateCoinData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
	if data.InitialReserve == nil || data.InitialAmount == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if !context.CoinExists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if len(data.Name) > maxCoinNameBytes {
		return &Response{
			Code: code.InvalidCoinName,
			Log:  fmt.Sprintf("Coin name is invalid. Allowed up to %d bytes.", maxCoinNameBytes)}
	}

	if match, _ := regexp.MatchString(allowedCoinSymbols, data.Symbol.String()); !match {
		return &Response{
			Code: code.InvalidCoinSymbol,
			Log:  fmt.Sprintf("Invalid coin symbol. Should be %s", allowedCoinSymbols)}
	}

	if context.CoinExists(data.Symbol) {
		return &Response{
			Code: code.CoinAlreadyExists,
			Log:  fmt.Sprintf("Coin already exists")}
	}

	if data.ConstantReserveRatio < 10 || data.ConstantReserveRatio > 100 {
		return &Response{
			Code: code.WrongCrr,
			Log:  fmt.Sprintf("Constant Reserve Ratio should be between 10 and 100")}
	}

	if data.InitialAmount.Cmp(minCoinSupply) == -1 || data.InitialAmount.Cmp(MaxCoinSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin supply should be between %s and %s", minCoinSupply.String(), MaxCoinSupply.String())}
	}

	if -1*data.InitialReserve.Cmp(minCoinReserve) != -1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin reserve should be greater than or equal to %s", minCoinReserve.String())}
	}

	return nil
}

func (data CreateCoinData) String() string {
	return fmt.Sprintf("CREATE COIN symbol:%s reserve:%s amount:%s crr:%d",
		data.Symbol.String(), data.InitialReserve, data.InitialAmount, data.ConstantReserveRatio)
}

func (data CreateCoinData) Gas() int64 {
	return commissions.CreateTx
}

// compute additional commission from letters count
func (data CreateCoinData) Commission() int64 {
	switch len(data.Symbol.String()) {
	case 3:
		return 1000000000 // 1mln bips
	case 4:
		return 100000000 // 100k bips
	case 5:
		return 10000000 // 10k bips
	case 6:
		return 1000000 // 1k bips
	case 7:
		return 100000 // 100 bips
	case 8:
		return 10000 // 10 bips
	}

	return 0
}

func (data CreateCoinData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()+data.Commission()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if tx.GasCoin != types.GetBaseCoin() {
		coin := context.GetStateCoin(tx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s", coin.ReserveBalance().String(), types.GetBaseCoin(), commissionInBaseCoin.String(), types.GetBaseCoin())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	if context.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), tx.GasCoin)}
	}

	if context.GetBalance(sender, types.GetBaseCoin()).Cmp(data.InitialReserve) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), data.InitialReserve.String(), types.GetBaseCoin())}
	}

	if tx.GasCoin.IsBaseCoin() {
		totalTxCost := big.NewInt(0)
		totalTxCost.Add(totalTxCost, data.InitialReserve)
		totalTxCost.Add(totalTxCost, commission)

		if context.GetBalance(sender, types.GetBaseCoin()).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), tx.GasCoin)}
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.SubCoinReserve(tx.GasCoin, commissionInBaseCoin)
		context.SubCoinVolume(tx.GasCoin, commission)

		context.SubBalance(sender, types.GetBaseCoin(), data.InitialReserve)
		context.SubBalance(sender, tx.GasCoin, commission)
		context.CreateCoin(data.Symbol, data.Name, data.InitialAmount, data.ConstantReserveRatio, data.InitialReserve)
		context.AddBalance(sender, data.Symbol, data.InitialAmount)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeCreateCoin)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.coin"), Value: []byte(data.Symbol.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
