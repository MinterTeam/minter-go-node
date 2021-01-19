package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
	"strconv"
)

type PriceCommissionData struct {
	Send                   *big.Int
	SellCoin               *big.Int
	SellAllCoin            *big.Int
	BuyCoin                *big.Int
	CreateCoin             *big.Int
	DeclareCandidacy       *big.Int
	Delegate               *big.Int
	Unbond                 *big.Int
	RedeemCheck            *big.Int
	SetCandidateOnline     *big.Int
	SetCandidateOffline    *big.Int
	CreateMultisig         *big.Int
	Multisend              *big.Int
	EditCandidate          *big.Int
	SetHaltBlock           *big.Int
	RecreateCoin           *big.Int
	EditCoinOwner          *big.Int
	EditMultisig           *big.Int
	PriceVote              *big.Int
	EditCandidatePublicKey *big.Int
	AddLiquidity           *big.Int
	RemoveLiquidity        *big.Int
	SellSwapPool           *big.Int
	BuySwapPool            *big.Int
	SellAllSwapPool        *big.Int
	EditCommission         *big.Int
	MoveStake              *big.Int
	MintToken              *big.Int
	BurnToken              *big.Int
	CreateToken            *big.Int
	RecreateToken          *big.Int
	PriceCommission        *big.Int
	UpdateNetwork          *big.Int
	Coin                   types.CoinID
	PubKey                 types.Pubkey
	Height                 uint64
}

func (data PriceCommissionData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data PriceCommissionData) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
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

func (data PriceCommissionData) Gas() int64 {
	return commissions.PriceVoteData // todo
}

func (data PriceCommissionData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	commissionInBaseCoin := tx.CommissionInBaseCoin()
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
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypePriceCommission)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}

func (data PriceCommissionData) price() *commission.Price {
	return &commission.Price{
		Send:                   data.Send,
		SellCoin:               data.SellCoin,
		SellAllCoin:            data.SellAllCoin,
		BuyCoin:                data.BuyCoin,
		CreateCoin:             data.CreateCoin,
		DeclareCandidacy:       data.DeclareCandidacy,
		Delegate:               data.Delegate,
		Unbond:                 data.Unbond,
		RedeemCheck:            data.RedeemCheck,
		SetCandidateOnline:     data.SetCandidateOnline,
		SetCandidateOffline:    data.SetCandidateOffline,
		CreateMultisig:         data.CreateMultisig,
		Multisend:              data.Multisend,
		EditCandidate:          data.EditCandidate,
		SetHaltBlock:           data.SetHaltBlock,
		RecreateCoin:           data.RecreateCoin,
		EditCoinOwner:          data.EditCoinOwner,
		EditMultisig:           data.EditMultisig,
		PriceVote:              data.PriceVote,
		EditCandidatePublicKey: data.EditCandidatePublicKey,
		AddLiquidity:           data.AddLiquidity,
		RemoveLiquidity:        data.RemoveLiquidity,
		SellSwapPool:           data.SellSwapPool,
		BuySwapPool:            data.BuySwapPool,
		SellAllSwapPool:        data.SellAllSwapPool,
		EditCommission:         data.EditCommission,
		MoveStake:              data.MoveStake,
		MintToken:              data.MintToken,
		BurnToken:              data.BurnToken,
		CreateToken:            data.CreateToken,
		RecreateToken:          data.RecreateToken,
		PriceCommission:        data.PriceCommission,
		UpdateNetwork:          data.UpdateNetwork,
		Coin:                   data.Coin,
	}
}
