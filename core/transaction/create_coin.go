package transaction

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
	"regexp"
)

const maxCoinNameBytes = 64
const allowedCoinSymbols = "^[A-Z0-9]{3,10}$"

type CreateCoinData struct {
	Name                 string
	Symbol               types.CoinSymbol
	InitialAmount        *big.Int
	InitialReserve       *big.Int
	ConstantReserveRatio uint
}

func (data CreateCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name                 string           `json:"name"`
		Symbol               types.CoinSymbol `json:"coin_symbol"`
		InitialAmount        string           `json:"initial_amount"`
		InitialReserve       string           `json:"initial_reserve"`
		ConstantReserveRatio uint             `json:"constant_reserve_ratio"`
	}{
		Name:                 data.Name,
		Symbol:               data.Symbol,
		InitialAmount:        data.InitialAmount.String(),
		InitialReserve:       data.InitialReserve.String(),
		ConstantReserveRatio: data.ConstantReserveRatio,
	})
}

func (data CreateCoinData) String() string {
	return fmt.Sprintf("CREATE COIN symbol:%s reserve:%s amount:%s crr:%d",
		data.Symbol.String(), data.InitialReserve, data.InitialAmount, data.ConstantReserveRatio)
}

func (data CreateCoinData) Gas() int64 {
	return commissions.CreateTx
}

func (data CreateCoinData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	if len(data.Name) > maxCoinNameBytes {
		return Response{
			Code: code.InvalidCoinName,
			Log:  fmt.Sprintf("Coin name is invalid. Allowed up to %d bytes.", maxCoinNameBytes)}
	}

	if match, _ := regexp.MatchString(allowedCoinSymbols, data.Symbol.String()); !match {
		return Response{
			Code: code.InvalidCoinSymbol,
			Log:  fmt.Sprintf("Invalid coin symbol. Should be %s", allowedCoinSymbols)}
	}

	commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commission.Mul(commission, CommissionMultiplier)

	// compute additional price from letters count
	lettersCount := len(data.Symbol.String())
	var price int64 = 0
	switch lettersCount {
	case 3:
		price += 1000000 // 1mln bips
	case 4:
		price += 100000 // 100k bips
	case 5:
		price += 10000 // 10k bips
	case 6:
		price += 1000 // 1k bips
	case 7:
		price += 100 // 100 bips
	case 8:
		price += 10 // 10 bips
	}
	p := big.NewInt(10)
	p.Exp(p, big.NewInt(18), nil)
	p.Mul(p, big.NewInt(price))
	commission.Add(commission, p)

	totalTxCost := big.NewInt(0).Add(data.InitialReserve, commission)

	if context.GetBalance(sender, types.GetBaseCoin()).Cmp(totalTxCost) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
	}

	if context.CoinExists(data.Symbol) {
		return Response{
			Code: code.CoinAlreadyExists,
			Log:  fmt.Sprintf("Coin already exists")}
	}

	if data.ConstantReserveRatio < 10 || data.ConstantReserveRatio > 100 {
		return Response{
			Code: code.WrongCrr,
			Log:  fmt.Sprintf("Constant Reserve Ratio should be between 10 and 100")}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commission)

		context.SubBalance(sender, types.GetBaseCoin(), totalTxCost)
		context.CreateCoin(data.Symbol, data.Name, data.InitialAmount, data.ConstantReserveRatio, data.InitialReserve, sender)
		context.AddBalance(sender, data.Symbol, data.InitialAmount)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeCreateCoin}},
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
