package minter

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/genesis"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/mintdb"
	abciTypes "github.com/tendermint/abci/types"
	"github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/rpc/lib/client"
	"math/big"
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

	maxTxLength    = 1024
	teamAddress    = types.HexToAddress("Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f")
	airdropAddress = types.HexToAddress("Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f")
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
		app.currentStateDeliver.CreateCandidate(genesisState.FirstValidatorAddress, validator.PubKey.Data, 10, 1, big.NewInt(1))
		app.currentStateDeliver.SetCandidateOnline(validator.PubKey.Data)
		app.activeValidators = append(app.activeValidators, validator)
	}

	return abciTypes.ResponseInitChain{}
}

func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.height = uint64(req.Header.Height)
	app.rewards = big.NewInt(0)

	// clear absent candidates
	app.absentCandidates = make(map[string]bool)

	// give penalty to absent validators
	for _, v := range req.Validators {
		if v.SignedLastBlock {
			app.currentStateDeliver.SetValidatorPresent(v.Validator.PubKey.Data)
		} else {
			app.currentStateDeliver.SetValidatorAbsent(v.Validator.PubKey.Data)
			app.absentCandidates[v.Validator.PubKey.String()] = true
		}
	}

	// give penalty to Byzantine validators
	for _, v := range req.ByzantineValidators {
		app.currentStateDeliver.PunishByzantineCandidate(v.Validator.PubKey.Data)
		app.currentStateDeliver.RemoveFrozenFundsWithPubKey(app.height, app.height+518400, v.Validator.PubKey.Data)
	}

	// apply frozen funds
	frozenFunds := app.currentStateDeliver.GetStateFrozenFunds(app.height)
	if frozenFunds != nil {
		for _, item := range frozenFunds.List() {
			app.currentStateDeliver.SetBalance(item.Address, item.Coin, item.Value)
		}

		frozenFunds.Delete()
	}

	// distributions:
	if app.height <= 3110400*6 && app.height%3110400 == 0 { // team distribution
		value := big.NewInt(300000000) // 300 000 000 bip (3%)
		app.currentStateDeliver.AddBalance(teamAddress, types.GetBaseCoin(), helpers.BipToPip(value))
	}

	if app.height <= 3110400*10 && app.height%3110400 == 0 { // airdrop distribution
		value := big.NewInt(500000000) // 500 000 000 bip (5%)
		app.currentStateDeliver.AddBalance(airdropAddress, types.GetBaseCoin(), helpers.BipToPip(value))
	}

	return abciTypes.ResponseBeginBlock{}
}

func (app *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {

	validatorsCount := validators.GetValidatorsCountForBlock(app.height)

	newValidators, newCandidates := app.currentStateDeliver.GetValidators(validatorsCount)

	// calculate total power of validators
	totalPower := big.NewInt(0)
	for _, candidate := range newCandidates {

		// skip if candidate is absent
		if app.absentCandidates[candidate.PubKey.String()] {
			continue
		}

		totalPower.Add(totalPower, candidate.TotalBipStake)
	}

	// accumulate rewards
	for _, candidate := range newCandidates {

		// skip if candidate is absent
		if app.absentCandidates[candidate.PubKey.String()] {
			continue
		}

		reward := rewards.GetRewardForBlock(uint64(app.height))

		reward.Add(reward, app.rewards)

		reward.Mul(reward, candidate.TotalBipStake)
		reward.Div(reward, totalPower)

		app.currentStateDeliver.AddAccumReward(candidate.PubKey, reward)
	}

	// pay rewards
	if app.height%5 == 0 {
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
			if bytes.Compare(validator.PubKey.Data, newValidator.PubKey.Data) == 0 {
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

	if len(tx) > maxTxLength {
		return abciTypes.ResponseDeliverTx{
			Code: code.TxTooLarge,
			Log:  "TX length is over 1024 bytes"}
	}

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

	if len(tx) > maxTxLength {
		return abciTypes.ResponseCheckTx{
			Code: code.TxTooLarge,
			Log:  "TX length is over 1024 bytes"}
	}

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

	// todo: make provider
	appTable := mintdb.NewTable(app.db, appTableId)
	err := appTable.Put([]byte("root"), hash.Bytes())

	if err != nil {
		panic(err)
	}

	// todo: make provider
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

	// todo: make provider
	result, _ := appTable.Get([]byte("root"))
	app.rootHash = types.BytesToHash(result)

	// todo: make provider
	result, err := appTable.Get([]byte("height"))
	if err == nil {
		app.height = binary.BigEndian.Uint64(result)
	} else {
		app.height = 0
	}
}

func (app *Blockchain) updateCurrentState() {
	app.currentStateDeliver, _ = state.New(app.rootHash, state.NewDatabase(mintdb.NewTable(app.db, stateTableId)))
	app.currentStateCheck, _ = state.New(app.rootHash, state.NewDatabase(mintdb.NewTable(app.db, stateTableId)))
}

func (app *Blockchain) CurrentState() *state.StateDB {
	return app.currentStateCheck
}

func (app *Blockchain) GetStateForHeight(height int) (*state.StateDB, error) {
	client := rpcclient.NewJSONRPCClient(*utils.TendermintRpcAddrFlag)
	core_types.RegisterAmino(client.Codec())

	result := new(core_types.ResultBlock)
	_, err := client.Call("block", map[string]interface{}{
		"height": height,
	}, result)

	if err != nil {
		return nil, err
	}

	var stateHash types.Hash

	copy(stateHash[:], result.Block.AppHash.Bytes())

	stateTable := mintdb.NewTable(app.db, stateTableId)
	return state.New(stateHash, state.NewDatabase(stateTable))
}

func (app *Blockchain) Height() uint64 {
	return app.height
}
