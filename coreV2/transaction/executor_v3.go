package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"sync"

	"github.com/MinterTeam/minter-go-node/coreV2/check"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	abcTypes "github.com/tendermint/tendermint/abci/types"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
)

type ExecutorV3 struct {
	*Executor
	decodeTxFunc func(txType TxType) (Data, bool)
}

func NewExecutorV3(decodeTxFunc func(txType TxType) (Data, bool)) ExecutorTx {
	return &ExecutorV3{decodeTxFunc: decodeTxFunc, Executor: &Executor{decodeTxFunc: decodeTxFunc}}
}

func (e *ExecutorV3) RunTx(context state.Interface, rawTx []byte, rewardPool *big.Int, currentBlock uint64, currentMempool *sync.Map, minGasPrice uint32, notSaveTags bool) Response {
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

	if tx.Type == TypeLockStake && currentBlock <= 10197360 {
		return Response{
			Code: code.Unavailable,
			Log:  "LockStake available from block 10197360 ",
			Info: EncodeError(code.NewCustomCode(code.Unavailable)),
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

	if !checkState.Coins().Exists(tx.CommissionCoin()) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.CommissionCoin()),
			Info: EncodeError(code.NewCoinNotExists("", tx.CommissionCoin().String())),
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
				Info: EncodeError(code.NewIncorrectMultiSignature("error in the number of signers")),
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
					Info: EncodeError(code.NewIncorrectMultiSignature(err.Error())),
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

	if price.Sign() != 0 {
		if !commissions.Coin.IsBaseCoin() {
			var resp *Response
			resp, price, _ = CheckSwap(checkState.Swap().GetSwapper(commissions.Coin, types.GetBaseCoinID()), checkState.Coins().GetCoin(commissions.Coin), checkState.Coins().GetCoin(0), price, big.NewInt(0), false)
			if resp != nil {
				return *resp
			}
		}
		if price == nil || price.Sign() != 1 {
			return Response{
				Code: code.CommissionCoinNotSufficient,
				Log:  fmt.Sprint("Not possible to pay commission"),
				Info: EncodeError(code.NewCommissionCoinNotSufficient("", "")),
			}
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

	if !isCheck {
		if response.Code != 0 {
			commissionInBaseCoin := big.NewInt(0).Add(commissions.FailedTx, big.NewInt(0).Mul(big.NewInt(tx.PayloadAndServiceDataLen()), commissions.PayloadByte))
			commissionInBaseCoin = tx.MulGasPrice(commissionInBaseCoin)

			if !commissions.Coin.IsBaseCoin() {
				var resp *Response
				resp, commissionInBaseCoin, _ = CheckSwap(checkState.Swap().GetSwapper(commissions.Coin, types.GetBaseCoinID()), checkState.Coins().GetCoin(commissions.Coin), checkState.Coins().GetCoin(0), commissionInBaseCoin, big.NewInt(0), false)
				if resp != nil {
					return *resp
				}
				if commissionInBaseCoin == nil || commissionInBaseCoin.Sign() != 1 {
					return Response{
						Code: code.CommissionCoinNotSufficient,
						Log:  fmt.Sprint("Not possible to pay commission"),
						Info: EncodeError(code.NewCommissionCoinNotSufficient("", "")),
					}
				}
			}

			commissionPoolSwapper := checkState.Swap().GetSwapper(tx.CommissionCoin(), types.GetBaseCoinID())
			gasCoin := checkState.Coins().GetCoin(tx.CommissionCoin())
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
					abcTypes.EventAttribute{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(intruder[:]))},
				)
			}
			balance := checkState.Accounts().GetBalance(intruder, tx.CommissionCoin())
			if balance.Sign() == 1 {
				if balance.Cmp(commission) == -1 {
					commission = big.NewInt(0).Set(balance)
					if isGasCommissionFromPoolSwap {
						if !commissions.Coin.IsBaseCoin() {
							var resp *Response
							resp, commissionInBaseCoin, _ = CheckSwap(commissionPoolSwapper, checkState.Coins().GetCoin(tx.CommissionCoin()), checkState.Coins().GetCoin(0), commission, big.NewInt(0), false)
							if resp != nil {
								return *resp
							}
						}
						if commissionInBaseCoin == nil || commissionInBaseCoin.Sign() != 1 {
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
						var tagsCom *tagPoolChange
						var (
							poolIDCom  uint32
							detailsCom *swap.ChangeDetailsWithOrders
							ownersCom  []*swap.OrderDetail
						)
						commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swapper().PairSellWithOrders(tx.CommissionCoin(), types.GetBaseCoinID(), commission, big.NewInt(0))
						tagsCom = &tagPoolChange{
							PoolID:   poolIDCom,
							CoinIn:   tx.CommissionCoin(),
							ValueIn:  commission.String(),
							CoinOut:  types.GetBaseCoinID(),
							ValueOut: commissionInBaseCoin.String(),
							Orders:   detailsCom,
							// Sellers:  ownersCom,
						}
						for _, value := range ownersCom {
							deliverState.Accounts.AddBalance(value.Owner, tx.CommissionCoin(), value.ValueBigInt)
						}
						response.Tags = append(response.Tags,
							abcTypes.EventAttribute{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())})
					} else if !tx.CommissionCoin().IsBaseCoin() {
						deliverState.Coins.SubVolume(tx.CommissionCoin(), commission)
						deliverState.Coins.SubReserve(tx.CommissionCoin(), commissionInBaseCoin)
						response.Tags = append(response.Tags,
							abcTypes.EventAttribute{Key: []byte("tx.fail_fee_reserve"), Value: []byte(commissionInBaseCoin.String())})
					}

					deliverState.Accounts.SubBalance(intruder, tx.CommissionCoin(), commission)

					rewardPool.Add(rewardPool, commissionInBaseCoin)
					response.Tags = append(response.Tags,
						abcTypes.EventAttribute{Key: []byte("tx.fail_fee"), Value: []byte(commission.String())},
						abcTypes.EventAttribute{Key: []byte("tx.fail"), Value: []byte{49}, Index: true}, // "1"
					)
				}
			}
		} else if deliverState, ok := context.(*state.State); ok {
			if tx.Type == TypeCreateCoin || tx.Type == TypeCreateToken {
				dataCreateSymbol := tx.decodedData.(symbolCreator)
				symbolPrice := tx.MulGasPrice(dataCreateSymbol.PayForSymbol(commissions))
				if !commissions.Coin.IsBaseCoin() {
					var resp *Response
					resp, symbolPrice, _ = CheckSwap(checkState.Swap().GetSwapper(commissions.Coin, types.GetBaseCoinID()), checkState.Coins().GetCoin(commissions.Coin), checkState.Coins().GetCoin(0), symbolPrice, big.NewInt(0), false)
					if resp != nil {
						return *resp
					}
				}
				if symbolPrice == nil || symbolPrice.Sign() != 1 {
					return Response{
						Code: code.CommissionCoinNotSufficient,
						Log:  fmt.Sprint("Not possible to pay commission"),
						Info: EncodeError(code.NewCommissionCoinNotSufficient("", "")),
					}
				}
				rewardPool.Sub(rewardPool, symbolPrice)
				deliverState.Accounts.AddBalance([20]byte{}, 0, symbolPrice)
				response.Tags = append(response.Tags,
					abcTypes.EventAttribute{Key: []byte("tx.burned_for_symbol"), Value: []byte(symbolPrice.String())},
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
			abcTypes.EventAttribute{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(tx.decodedData.TxType())})), Index: true},
			abcTypes.EventAttribute{Key: []byte("tx.commission_coin"), Value: []byte(tx.CommissionCoin().String()), Index: true},
		)
		if tx.Type != TypeRedeemCheck {
			response.Tags = append(response.Tags, abcTypes.EventAttribute{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:])), Index: true})
		}
	}

	response.GasUsed = tx.Gas()
	response.GasWanted = response.GasUsed
	response.GasPrice = tx.GasPrice

	return response
}
