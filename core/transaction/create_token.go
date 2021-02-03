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
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type CreateTokenData struct {
	Name          string
	Symbol        types.CoinSymbol
	InitialAmount *big.Int
	MaxSupply     *big.Int
	Mintable      bool
	Burnable      bool
}

func (data CreateTokenData) TxType() TxType {
	return TypeCreateToken
}

func (data CreateTokenData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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

	if data.InitialAmount.Cmp(minTokenSupply) == -1 || data.InitialAmount.Cmp(data.MaxSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Coin amount should be between %s and %s", minTokenSupply.String(), data.MaxSupply.String()),
			Info: EncodeError(code.NewWrongCoinSupply(minTokenSupply.String(), minTokenSupply.String(), data.MaxSupply.String(), "", "", data.InitialAmount.String())),
		}
	}

	if data.MaxSupply.Cmp(maxCoinSupply) == 1 {
		return &Response{
			Code: code.WrongCoinSupply,
			Log:  fmt.Sprintf("Max coin supply should be less %s", maxCoinSupply.String()),
			Info: EncodeError(code.NewWrongCoinSupply(minTokenSupply.String(), maxCoinSupply.String(), data.MaxSupply.String(), "", "", data.InitialAmount.String())),
		}
	}

	return nil
}

func (data CreateTokenData) String() string {
	return fmt.Sprintf("CREATE TOKEN symbol:%s emission:%s",
		data.Symbol.String(), data.MaxSupply)
}

func (data CreateTokenData) CommissionData(price *commission.Price) *big.Int {
	createTicker := new(big.Int).Set(price.CreateTicker7to10)
	switch len(data.Symbol.String()) {
	case 3:
		createTicker = price.CreateTicker3
	case 4:
		createTicker = price.CreateTicker4
	case 5:
		createTicker = price.CreateTicker5
	case 6:
		createTicker = price.CreateTicker6
	}

	return big.NewInt(0).Add(createTicker, price.CreateToken)
}

func (data CreateTokenData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		coinId := checkState.App().GetNextCoinID()
		deliverState.Coins.CreateToken(
			coinId,
			data.Symbol,
			data.Name,
			data.Mintable,
			data.Burnable,
			data.InitialAmount,
			data.MaxSupply,
			&sender,
		)

		deliverState.App.SetCoinsCount(coinId.Uint32())
		deliverState.Accounts.AddBalance(sender, coinId, data.InitialAmount)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			{Key: []byte("tx.coin_symbol"), Value: []byte(data.Symbol.String())},
			{Key: []byte("tx.coin_id"), Value: []byte(coinId.String())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
