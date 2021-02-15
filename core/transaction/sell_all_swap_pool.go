package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type SellAllSwapPoolData struct {
	CoinToSell        types.CoinID
	CoinToBuy         types.CoinID
	MinimumValueToBuy *big.Int
}

func (data SellAllSwapPoolData) Gas() int {
	return gasSellAllSwapPool
}

func (data SellAllSwapPoolData) TxType() TxType {
	return TypeSellAllSwapPool
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
			Info: EncodeError(code.NewPairNotExists(data.CoinToSell.String(), data.CoinToBuy.String())),
		}
	}
	return nil
}

func (data SellAllSwapPoolData) String() string {
	return fmt.Sprintf("SWAP POOL SELL ALL")
}

func (data SellAllSwapPoolData) CommissionData(price *commission.Price) *big.Int {
	return price.SellAllPoolBase // todo
}

func (data SellAllSwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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
	commissionPoolSwapper := checkState.Swap().GetSwapper(data.CoinToSell, types.GetBaseCoinID())
	sellCoin := checkState.Coins().GetCoin(data.CoinToSell)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, sellCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	balance := checkState.Accounts().GetBalance(sender, data.CoinToSell)
	available := big.NewInt(0).Set(balance)
	balance.Sub(available, commission)

	if balance.Sign() != 1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), balance.String(), sellCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), balance.String(), sellCoin.GetFullSymbol(), data.CoinToSell.String())),
		}
	}

	swapper := checkState.Swap().GetSwapper(data.CoinToSell, data.CoinToBuy)
	if isGasCommissionFromPoolSwap == true && data.CoinToBuy.IsBaseCoin() {
		swapper = commissionPoolSwapper.AddLastSwapStep(commission, commissionInBaseCoin)
	}

	errResp = CheckSwap(swapper, sellCoin, checkState.Coins().GetCoin(data.CoinToBuy), balance, data.MinimumValueToBuy, false)
	if errResp != nil {
		return *errResp
	}

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(data.CoinToSell, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !data.CoinToSell.IsBaseCoin() {
			deliverState.Coins.SubVolume(data.CoinToSell, commission)
			deliverState.Coins.SubReserve(data.CoinToSell, commissionInBaseCoin)
		}

		amountIn, amountOut, _ := deliverState.Swap.PairSell(data.CoinToSell, data.CoinToBuy, balance, data.MinimumValueToBuy)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, amountIn)
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, amountOut)

		deliverState.Accounts.SubBalance(sender, data.CoinToSell, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
			{Key: []byte("tx.return"), Value: []byte(amountOut.String())},
			{Key: []byte("tx.sell_amount"), Value: []byte(available.String())},
			{Key: []byte("tx.pair_ids"), Value: []byte(liquidityCoinName(data.CoinToBuy, data.CoinToSell))},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

type DummyCoin struct {
	id         types.CoinID
	volume     *big.Int
	reserve    *big.Int
	crr        uint32
	fullSymbol string
	maxSupply  *big.Int
}

func NewDummyCoin(id types.CoinID, volume *big.Int, reserve *big.Int, crr uint32, fullSymbol string, maxSupply *big.Int) *DummyCoin {
	return &DummyCoin{id: id, volume: volume, reserve: reserve, crr: crr, fullSymbol: fullSymbol, maxSupply: maxSupply}
}

func (m DummyCoin) ID() types.CoinID {
	return m.id
}

func (m DummyCoin) BaseOrHasReserve() bool {
	return m.ID().IsBaseCoin() || (m.Crr() > 0 && m.Reserve().Sign() == 1)
}

func (m DummyCoin) Volume() *big.Int {
	return m.volume
}

func (m DummyCoin) Reserve() *big.Int {
	return m.reserve
}

func (m DummyCoin) Crr() uint32 {
	return m.crr
}

func (m DummyCoin) GetFullSymbol() string {
	return m.fullSymbol
}
func (m DummyCoin) MaxSupply() *big.Int {
	return m.maxSupply
}

type CalculateCoin interface {
	ID() types.CoinID
	BaseOrHasReserve() bool
	Volume() *big.Int
	Reserve() *big.Int
	Crr() uint32
	GetFullSymbol() string
	MaxSupply() *big.Int
}
type gasMethod bool

func (isGasCommissionFromPoolSwap gasMethod) String() string {
	if isGasCommissionFromPoolSwap {
		return "pool"
	}
	return "bancor"
}

func CalculateCommission(checkState *state.CheckState, swapper swap.EditableChecker, gasCoin CalculateCoin, commissionInBaseCoin *big.Int) (commission *big.Int, poolSwap gasMethod, errResp *Response) {
	if gasCoin.ID().IsBaseCoin() {
		return new(big.Int).Set(commissionInBaseCoin), false, nil
	}
	commissionFromPool, responseFromPool := commissionFromPool(swapper, gasCoin, checkState.Coins().GetCoin(types.BasecoinID), commissionInBaseCoin)
	commissionFromReserve, responseFromReserve := commissionFromReserve(gasCoin, commissionInBaseCoin)

	if responseFromPool != nil && responseFromReserve != nil {
		return nil, false, &Response{
			Code: code.CommissionCoinNotSufficient,
			Log:  fmt.Sprintf("Not possible to pay commission in coin %s", gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewCommissionCoinNotSufficient(responseFromReserve.Log, responseFromPool.Log)),
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

func commissionFromPool(swapChecker swap.EditableChecker, coin CalculateCoin, baseCoin CalculateCoin, commissionInBaseCoin *big.Int) (*big.Int, *Response) {
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

func commissionFromReserve(gasCoin CalculateCoin, commissionInBaseCoin *big.Int) (*big.Int, *Response) {
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
