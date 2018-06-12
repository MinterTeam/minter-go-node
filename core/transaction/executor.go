package transaction

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/tendermint/tmlibs/common"
	"math/big"
	"minter/core/check"
	"minter/core/code"
	"minter/core/state"
	"minter/core/types"
	"minter/crypto"
	"minter/crypto/sha3"
	"minter/formula"
	"minter/rlp"
	"regexp"
)

var (
	CommissionMultiplier = big.NewInt(10e8)
)

type Response struct {
	Code      uint32          `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Data      []byte          `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Log       string          `protobuf:"bytes,3,opt,name=log,proto3" json:"log,omitempty"`
	Info      string          `protobuf:"bytes,4,opt,name=info,proto3" json:"info,omitempty"`
	GasWanted int64           `protobuf:"varint,5,opt,name=gas_wanted,json=gasWanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   int64           `protobuf:"varint,6,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`
	Tags      []common.KVPair `protobuf:"bytes,7,rep,name=tags" json:"tags,omitempty"`
	Fee       common.KI64Pair `protobuf:"bytes,8,opt,name=fee" json:"fee"`
}

func RunTx(context *state.StateDB, isCheck bool, tx *Transaction, rewardPull *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	// TODO: deal smth about multiple outgoing transactions from one sender
	if expectedNonce := context.GetNonce(sender) + 1; expectedNonce != tx.Nonce {
		return Response{
			Code: code.WrongNonce,
			Log:  fmt.Sprintf("Unexpected nonce. Expected: %d, got %d.", expectedNonce, tx.Nonce)}
	}

	if len(tx.Payload)+len(tx.ServiceData) > 1024 {
		return Response{
			Code: code.TooLongPayload,
			Log:  fmt.Sprintf("Too long Payload + ServiceData. Max 1024 bytes.")}
	}

	switch tx.Type {
	case TypeDeclareCandidacy:

		data := tx.GetDecodedData().(DeclareCandidacyData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commission.Mul(commission, CommissionMultiplier)
		totalTxCost := big.NewInt(0).Add(data.Stake, commission)

		if context.GetBalance(sender, types.GetBaseCoin()).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		if context.CandidateExists(data.PubKey) {
			return Response{
				Code: code.CandidateExists,
				Log:  fmt.Sprintf("Candidate with such public key (%x) already exists", data.PubKey)}
		}

		if data.Commission < 0 || data.Commission > 100 {
			return Response{
				Code: code.WrongCommission,
				Log:  fmt.Sprintf("Commission should be between 0 and 100")}
		}

		// TODO: limit number of candidates to prevent flooding

		if !isCheck {
			rewardPull.Add(rewardPull, commission)

			context.SubBalance(sender, types.GetBaseCoin(), totalTxCost)
			context.CreateCandidate(data.Address, data.PubKey, data.Commission, uint(currentBlock), data.Stake)
			context.SetNonce(sender, tx.Nonce)
		}

		return Response{
			Code:      code.OK,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}
	case TypeSetCandidateOnline:

		data := tx.GetDecodedData().(SetCandidateOnData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commission.Mul(commission, CommissionMultiplier)

		if context.GetBalance(sender, types.GetBaseCoin()).Cmp(commission) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), commission)}
		}

		if !context.CandidateExists(data.PubKey) {
			return Response{
				Code: code.CandidateNotFound,
				Log:  fmt.Sprintf("Candidate with such public key (%x) not found", data.PubKey)}
		}

		candidate := context.GetStateCandidate(data.PubKey)

		if bytes.Compare(candidate.CandidateAddress.Bytes(), sender.Bytes()) != 0 {
			return Response{
				Code: code.IsNotOwnerOfCandidate,
				Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
		}

		if !isCheck {
			rewardPull.Add(rewardPull, commission)

			context.SubBalance(sender, types.GetBaseCoin(), commission)
			context.SetCandidateOnline(data.PubKey)
			context.SetNonce(sender, tx.Nonce)
		}

		return Response{
			Code:      code.OK,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}
	case TypeSetCandidateOffline:

		data := tx.GetDecodedData().(SetCandidateOffData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commission.Mul(commission, CommissionMultiplier)

		if context.GetBalance(sender, types.GetBaseCoin()).Cmp(commission) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), commission)}
		}

		if !context.CandidateExists(data.PubKey) {
			return Response{
				Code: code.CandidateNotFound,
				Log:  fmt.Sprintf("Candidate with such public key not found")}
		}

		candidate := context.GetStateCandidate(data.PubKey)

		if bytes.Compare(candidate.CandidateAddress.Bytes(), sender.Bytes()) != 0 {
			return Response{
				Code: code.IsNotOwnerOfCandidate,
				Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
		}

		if !isCheck {
			rewardPull.Add(rewardPull, commission)

			context.SubBalance(sender, types.GetBaseCoin(), commission)
			context.SetCandidateOffline(data.PubKey)
			context.SetNonce(sender, tx.Nonce)
		}

		return Response{
			Code:      code.OK,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}
	case TypeDelegate:

		data := tx.GetDecodedData().(DelegateData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commission.Mul(commission, CommissionMultiplier)
		totalTxCost := big.NewInt(0).Add(data.Stake, commission)

		if context.GetBalance(sender, types.GetBaseCoin()).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		if !context.CandidateExists(data.PubKey) {
			return Response{
				Code: code.CandidateNotFound,
				Log:  fmt.Sprintf("Candidate with such public key not found")}
		}

		if !isCheck {
			rewardPull.Add(rewardPull, commission)

			context.SubBalance(sender, types.GetBaseCoin(), totalTxCost)
			context.Delegate(sender, data.PubKey, data.Stake)
			context.SetNonce(sender, tx.Nonce)
		}

		return Response{
			Code:      code.OK,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}
	case TypeUnbond:

		data := tx.GetDecodedData().(UnbondData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commission.Mul(commission, CommissionMultiplier)

		if context.GetBalance(sender, types.GetBaseCoin()).Cmp(commission) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), commission)}
		}

		if !context.CandidateExists(data.PubKey) {
			return Response{
				Code: code.CandidateNotFound,
				Log:  fmt.Sprintf("Candidate with such public key not found")}
		}

		candidate := context.GetStateCandidate(data.PubKey)

		stake := candidate.GetStakeOfAddress(sender)

		if stake == nil {
			return Response{
				Code: code.StakeNotFound,
				Log:  fmt.Sprintf("Stake of current user not found")}
		}

		if stake.Value.Cmp(data.Value) < 0 {
			return Response{
				Code: code.InsufficientStake,
				Log:  fmt.Sprintf("Insufficient stake for sender account")}
		}

		if !isCheck {
			// now + 31 days
			unboundAtBlock := currentBlock + 518400

			rewardPull.Add(rewardPull, commission)

			context.SubBalance(sender, types.GetBaseCoin(), commission)
			context.SubStake(sender, data.PubKey, data.Value)
			context.GetOrNewStateFrozenFunds(unboundAtBlock).AddFund(sender, data.PubKey, data.Value)
			context.SetNonce(sender, tx.Nonce)
		}

		return Response{
			Code:      code.OK,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}
	case TypeSend:

		data := tx.GetDecodedData().(SendData)

		if !context.CoinExists(data.Coin) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if data.Coin != types.GetBaseCoin() {
			coin := context.GetStateCoin(data.Coin)

			if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
				return Response{
					Code: code.CoinReserveNotSufficient,
					Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
			}

			commission = formula.CalculateBuyDeposit(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
		}

		totalTxCost := big.NewInt(0).Add(data.Value, commission)

		if context.GetBalance(sender, data.Coin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		// deliver TX

		if !isCheck {
			rewardPull.Add(rewardPull, commissionInBaseCoin)

			if data.Coin != types.GetBaseCoin() {
				context.SubCoinVolume(data.Coin, commission)
				context.SubCoinReserve(data.Coin, commissionInBaseCoin)
			}

			context.SubBalance(sender, data.Coin, totalTxCost)
			context.AddBalance(data.To, data.Coin, data.Value)
			context.SetNonce(sender, tx.Nonce)
		}

		tags := common.KVPairs{
			common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeSend}},
			common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			common.KVPair{Key: []byte("tx.to"), Value: []byte(hex.EncodeToString(data.To[:]))},
			common.KVPair{Key: []byte("tx.coin"), Value: []byte(data.Coin.String())},
		}

		return Response{
			Code:      code.OK,
			Tags:      tags,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}

	case TypeRedeemCheck:

		data := tx.GetDecodedData().(RedeemCheckData)

		decodedCheck, _ := check.DecodeFromBytes(data.RawCheck)
		checkSender, _ := decodedCheck.Sender()

		if !context.CoinExists(decodedCheck.Coin) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		if decodedCheck.DueBlock < uint64(currentBlock) {
			return Response{
				Code: code.CheckExpired,
				Log:  fmt.Sprintf("Check expired")}
		}

		if context.IsCheckUsed(decodedCheck) {
			return Response{
				Code: code.CheckUsed,
				Log:  fmt.Sprintf("Check already redeemed")}
		}

		// fixed potential problem with making too high commission for sender
		if tx.GasPrice.Cmp(big.NewInt(1)) == 1 {
			return Response{
				Code: code.TooHighGasPrice,
				Log:  fmt.Sprintf("Gas price for check is limited to 1")}
		}

		lockPublicKey, _ := decodedCheck.LockPubKey()

		var senderAddressHash types.Hash
		hw := sha3.NewKeccak256()
		rlp.Encode(hw, []interface{}{
			sender,
		})
		hw.Sum(senderAddressHash[:0])

		pub, _ := crypto.Ecrecover(senderAddressHash[:], data.Proof[:])

		if bytes.Compare(lockPublicKey, pub) != 0 {
			return Response{
				Code: code.CheckInvalidLock,
				Log:  fmt.Sprintf("Invalid proof")}
		}

		commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if decodedCheck.Coin != types.GetBaseCoin() {
			coin := context.GetStateCoin(decodedCheck.Coin)
			commission = formula.CalculateBuyDeposit(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
		}

		totalTxCost := big.NewInt(0).Add(decodedCheck.Value, commission)

		if context.GetBalance(checkSender, decodedCheck.Coin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s. Wanted %d ", checkSender.String(), totalTxCost)}
		}

		// deliver TX

		if !isCheck {
			context.UseCheck(decodedCheck)
			rewardPull.Add(rewardPull, commissionInBaseCoin)

			if decodedCheck.Coin != types.GetBaseCoin() {
				context.SubCoinVolume(decodedCheck.Coin, commission)
				context.SubCoinReserve(decodedCheck.Coin, commissionInBaseCoin)
			}

			context.SubBalance(checkSender, decodedCheck.Coin, totalTxCost)
			context.AddBalance(sender, decodedCheck.Coin, decodedCheck.Value)
			context.SetNonce(sender, tx.Nonce)
		}

		tags := common.KVPairs{
			common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeRedeemCheck}},
			common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(checkSender[:]))},
			common.KVPair{Key: []byte("tx.to"), Value: []byte(hex.EncodeToString(sender[:]))},
			common.KVPair{Key: []byte("tx.coin"), Value: []byte(decodedCheck.Coin.String())},
		}

		return Response{
			Code:      code.OK,
			Tags:      tags,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}

	case TypeConvert:

		data := tx.GetDecodedData().(ConvertData)

		if data.FromCoinSymbol == data.ToCoinSymbol {
			return Response{
				Code: code.CrossConvert,
				Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin")}
		}

		if !context.CoinExists(data.FromCoinSymbol) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		if !context.CoinExists(data.ToCoinSymbol) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if data.FromCoinSymbol != types.GetBaseCoin() {
			coin := context.GetStateCoin(data.FromCoinSymbol)

			if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
				return Response{
					Code: code.CoinReserveNotSufficient,
					Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
			}

			commission = formula.CalculateBuyDeposit(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
		}

		totalTxCost := big.NewInt(0).Add(data.Value, commission)

		if context.GetBalance(sender, data.FromCoinSymbol).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		// deliver TX

		if !isCheck {
			rewardPull.Add(rewardPull, commissionInBaseCoin)

			context.SubBalance(sender, data.FromCoinSymbol, totalTxCost)

			if data.FromCoinSymbol != types.GetBaseCoin() {
				context.SubCoinVolume(data.FromCoinSymbol, commission)
				context.SubCoinReserve(data.FromCoinSymbol, commissionInBaseCoin)
			}
		}

		var value *big.Int

		if data.FromCoinSymbol == types.GetBaseCoin() {
			coin := context.GetStateCoin(data.ToCoinSymbol).Data()

			value = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.Value)

			if !isCheck {
				context.AddCoinVolume(data.ToCoinSymbol, value)
				context.AddCoinReserve(data.ToCoinSymbol, data.Value)
			}
		} else if data.ToCoinSymbol == types.GetBaseCoin() {
			coin := context.GetStateCoin(data.FromCoinSymbol).Data()

			value = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.Value)

			if !isCheck {
				context.SubCoinVolume(data.FromCoinSymbol, data.Value)
				context.SubCoinReserve(data.FromCoinSymbol, value)
			}
		} else {
			coinFrom := context.GetStateCoin(data.FromCoinSymbol).Data()
			coinTo := context.GetStateCoin(data.ToCoinSymbol).Data()

			val := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, data.Value)
			value = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, val)

			if !isCheck {
				context.AddCoinVolume(data.ToCoinSymbol, data.Value)
				context.SubCoinVolume(data.FromCoinSymbol, value)

				context.AddCoinReserve(data.ToCoinSymbol, data.Value)
				context.SubCoinReserve(data.ToCoinSymbol, value)
			}
		}

		if !isCheck {
			context.AddBalance(sender, data.ToCoinSymbol, value)
			context.SetNonce(sender, tx.Nonce)
		}

		tags := common.KVPairs{
			common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeConvert}},
			common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			common.KVPair{Key: []byte("tx.coin_to"), Value: []byte(data.ToCoinSymbol.String())},
			common.KVPair{Key: []byte("tx.coin_from"), Value: []byte(data.FromCoinSymbol.String())},
			common.KVPair{Key: []byte("tx.return"), Value: value.Bytes()},
		}

		return Response{
			Code:      code.OK,
			Tags:      tags,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}

	case TypeCreateCoin:

		data := tx.GetDecodedData().(CreateCoinData)

		if match, _ := regexp.MatchString("^[A-Z0-9]{3,10}$", data.Symbol.String()); !match {
			return Response{
				Code: code.InvalidCoinSymbol,
				Log:  fmt.Sprintf("Invalid coin symbol. Should be ^[A-Z0-9]{3,10}$")}
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

		// deliver TX

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

	default:
		return Response{Code: code.UnknownTransactionType}
	}

	return Response{Code: code.UnknownTransactionType}
}
