package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type VoteCommissionDataV1 struct {
	PubKey                  types.Pubkey
	Height                  uint64
	Coin                    types.CoinID
	PayloadByte             *big.Int
	Send                    *big.Int
	BuyBancor               *big.Int
	SellBancor              *big.Int
	SellAllBancor           *big.Int
	BuyPoolBase             *big.Int
	BuyPoolDelta            *big.Int
	SellPoolBase            *big.Int
	SellPoolDelta           *big.Int
	SellAllPoolBase         *big.Int
	SellAllPoolDelta        *big.Int
	CreateTicker3           *big.Int
	CreateTicker4           *big.Int
	CreateTicker5           *big.Int
	CreateTicker6           *big.Int
	CreateTicker7to10       *big.Int
	CreateCoin              *big.Int
	CreateToken             *big.Int
	RecreateCoin            *big.Int
	RecreateToken           *big.Int
	DeclareCandidacy        *big.Int
	Delegate                *big.Int
	Unbond                  *big.Int
	RedeemCheck             *big.Int
	SetCandidateOn          *big.Int
	SetCandidateOff         *big.Int
	CreateMultisig          *big.Int
	MultisendBase           *big.Int
	MultisendDelta          *big.Int
	EditCandidate           *big.Int
	SetHaltBlock            *big.Int
	EditTickerOwner         *big.Int
	EditMultisig            *big.Int
	EditCandidatePublicKey  *big.Int
	CreateSwapPool          *big.Int
	AddLiquidity            *big.Int
	RemoveLiquidity         *big.Int
	EditCandidateCommission *big.Int
	MintToken               *big.Int
	BurnToken               *big.Int
	VoteCommission          *big.Int
	VoteUpdate              *big.Int
	More                    []*big.Int `rlp:"tail"`
}

func (data VoteCommissionDataV1) TxType() TxType {
	return TypeVoteCommission
}
func (data VoteCommissionDataV1) Gas() int64 {
	return gasVoteCommission
}

func (data VoteCommissionDataV1) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data VoteCommissionDataV1) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
	if len(data.More) > 0 { // todo
		return &Response{
			Code: code.DecodeError,
			Log:  "More parameters than expected",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if data.Height < block {
		return &Response{
			Code: code.VoteExpired,
			Log:  "vote is produced for the past state",
			Info: EncodeError(code.NewVoteExpired(strconv.Itoa(int(block)), strconv.Itoa(int(data.Height)))),
		}
	}

	if context.Commission().IsVoteExists(data.Height, data.PubKey) {
		return &Response{
			Code: code.VoteAlreadyExists,
			Log:  "Commission price vote with such public key and height already exists",
			Info: EncodeError(code.NewVoteAlreadyExists(strconv.FormatUint(data.Height, 10), data.GetPubKey().String())),
		}
	}

	coin := context.Coins().GetCoin(data.Coin)
	if coin == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin to sell not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if !data.Coin.IsBaseCoin() && !context.Swap().SwapPoolExist(data.Coin, types.GetBaseCoinID()) {
		return &Response{
			Code: code.PairNotExists,
			Log:  "swap pool not found",
			Info: EncodeError(code.NewPairNotExists(data.Coin.String(), types.GetBaseCoinID().String())),
		}
	}
	return checkCandidateOwnership(data, tx, context)
}

func (data VoteCommissionDataV1) String() string {
	return fmt.Sprintf("PRICE COMMISSION in coin: %d", data.Coin)
}

func (data VoteCommissionDataV1) CommissionData(price *commission.Price) *big.Int {
	return price.VoteCommission
}

func (data VoteCommissionDataV1) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState, currentBlock)
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

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  []*swap.OrderDetail
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.CommissionCoin(), types.GetBaseCoinID(), commission, big.NewInt(0))
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.CommissionCoin(),
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  ownersCom,
			}
			for _, value := range ownersCom {
				deliverState.Accounts.AddBalance(value.Owner, tx.CommissionCoin(), value.ValueBigInt)
			}
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.CommissionCoin(), commission)
			deliverState.Coins.SubReserve(tx.CommissionCoin(), commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Commission.AddVote(data.Height, data.PubKey, data.price().Encode())

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.public_key"), Value: []byte(hex.EncodeToString(data.PubKey[:])), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

func (data VoteCommissionDataV1) price() *commission.Price {
	return &commission.Price{
		Coin:                    data.Coin,
		PayloadByte:             data.PayloadByte,
		Send:                    data.Send,
		BuyBancor:               data.BuyBancor,
		SellBancor:              data.SellBancor,
		SellAllBancor:           data.SellAllBancor,
		BuyPoolBase:             data.BuyPoolBase,
		BuyPoolDelta:            data.BuyPoolDelta,
		SellPoolBase:            data.SellPoolBase,
		SellPoolDelta:           data.SellPoolDelta,
		SellAllPoolBase:         data.SellAllPoolBase,
		SellAllPoolDelta:        data.SellAllPoolDelta,
		CreateTicker3:           data.CreateTicker3,
		CreateTicker4:           data.CreateTicker4,
		CreateTicker5:           data.CreateTicker5,
		CreateTicker6:           data.CreateTicker6,
		CreateTicker7to10:       data.CreateTicker7to10,
		CreateCoin:              data.CreateCoin,
		CreateToken:             data.CreateToken,
		RecreateCoin:            data.RecreateCoin,
		RecreateToken:           data.RecreateToken,
		DeclareCandidacy:        data.DeclareCandidacy,
		Delegate:                data.Delegate,
		Unbond:                  data.Unbond,
		RedeemCheck:             data.RedeemCheck,
		SetCandidateOn:          data.SetCandidateOn,
		SetCandidateOff:         data.SetCandidateOff,
		CreateMultisig:          data.CreateMultisig,
		MultisendBase:           data.MultisendBase,
		MultisendDelta:          data.MultisendDelta,
		EditCandidate:           data.EditCandidate,
		SetHaltBlock:            data.SetHaltBlock,
		EditTickerOwner:         data.EditTickerOwner,
		EditMultisig:            data.EditMultisig,
		EditCandidatePublicKey:  data.EditCandidatePublicKey,
		CreateSwapPool:          data.CreateSwapPool,
		AddLiquidity:            data.AddLiquidity,
		RemoveLiquidity:         data.RemoveLiquidity,
		EditCandidateCommission: data.EditCandidateCommission,
		BurnToken:               data.BurnToken,
		MintToken:               data.MintToken,
		VoteCommission:          data.VoteCommission,
		VoteUpdate:              data.VoteUpdate,
		More:                    data.More,
	}
}
