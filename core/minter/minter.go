package minter

import (
	abciTypes "github.com/tendermint/abci/types"
	"minter/mintdb"
	"github.com/tendermint/tmlibs/common"
	"os"
	"fmt"

	"minter/core/transaction"
	"minter/core/code"
	"minter/core/types"
	"minter/core/state"
	"math/big"
	"encoding/hex"
	"encoding/binary"
	"minter/formula"
	"minter/core/check"
	"bytes"
	"minter/crypto"
	"minter/crypto/sha3"
	"minter/rlp"
	"minter/core/rewards"
)

type Blockchain struct {
	abciTypes.BaseApplication

	db              *mintdb.LDBDatabase
	currentState    *state.StateDB
	rootHash        types.Hash
	height          uint64
	nextBlockHeight uint64
	rewards         *big.Int

	BaseCoin types.CoinSymbol
}

var (
	stateTableId = "state"
	appTableId   = "app"
)

func NewMinterBlockchain() *Blockchain {

	dir, err := os.Getwd()
	db, err := mintdb.NewLDBDatabase(dir+"/.data/minter", 1000, 1000)

	if err != nil {
		panic(err)
	}

	blockchain := &Blockchain{
		db:       db,
		BaseCoin: types.GetBaseCoin(),
	}

	blockchain.updateCurrentRootHash()
	blockchain.updateCurrentState()

	return blockchain
}

func (app *Blockchain) SetOption(req abciTypes.RequestSetOption) abciTypes.ResponseSetOption {
	return abciTypes.ResponseSetOption{}
}

func (app *Blockchain) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {

	coinbase := types.HexToAddress("Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f")
	app.currentState.SetBalance(coinbase, app.BaseCoin, big.NewInt(1e15))

	faucet := types.HexToAddress("Mxfe60014a6e9ac91618f5d1cab3fd58cded61ee99")
	app.currentState.SetBalance(faucet, app.BaseCoin, big.NewInt(1e15))

	for i := range req.Validators {
		app.currentState.CreateCandidate(coinbase, req.Validators[i].PubKey, 10, 1)
		app.currentState.SetCandidateOnline(req.Validators[i].PubKey)
	}

	return abciTypes.ResponseInitChain{}
}

func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.rewards = big.NewInt(0)

	return abciTypes.ResponseBeginBlock{}
}

func (app *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	app.nextBlockHeight = uint64(req.Height)

	validators, candidates := app.currentState.GetValidators()

	// calculate total power of validators
	totalPower := big.NewInt(0)
	for i := range candidates {
		totalPower.Add(totalPower, candidates[i].TotalStake)
	}

	// accumulate rewards
	for i := range candidates {
		reward := rewards.GetRewardForBlock(req.Height)
		reward.Add(reward, app.rewards)

		reward.Mul(reward, candidates[i].TotalStake)
		reward.Div(reward, totalPower)

		app.currentState.AddAccumReward(candidates[i].PubKey, reward)
	}

	// pay rewards
	if req.Height%5 == 0 {
		app.currentState.PayRewards()
	}

	// update validators
	if req.Height%5 == 0 {
		return abciTypes.ResponseEndBlock{
			ValidatorUpdates: validators,
		}
	}

	return abciTypes.ResponseEndBlock{}
}

func (app *Blockchain) Info(req abciTypes.RequestInfo) (resInfo abciTypes.ResponseInfo) {
	return abciTypes.ResponseInfo{
		LastBlockHeight:  int64(app.height),
		LastBlockAppHash: app.rootHash.Bytes(),
	}
}

