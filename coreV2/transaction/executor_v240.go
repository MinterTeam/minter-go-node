package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"sync"

	"github.com/MinterTeam/minter-go-node/coreV2/check"
	abcTypes "github.com/tendermint/tendermint/abci/types"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
)

type ExecutorTx interface {
	RunTx(context state.Interface, rawTx []byte, rewardPool *big.Int, currentBlock uint64, currentMempool *sync.Map, minGasPrice uint32, notSaveTags bool) Response
	DecoderTx
}

type DecoderTx interface {
	DecodeFromBytesWithoutSig(buf []byte) (*Transaction, error)
	DecodeFromBytes(buf []byte) (*Transaction, error)
}

type ExecutorV240 struct {
	*Executor
	decodeTxFunc func(txType TxType) (Data, bool)
}

func NewExecutorV250(decodeTxFunc func(txType TxType) (Data, bool)) ExecutorTx {
	return &ExecutorV240{decodeTxFunc: decodeTxFunc, Executor: &Executor{decodeTxFunc: decodeTxFunc}}
}

func (e *ExecutorV240) RunTx(context state.Interface, rawTx []byte, rewardPool *big.Int, currentBlock uint64, currentMempool *sync.Map, minGasPrice uint32, notSaveTags bool) Response {
	lenRawTx := len(rawTx)
	if lenRawTx > maxTxLength {
		return Response{
			Code: code.TxTooLarge,
			Log:  fmt.Sprintf("TX length is over %d bytes", maxTxLength),
			Info: EncodeError(code.NewTxTooLarge(fmt.Sprintf("%d", maxTxLength), fmt.Sprintf("%d", lenRawTx))),
		}
	}

	tx, err := e.DecodeFromBytes(rawTx)
	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if tx.ChainID != types.CurrentChainID {
		return Response{
			Code: code.WrongChainID,
			Log:  "Wrong chain id",
			Info: EncodeError(code.NewWrongChainID(fmt.Sprintf("%d", types.CurrentChainID), fmt.Sprintf("%d", tx.ChainID))),
		}
	}

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	if !checkState.Coins().Exists(tx.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin),
			Info: EncodeError(code.NewCoinNotExists("", tx.GasCoin.String())),
		}
	}

	if isCheck && tx.GasPrice < minGasPrice {
		return Response{
			Code: code.TooLowGasPrice,
			Log:  fmt.Sprintf("Gas price of tx is too low to be included in mempool. Expected %d", minGasPrice),
			Info: EncodeError(code.NewTooLowGasPrice(fmt.Sprintf("%d", minGasPrice), fmt.Sprintf("%d", tx.GasPrice))),
		}
	}

	lenPayload := len(tx.Payload)
	if lenPayload > maxPayloadLength {
		return Response{
			Code: code.TxPayloadTooLarge,
			Log:  fmt.Sprintf("TX payload length is over %d bytes", maxPayloadLength),
			Info: EncodeError(code.NewTxPayloadTooLarge(fmt.Sprintf("%d", maxPayloadLength), fmt.Sprintf("%d", lenPayload))),
		}
	}

	lenServiceData := len(tx.ServiceData)
	if lenServiceData > maxServiceDataLength {
		return Response{
			Code: code.TxServiceDataTooLarge,
			Log:  fmt.Sprintf("TX service data length is over %d bytes", maxServiceDataLength),
			Info: EncodeError(code.NewTxServiceDataTooLarge(fmt.Sprintf("%d", maxServiceDataLength), fmt.Sprintf("%d", lenServiceData))),
		}
	}

	sender, err := tx.Sender()
	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	// check multi-signature
	if tx.SignatureType == SigTypeMulti {
		multisig := checkState.Accounts().GetAccount(tx.multisig.Multisig)

		if !multisig.IsMultisig() {
			return Response{
				Code: code.MultisigNotExists,
				Log:  "Multisig does not exists",
				Info: EncodeError(code.NewMultisigNotExists(tx.multisig.Multisig.String())),
			}
		}

		multisigData := multisig.Multisig()

		if len(tx.multisig.Signatures) > 32 || len(multisigData.Weights) < len(tx.multisig.Signatures) {
			return Response{
				Code: code.IncorrectMultiSignature,
				Log:  "Incorrect multi-signature",
				Info: EncodeError(code.NewIncorrectMultiSignature()),
			}
		}

		txHash := tx.Hash()
		var totalWeight uint32
		var usedAccounts = map[types.Address]bool{}

		for _, sig := range tx.multisig.Signatures {
			signer, err := RecoverPlain(txHash, sig.R, sig.S, sig.V)
			if err != nil {
				return Response{
					Code: code.IncorrectMultiSignature,
					Log:  "Incorrect multi-signature",
					Info: EncodeError(code.NewIncorrectMultiSignature()),
				}
			}

			if usedAccounts[signer] {
				return Response{
					Code: code.DuplicatedAddresses,
					Log:  "Duplicated multisig addresses",
					Info: EncodeError(code.NewDuplicatedAddresses(signer.String())),
				}
			}

			usedAccounts[signer] = true
			totalWeight += multisigData.GetWeight(signer)
		}

		if totalWeight < multisigData.Threshold {
			return Response{
				Code: code.NotEnoughMultisigVotes,
				Log:  fmt.Sprintf("Not enough multisig votes. Needed %d, has %d", multisigData.Threshold, totalWeight),
				Info: EncodeError(code.NewNotEnoughMultisigVotes(fmt.Sprintf("%d", multisigData.Threshold), fmt.Sprintf("%d", totalWeight))),
			}
		}

	}

	if expectedNonce := checkState.Accounts().GetNonce(sender) + 1; expectedNonce != tx.Nonce {
		return Response{
			Code: code.WrongNonce,
			Log:  fmt.Sprintf("Unexpected nonce. Expected: %d, got %d.", expectedNonce, tx.Nonce),
			Info: EncodeError(code.NewWrongNonce(fmt.Sprintf("%d", expectedNonce), fmt.Sprintf("%d", tx.Nonce))),
		}
	}

	commissions := checkState.Commission().GetCommissions()
	price := tx.MulGasPrice(tx.Price(commissions))
	coinCommission := abcTypes.EventAttribute{Key: []byte("tx.commission_price_coin"), Value: []byte(strconv.Itoa(int(commissions.Coin)))}
	priceCommission := abcTypes.EventAttribute{Key: []byte("tx.commission_price"), Value: []byte(price.String())}

	if !commissions.Coin.IsBaseCoin() {
		price = checkState.Swap().GetSwapper(commissions.Coin, types.GetBaseCoinID()).CalculateBuyForSell(price)
	}
	if price == nil {
		return Response{
			Code: code.CommissionCoinNotSufficient,
			Log:  fmt.Sprint("Not possible to pay commission"),
			Info: EncodeError(code.NewCommissionCoinNotSufficient("", "")),
		}
	}

	response := tx.decodedData.Run(tx, context, rewardPool, currentBlock, price)
	if response.Code == code.OK && isCheck {
		// check if mempool already has transactions from this address
		if _, has := currentMempool.LoadOrStore(sender, true); has {
			return Response{
				Code: code.TxFromSenderAlreadyInMempool,
				Log:  fmt.Sprintf("Tx from %s already exists in mempool", sender.String()),
				Info: EncodeError(code.NewTxFromSenderAlreadyInMempool(sender.String(), strconv.Itoa(int(currentBlock)))),
			}
		}
	}

	if !isCheck && response.Code != 0 {
		commissionInBaseCoin := big.NewInt(0).Add(commissions.FailedTxPrice(), big.NewInt(0).Mul(big.NewInt(tx.payloadAndServiceDataLen()), commissions.PayloadByte))
		if types.CurrentChainID != types.ChainTestnet || currentBlock > 4451966 { // todo: remove check (need for testnet)
			commissionInBaseCoin = tx.MulGasPrice(commissionInBaseCoin)
		}

		if !commissions.Coin.IsBaseCoin() {
			commissionInBaseCoin = checkState.Swap().GetSwapper(commissions.Coin, types.GetBaseCoinID()).CalculateBuyForSell(commissionInBaseCoin)
		}

		commissionPoolSwapper := checkState.Swap().GetSwapper(tx.commissionCoin(), types.GetBaseCoinID())
		gasCoin := checkState.Coins().GetCoin(tx.commissionCoin())
		commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		var intruder = sender
		if tx.Type == TypeRedeemCheck {
			decodedCheck, err := check.DecodeFromBytes(tx.decodedData.(*RedeemCheckData).RawCheck)
			if err != nil {
				return Response{
					Code: code.DecodeError,
					Log:  err.Error(),
					Info: EncodeError(code.NewDecodeError()),
				}
			}
			checkSender, err := decodedCheck.Sender()
			if err != nil {
				return Response{
					Code: code.DecodeError,
					Log:  err.Error(),
					Info: EncodeError(code.NewDecodeError()),
				}
			}

			intruder = checkSender
			response.Tags = append(response.Tags,
				abcTypes.EventAttribute{Key: []byte("tx.check_owner"), Value: []byte(hex.EncodeToString(intruder[:]))},
			)
		}
		balance := checkState.Accounts().GetBalance(intruder, tx.commissionCoin())
		if balance.Sign() == 1 {
			if balance.Cmp(commission) == -1 {
				commission = big.NewInt(0).Set(balance)
				if isGasCommissionFromPoolSwap {
					commissionInBaseCoin = commissionPoolSwapper.CalculateBuyForSell(commission)
					if commissionInBaseCoin == nil || commissionInBaseCoin.Sign() == 0 {
						return Response{
							Code: code.CommissionCoinNotSufficient,
							Log:  fmt.Sprint("Not possible to pay commission"),
							Info: EncodeError(code.NewCommissionCoinNotSufficient("", "")),
						}
					}
				} else if !gasCoin.ID().IsBaseCoin() && gasCoin.BaseOrHasReserve() {
					commissionInBaseCoin, errResp = CalculateSaleReturnAndCheck(gasCoin, commission)
					if errResp != nil {
						return *errResp
					}
				} else {
					commissionInBaseCoin = commission
				}
			}

			if deliverState, ok := context.(*state.State); ok {
				if isGasCommissionFromPoolSwap {
					commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.commissionCoin(), types.GetBaseCoinID(), commission, commissionInBaseCoin)
				} else if !tx.commissionCoin().IsBaseCoin() {
					deliverState.Coins.SubVolume(tx.commissionCoin(), commission)
					deliverState.Coins.SubReserve(tx.commissionCoin(), commissionInBaseCoin)
				}

				deliverState.Accounts.SubBalance(intruder, tx.commissionCoin(), commission)

				rewardPool.Add(rewardPool, commissionInBaseCoin)
				response.Tags = append(response.Tags,
					abcTypes.EventAttribute{Key: []byte("tx.fail_fee"), Value: []byte(commission.String())},
				)
			}
		}
	}

	if notSaveTags || isCheck {
		response.Tags = nil
	} else {
		response.Tags = append(response.Tags,
			coinCommission,
			priceCommission,
			abcTypes.EventAttribute{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:])), Index: true},
			abcTypes.EventAttribute{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(tx.decodedData.TxType())})), Index: true},
			abcTypes.EventAttribute{Key: []byte("tx.commission_coin"), Value: []byte(tx.commissionCoin().String()), Index: true},
		)
	}

	response.GasUsed = tx.Gas()
	response.GasWanted = response.GasUsed
	response.GasPrice = tx.GasPrice

	return response
}
