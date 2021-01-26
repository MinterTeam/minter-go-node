package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
	"strconv"
)

type PriceCommissionData struct {
	PubKey                  types.Pubkey
	Height                  uint64
	Coin                    types.CoinID
	PayloadByte             *big.Int
	Send                    *big.Int
	BuyBancor               *big.Int
	SellBancor              *big.Int
	SellAllBancor           *big.Int
	BuyPool                 *big.Int
	SellPool                *big.Int
	SellAllPool             *big.Int
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
	PriceVote               *big.Int
	EditCandidatePublicKey  *big.Int
	AddLiquidity            *big.Int
	RemoveLiquidity         *big.Int
	EditCandidateCommission *big.Int
	MoveStake               *big.Int
	MintToken               *big.Int
	BurnToken               *big.Int
	PriceCommission         *big.Int
	UpdateNetwork           *big.Int
	More                    []*big.Int `rlp:"tail"`
}

func (data PriceCommissionData) TxType() TxType {
	return TypePriceCommission
}

func (data PriceCommissionData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data PriceCommissionData) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
	if len(data.More) > 0 { // todo
		return &Response{
			Code: code.DecodeError,
			Log:  "More parameters than expected",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if data.Height < block {
		return &Response{
			Code: code.VoiceExpired,
			Log:  "voice is produced for the past state",
			Info: EncodeError(code.NewVoiceExpired(strconv.Itoa(int(block)), strconv.Itoa(int(data.Height)))),
		}
	}

	if context.Commission().IsVoteExists(data.Height, data.PubKey) {
		return &Response{
			Code: code.VoiceAlreadyExists,
			Log:  "Commission price vote with such public key and height already exists",
			Info: EncodeError(code.NewVoiceAlreadyExists(strconv.FormatUint(data.Height, 10), data.GetPubKey().String())),
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

func (data PriceCommissionData) String() string {
	return fmt.Sprintf("PRICE COMMISSION in coin: %d", data.Coin)
}

func (data PriceCommissionData) CommissionData(price *commission.Price) *big.Int {
	return price.PriceCommission
}

func (data PriceCommissionData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int, gas int64) Response {
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

	commissionInBaseCoin := tx.Commission(price)
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

	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Commission.AddVoice(data.Height, data.PubKey, data.price().Encode())

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.gas"), Value: []byte(strconv.Itoa(int(gas)))},
		kv.Pair{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypePriceCommission)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   gas,
		GasWanted: gas,
		Tags:      tags,
	}
}

func (data PriceCommissionData) price() *commission.Price {
	return &commission.Price{
		Coin:                    data.Coin,
		PayloadByte:             data.PayloadByte,
		Send:                    data.Send,
		BuyBancor:               data.BuyBancor,
		SellBancor:              data.SellBancor,
		SellAllBancor:           data.SellAllBancor,
		BuyPool:                 data.BuyPool,
		SellPool:                data.SellPool,
		SellAllPool:             data.SellAllPool,
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
		PriceVote:               data.PriceVote,
		EditCandidatePublicKey:  data.EditCandidatePublicKey,
		AddLiquidity:            data.AddLiquidity,
		RemoveLiquidity:         data.RemoveLiquidity,
		EditCandidateCommission: data.EditCandidateCommission,
		MoveStake:               data.MoveStake,
		BurnToken:               data.BurnToken,
		MintToken:               data.MintToken,
		PriceCommission:         data.PriceCommission,
		UpdateNetwork:           data.UpdateNetwork,
		More:                    data.More,
	}
}
