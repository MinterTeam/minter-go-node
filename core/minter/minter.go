package minter

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	abciTypes "github.com/tendermint/abci/types"
	"math/big"
	"minter/cmd/utils"
	"minter/core/code"
	"minter/core/rewards"
	"minter/core/state"
	"minter/core/transaction"
	"minter/core/types"
	"minter/genesis"
	"minter/mintdb"
)

type Blockchain struct {
	abciTypes.BaseApplication

	db                  *mintdb.LDBDatabase
	currentStateDeliver *state.StateDB
	currentStateCheck   *state.StateDB
	rootHash            types.Hash
	height              uint64
	rewards             *big.Int
	activeValidators    abciTypes.Validators
	absentCandidates    map[string]bool

	BaseCoin types.CoinSymbol
}

var (
	blockchain *Blockchain

	stateTableId = "state"
	appTableId   = "app"
)

func NewMinterBlockchain() *Blockchain {

	db, err := mintdb.NewLDBDatabase(utils.GetMinterHome()+"/data", 1000, 1000)

	if err != nil {
		panic(err)
	}

	blockchain = &Blockchain{
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
	var genesisState genesis.AppState
	err := json.Unmarshal(req.AppStateBytes, &genesisState)

	if err != nil {
		panic(err)
	}

	for _, account := range genesisState.InitialBalances {
		for coinSymbol, value := range account.Balance {
			bigIntValue, _ := big.NewInt(0).SetString(value, 10)
			var coin types.CoinSymbol
			copy(coin[:], []byte(coinSymbol))
			app.currentStateDeliver.SetBalance(account.Address, coin, bigIntValue)
		}
	}

	for _, validator := range req.Validators {
		app.currentStateDeliver.CreateCandidate(genesisState.FirstValidatorAddress, validator.PubKey, 10, 1)
		app.currentStateDeliver.SetCandidateOnline(validator.PubKey)
		app.activeValidators = append(app.activeValidators, validator)
	}

	return abciTypes.ResponseInitChain{}
}

func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.rewards = big.NewInt(0)

	// todo: calculate validators count from current block height
	validatorsCount := 10

	_, candidates := app.currentStateDeliver.GetValidators(validatorsCount)

	// clear absent candidates
	app.absentCandidates = make(map[string]bool)

	// give penalty to absent validators
	for _, i := range req.AbsentValidators {
		app.currentStateDeliver.SetValidatorAbsent(candidates[i].PubKey)
		app.absentCandidates[candidates[i].PubKey.String()] = true
	}

	// give penalty to Byzantine validators
	for _, b := range req.ByzantineValidators {
		app.currentStateDeliver.PunishByzantineCandidate(b.PubKey)
		app.currentStateDeliver.RemoveFrozenFundsWithPubKey(app.height, app.height + 518400, b.PubKey)
	}

	return abciTypes.ResponseBeginBlock{}
}

func (app *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	app.height = uint64(req.Height)

	// apply frozen funds
	frozenFunds := app.currentStateDeliver.GetStateFrozenFunds(req.Height)
	if frozenFunds != nil {
		for _, item := range frozenFunds.List() {
			app.currentStateDeliver.SetBalance(item.Address, app.BaseCoin, item.Value)
		}

		frozenFunds.Delete()
	}

	// todo: calculate validators count from current block height
	validatorsCount := 10

	newValidators, newCandidates := app.currentStateDeliver.GetValidators(validatorsCount)

	// calculate total power of validators
	totalPower := big.NewInt(0)
	for _, candidate := range newCandidates {

		// skip if candidate is absent
		if app.absentCandidates[candidate.PubKey.String()] {
			continue
		}

		totalPower.Add(totalPower, candidate.TotalStake)
	}

	// accumulate rewards
	for _, candidate := range newCandidates {

		// skip if candidate is absent
		if app.absentCandidates[candidate.PubKey.String()] {
			continue
		}

		reward := rewards.GetRewardForBlock(uint64(req.Height))

		reward.Add(reward, app.rewards)

		reward.Mul(reward, candidate.TotalStake)
		reward.Div(reward, totalPower)

		app.currentStateDeliver.AddAccumReward(candidate.PubKey, reward)
	}

	// pay rewards
	if req.Height%5 == 0 {
		app.currentStateDeliver.PayRewards()
	}

	// update validators
	defer func() {
		app.activeValidators = newValidators
	}()

	updates := newValidators

	for _, validator := range app.activeValidators {
		persisted := false
		for _, newValidator := range newValidators {
			if bytes.Compare(validator.PubKey, newValidator.PubKey) == 0 {
				persisted = true
				break
			}
		}

		// remove validator
		if !persisted {
			updates = append(updates, abciTypes.Validator{
				PubKey: validator.PubKey,
				Power:  0,
			})
		}
	}

	return abciTypes.ResponseEndBlock{
		ValidatorUpdates: updates,
	}
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

	response := transaction.RunTx(app.currentStateDeliver, false, decodedTx, app.rewards, app.height)

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

	// todo: lock while producing block

	decodedTx, err := transaction.DecodeFromBytes(tx)

	if err != nil {
		return abciTypes.ResponseCheckTx{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	response := transaction.RunTx(app.currentStateCheck, true, decodedTx, nil, app.height)

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

	hash, _ := app.currentStateDeliver.Commit(false)
	app.currentStateDeliver.Database().TrieDB().Commit(hash, true)

	appTable := mintdb.NewTable(app.db, appTableId)
	err := appTable.Put([]byte("root"), hash.Bytes())

	if err != nil {
		panic(err)
	}

	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height[:], app.height)
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
	app.currentStateDeliver, _ = state.New(app.rootHash, state.NewDatabase(stateTable))
	app.currentStateCheck, _ = state.New(app.rootHash, state.NewDatabase(stateTable))
}

func (app *Blockchain) CurrentState() *state.StateDB {
	return app.currentStateCheck
}