func (app *Blockchain) DeliverTx(tx []byte) abciTypes.ResponseDeliverTx {

	decodedTx, err := transaction.DecodeFromBytes(tx)

	if err != nil {
		return abciTypes.ResponseDeliverTx{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	fmt.Println("deliver", decodedTx)

	response := app.RunTx(false, decodedTx)

	return abciTypes.ResponseDeliverTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Tags:      response.Tags,
		Fee:       response.Fee,
	}
}

func (app *Blockchain) CheckTx(tx []byte) abciTypes.ResponseCheckTx {

	decodedTx, err := transaction.DecodeFromBytes(tx)

	if err != nil {
		return abciTypes.ResponseCheckTx{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	response := app.RunTx(true, decodedTx)

	return abciTypes.ResponseCheckTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Tags:      response.Tags,
		Fee:       response.Fee,
	}
}

func (app *Blockchain) Commit() abciTypes.ResponseCommit {

	hash, _ := app.currentState.Commit(false)
	app.currentState.Database().TrieDB().Commit(hash, true)

	appTable := mintdb.NewTable(app.db, appTableId)
	err := appTable.Put([]byte("root"), hash.Bytes())

	if err != nil {
		panic(err)
	}

	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height[:], app.nextBlockHeight)
	err = appTable.Put([]byte("height"), height[:])

	if err != nil {
		panic(err)
	}

	// TODO: clear candidates list

	app.updateCurrentRootHash()
	app.updateCurrentState()

	return abciTypes.ResponseCommit{
		Data: app.rootHash.Bytes(),
	}
}

func (app *Blockchain) Query(reqQuery abciTypes.RequestQuery) abciTypes.ResponseQuery {
	return abciTypes.ResponseQuery{}
}

func (app *Blockchain) Stop() {
	app.db.Close()
}

func (app *Blockchain) updateCurrentRootHash() {
	appTable := mintdb.NewTable(app.db, appTableId)

	result, _ := appTable.Get([]byte("root"))
	app.rootHash = types.BytesToHash(result)

	result, err := appTable.Get([]byte("height"))
	if err == nil {
		app.height = binary.BigEndian.Uint64(result)
	} else {
		app.height = 0
	}
}

func (app *Blockchain) updateCurrentState() {
	stateTable := mintdb.NewTable(app.db, stateTableId)
	cState, _ := state.New(app.rootHash, state.NewDatabase(stateTable))

	app.currentState = cState
}

func (app *Blockchain) CurrentState() *state.StateDB {
	return app.currentState
}

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

func (app *Blockchain) RunTx(isCheck bool, tx *transaction.Transaction) Response {
	// TODO: separate State Objects for checking and for delivering

	sender, _ := tx.Sender()

	if expectedNonce := app.currentState.GetNonce(sender) + 1; expectedNonce != tx.Nonce {
		return Response{
			Code: code.WrongNonce,
			Log:  fmt.Sprintf("Unexpected nonce. Expected: %d, got %d.", expectedNonce, tx.Nonce)}
	}

	// TODO: add "set candidate online/offline", unbound

	switch tx.Type {
	case transaction.TypeDeclareCandidacy:

		data := tx.GetDecodedData().(transaction.DeclareCandidacyData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		totalTxCost := big.NewInt(0).Add(data.Stake, commission)

		if app.currentState.GetBalance(sender, app.BaseCoin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		if app.currentState.CandidateExists(data.PubKey) {
			return Response{
				Code: code.CandidateExists,
				Log:  fmt.Sprintf("Candidate with such public key already exists")}
		}

		if data.Commission < 0 || data.Commission > 100 {
			return Response{
				Code: code.WrongCommission,
				Log:  fmt.Sprintf("Commission should be between 0 and 100")}
		}

		// TODO: limit number of candidates to prevent flooding

		if !isCheck {
			app.rewards.Add(app.rewards, commission)

			app.currentState.SubBalance(sender, app.BaseCoin, totalTxCost)
			app.currentState.CreateCandidate(data.Address, data.PubKey, data.Commission, uint(app.nextBlockHeight))
			app.currentState.Delegate(sender, data.PubKey, data.Stake)
			app.currentState.SetNonce(sender, tx.Nonce)
		}

		return Response{
			Code:      code.OK,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}
	case transaction.TypeDelegate:

		data := tx.GetDecodedData().(transaction.DelegateData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		totalTxCost := big.NewInt(0).Add(data.Stake, commission)

		if app.currentState.GetBalance(sender, app.BaseCoin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		if !app.currentState.CandidateExists(data.PubKey) {
			return Response{
				Code: code.CandidateNotFound,
				Log:  fmt.Sprintf("Candidate with such public key not found")}
		}


		if !isCheck {
			app.rewards.Add(app.rewards, commission)

			app.currentState.SubBalance(sender, app.BaseCoin, totalTxCost)
			app.currentState.Delegate(sender, data.PubKey, data.Stake)
			app.currentState.SetNonce(sender, tx.Nonce)
		}

		return Response{
			Code:      code.OK,
			GasUsed:   tx.Gas(),
			GasWanted: tx.Gas(),
		}
	case transaction.TypeSend:

		data := tx.GetDecodedData().(transaction.SendData)

		if !app.currentState.CoinExists(data.Coin) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if data.Coin != app.BaseCoin {
			coin := app.currentState.GetStateCoin(data.Coin)
			commission = formula.CalculateBuyDeposit(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
		}

		totalTxCost := big.NewInt(0).Add(data.Value, commission)

		if app.currentState.GetBalance(sender, data.Coin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		// deliver TX

		if !isCheck {
			app.rewards.Add(app.rewards, commissionInBaseCoin)

			if data.Coin != app.BaseCoin {
				app.currentState.SubCoinVolume(data.Coin, commission)
				app.currentState.SubCoinReserve(data.Coin, commissionInBaseCoin)
			}

			app.currentState.SubBalance(sender, data.Coin, totalTxCost)
			app.currentState.AddBalance(data.To, data.Coin, data.Value)
			app.currentState.SetNonce(sender, tx.Nonce)
		}

		tags := common.KVPairs{
			common.KVPair{Key: []byte("tx.type"), Value: []byte{transaction.TypeSend}},
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

	case transaction.TypeRedeemCheck:

		data := tx.GetDecodedData().(transaction.RedeemCheckData)

		decodedCheck, _ := check.DecodeFromBytes(data.RawCheck)
		checkSender, _ := decodedCheck.Sender()

		if !app.currentState.CoinExists(decodedCheck.Coin) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		if decodedCheck.DueBlock < app.nextBlockHeight {
			return Response{
				Code: code.CheckExpired,
				Log:  fmt.Sprintf("Check expired")}
		}

		if app.currentState.IsCheckUsed(decodedCheck) {
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
		rlp.Encode(hw,[]interface{}{
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
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if decodedCheck.Coin != app.BaseCoin {
			coin := app.currentState.GetStateCoin(decodedCheck.Coin)
			commission = formula.CalculateBuyDeposit(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
		}

		totalTxCost := big.NewInt(0).Add(decodedCheck.Value, commission)

		if app.currentState.GetBalance(checkSender, decodedCheck.Coin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s. Wanted %d ", checkSender.String(), totalTxCost)}
		}

		// deliver TX

		if !isCheck {
			app.currentState.UseCheck(decodedCheck)
			app.rewards.Add(app.rewards, commissionInBaseCoin)

			if decodedCheck.Coin != app.BaseCoin {
				app.currentState.SubCoinVolume(decodedCheck.Coin, commission)
				app.currentState.SubCoinReserve(decodedCheck.Coin, commissionInBaseCoin)
			}

			app.currentState.SubBalance(checkSender, decodedCheck.Coin, totalTxCost)
			app.currentState.AddBalance(sender, decodedCheck.Coin, decodedCheck.Value)
			app.currentState.SetNonce(sender, tx.Nonce)
		}

		tags := common.KVPairs{
			common.KVPair{Key: []byte("tx.type"), Value: []byte{transaction.TypeRedeemCheck}},
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

	case transaction.TypeConvert:

		data := tx.GetDecodedData().(transaction.ConvertData)

		if data.FromCoinSymbol == data.ToCoinSymbol {
			return Response{
				Code: code.CrossConvert,
				Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin")}
		}

		if !app.currentState.CoinExists(data.FromCoinSymbol) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		if !app.currentState.CoinExists(data.ToCoinSymbol) {
			return Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin not exists")}
		}

		commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if data.FromCoinSymbol != app.BaseCoin {
			coin := app.currentState.GetStateCoin(data.FromCoinSymbol)
			commission = formula.CalculateBuyDeposit(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
		}

		totalTxCost := big.NewInt(0).Add(data.Value, commission)

		if app.currentState.GetBalance(sender, data.FromCoinSymbol).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		// deliver TX

		if !isCheck {
			app.rewards.Add(app.rewards, commissionInBaseCoin)

			app.currentState.SubBalance(sender, data.FromCoinSymbol, totalTxCost)

			if data.FromCoinSymbol != app.BaseCoin {
				app.currentState.SubCoinVolume(data.FromCoinSymbol, commission)
				app.currentState.SubCoinReserve(data.FromCoinSymbol, commissionInBaseCoin)
			}
		}

		var value *big.Int

		if data.FromCoinSymbol == app.BaseCoin {
			coin := app.currentState.GetStateCoin(data.ToCoinSymbol).Data()

			value = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.Value)

			if !isCheck {
				app.currentState.AddCoinVolume(data.ToCoinSymbol, value)
				app.currentState.AddCoinReserve(data.ToCoinSymbol, data.Value)
			}
		} else if data.ToCoinSymbol == app.BaseCoin {
			coin := app.currentState.GetStateCoin(data.FromCoinSymbol).Data()

			value = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.Value)

			if !isCheck {
				app.currentState.SubCoinVolume(data.FromCoinSymbol, data.Value)
				app.currentState.SubCoinReserve(data.FromCoinSymbol, value)
			}
		} else {
			coinFrom := app.currentState.GetStateCoin(data.FromCoinSymbol).Data()
			coinTo := app.currentState.GetStateCoin(data.ToCoinSymbol).Data()

			val := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, data.Value)
			value = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, val)

			if !isCheck {
				app.currentState.AddCoinVolume(data.ToCoinSymbol, data.Value)
				app.currentState.SubCoinVolume(data.FromCoinSymbol, value)

				app.currentState.AddCoinReserve(data.ToCoinSymbol, data.Value)
				app.currentState.SubCoinReserve(data.ToCoinSymbol, value)
			}
		}

		if !isCheck {
			app.currentState.AddBalance(sender, data.ToCoinSymbol, value)
			app.currentState.SetNonce(sender, tx.Nonce)
		}

		tags := common.KVPairs{
			common.KVPair{Key: []byte("tx.type"), Value: []byte{transaction.TypeConvert}},
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

	case transaction.TypeCreateCoin:

		data := tx.GetDecodedData().(transaction.CreateCoinData)

		commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))

		totalTxCost := big.NewInt(0).Add(data.InitialReserve, commission)

		if app.currentState.GetBalance(sender, app.BaseCoin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
		}

		if app.currentState.CoinExists(data.Symbol) {
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
			app.rewards.Add(app.rewards, commission)

			app.currentState.SubBalance(sender, app.BaseCoin, totalTxCost)
			app.currentState.CreateCoin(data.Symbol, data.Name, data.InitialAmount, data.ConstantReserveRatio, data.InitialReserve, sender)
			app.currentState.AddBalance(sender, data.Symbol, data.InitialAmount)
			app.currentState.SetNonce(sender, tx.Nonce)
		}

		tags := common.KVPairs{
			common.KVPair{Key: []byte("tx.type"), Value: []byte{transaction.TypeCreateCoin}},
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
