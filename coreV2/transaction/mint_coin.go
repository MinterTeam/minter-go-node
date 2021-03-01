package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type MintTokenData struct {
	Coin  types.CoinID
	Value *big.Int
}

func (data MintTokenData) Gas() int {
	return gasMintToken
}
func (data MintTokenData) TxType() TxType {
	return TypeMintToken
}

func (data MintTokenData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	coin := context.Coins().GetCoin(data.Coin)
	if coin == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if !coin.IsMintable() {
		return &Response{
			Code: code.CoinNotMintable,
			Log:  "Coin not mintable",
			Info: EncodeError(code.NewCoinIsNotMintable(coin.GetFullSymbol(), data.Coin.String())),
		}
	}

	if big.NewInt(0).Add(coin.Volume(), data.Value).Cmp(coin.MaxSupply()) == 1 {
		return &Response{
			Code: code.WrongCoinEmission,
			Log:  fmt.Sprintf("Coin supply should not exceed the limit: %s", coin.MaxSupply()),
			Info: EncodeError(code.NewWrongCoinEmission("", coin.MaxSupply().String(), coin.Volume().String(), data.Value.String(), "")),
		}
	}

	sender, _ := tx.Sender()
	symbolInfo := context.Coins().GetSymbolInfo(coin.Symbol())
	if coin.Version() != 0 || symbolInfo == nil || symbolInfo.OwnerAddress().Compare(sender) != 0 {
		var owner *string
		if symbolInfo != nil && symbolInfo.OwnerAddress() != nil {
			own := symbolInfo.OwnerAddress().String()
			owner = &own
		}
		return &Response{
			Code: code.IsNotOwnerOfCoin,
			Log:  "Sender is not owner of coin",
			Info: EncodeError(code.NewIsNotOwnerOfCoin(coin.Symbol().String(), owner)),
		}
	}

	return nil
}

func (data MintTokenData) String() string {
	return fmt.Sprintf("MINT COIN: %d", data.Coin)
}

func (data MintTokenData) CommissionData(price *commission.Price) *big.Int {
	return price.MintToken
}

func (data MintTokenData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.AddVolume(data.Coin, data.Value)
		deliverState.Accounts.AddBalance(sender, data.Coin, data.Value)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_id"), Value: []byte(data.Coin.String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
