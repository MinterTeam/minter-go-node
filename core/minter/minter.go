package minter

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/genesis"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/mintdb"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/node"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"math/big"
	"os"
)

type Blockchain struct {
	abciTypes.BaseApplication

	db                 *mintdb.LDBDatabase
	stateDeliver       *state.StateDB
	stateCheck         *state.StateDB
	rootHash           types.Hash
	height             uint64
	rewards            *big.Int
	activeValidators   abciTypes.ValidatorUpdates
	validatorsStatuses map[[20]byte]int8
	tendermintRPC      *rpc.Local

	BaseCoin types.CoinSymbol
}

const (
	ValidatorPresent = 1
	ValidatorAbsent  = 2

	stateTableId = "state"
	appTableId   = "app"
)

var (
	blockchain *Blockchain
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

func (app *Blockchain) RunRPC(node *node.Node) {
	app.tendermintRPC = rpc.NewLocal(node)

	status, _ := app.tendermintRPC.Status()

	if status.NodeInfo.Network != genesis.Network {
		log.Error("Different networks")
		os.Exit(1)
	}
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
			app.stateDeliver.SetBalance(account.Address, coin, bigIntValue)
		}
	}

	for _, validator := range req.Validators {
		app.stateDeliver.CreateCandidate(genesisState.FirstValidatorAddress, validator.PubKey.Data, 10, 1, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1000000)))
		app.stateDeliver.CreateValidator(genesisState.FirstValidatorAddress, validator.PubKey.Data, 10, 1, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1000000)))
		app.stateDeliver.SetCandidateOnline(validator.PubKey.Data)
		app.activeValidators = append(app.activeValidators, validator)
	}

	return abciTypes.ResponseInitChain{}
}

func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.height = uint64(req.Header.Height)
	app.rewards = big.NewInt(0)

	// clear absent candidates
	app.validatorsStatuses = map[[20]byte]int8{}

	// give penalty to absent validators
	for _, v := range req.LastCommitInfo.Votes {
		var address [20]byte
		copy(address[:], v.Validator.Address)

		if v.SignedLastBlock {
			app.stateDeliver.SetValidatorPresent(address)
			app.validatorsStatuses[address] = ValidatorPresent
		} else {
			app.stateDeliver.SetValidatorAbsent(address)
			app.validatorsStatuses[address] = ValidatorAbsent
		}
	}

	// give penalty to Byzantine validators
	for _, v := range req.ByzantineValidators {
		var address [20]byte
		copy(address[:], v.Validator.Address)

		app.stateDeliver.PunishByzantineValidator(address)
		app.stateDeliver.RemoveFrozenFundsWithAddress(app.height, app.height+518400, address)
	}

	// apply frozen funds
	frozenFunds := app.stateDeliver.GetStateFrozenFunds(app.height)
	if frozenFunds != nil {
		for _, item := range frozenFunds.List() {
			app.stateDeliver.SetBalance(item.Address, item.Coin, item.Value)
		}

		frozenFunds.Delete()
	}

	return abciTypes.ResponseBeginBlock{}
}

func (app *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {

	var updates []abciTypes.ValidatorUpdate

	stateValidators := app.stateDeliver.GetStateValidators()
	vals := stateValidators.Data()
	// calculate total power of validators
	totalPower := big.NewInt(0)
	for _, val := range vals {
		// skip if candidate is not present
		if app.validatorsStatuses[val.GetAddress()] != ValidatorPresent {
			continue
		}

		totalPower.Add(totalPower, val.TotalBipStake)
	}

	// accumulate rewards
	for i, val := range vals {

		// skip if candidate is not present
		if app.validatorsStatuses[val.GetAddress()] != ValidatorPresent {
			continue
		}

		reward := rewards.GetRewardForBlock(uint64(app.height))

		reward.Add(reward, app.rewards)

		reward.Mul(reward, val.TotalBipStake)
		reward.Div(reward, totalPower)

		vals[i].AccumReward.Add(vals[i].AccumReward, reward)
	}

	stateValidators.SetData(vals)
	app.stateDeliver.SetStateValidators(stateValidators)

	// pay rewards
	if app.height%12 == 0 {
		app.stateDeliver.PayRewards()
	}

	hasDroppedValidators := false
	for _, val := range vals {
		if val.IsToDrop() {
			hasDroppedValidators = true
			break
		}
	}

	// update validators
	if app.height%120 == 0 || hasDroppedValidators {
		app.stateDeliver.RecalculateTotalStakeValues()

		valsCount := validators.GetValidatorsCountForBlock(app.height)

		newCandidates := app.stateDeliver.GetCandidates(valsCount, req.Height)

		if len(newCandidates) < valsCount {
			valsCount = len(newCandidates)
		}

		newValidators := make([]abciTypes.ValidatorUpdate, valsCount)

		// calculate total power
		totalPower := big.NewInt(0)
		for _, candidate := range newCandidates[:valsCount] {
			totalPower.Add(totalPower, candidate.TotalBipStake)
		}

		for i := range newCandidates {
			power := big.NewInt(0).Div(big.NewInt(0).Mul(newCandidates[i].TotalBipStake, big.NewInt(100000000)), totalPower).Int64()

			if power == 0 {
				power = 1
			}

			newValidators[i] = abciTypes.Ed25519ValidatorUpdate(newCandidates[i].PubKey, power)
		}

		// update validators in state
		app.stateDeliver.SetNewValidators(newCandidates)

		// update validators in app cache
		defer func() {
			app.activeValidators = newValidators
		}()

		updates = newValidators

		for _, validator := range app.activeValidators {
			persisted := false
			for _, newValidator := range newValidators {
				if bytes.Equal(validator.PubKey.Data, newValidator.PubKey.Data) {
					persisted = true
					break
				}
			}

			// remove validator
			if !persisted {
				updates = append(updates, abciTypes.ValidatorUpdate{
					PubKey: validator.PubKey,
					Power:  0,
				})
			}
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

func (app *Blockchain) DeliverTx(rawTx []byte) abciTypes.ResponseDeliverTx {
	response := transaction.RunTx(app.stateDeliver, false, rawTx, app.rewards, app.height)

	return abciTypes.ResponseDeliverTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Tags:      response.Tags,
	}
}

func (app *Blockchain) CheckTx(rawTx []byte) abciTypes.ResponseCheckTx {
	response := transaction.RunTx(app.stateCheck, true, rawTx, nil, app.height)

	return abciTypes.ResponseCheckTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Tags:      response.Tags,
	}
}

func (app *Blockchain) Commit() abciTypes.ResponseCommit {

	hash, _ := app.stateDeliver.Commit(false)
	app.stateDeliver.Database().TrieDB().Commit(hash, true)

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
	app.stateDeliver, _ = state.New(app.rootHash, state.NewDatabase(mintdb.NewTable(app.db, stateTableId)))
	app.stateCheck, _ = state.New(app.rootHash, state.NewDatabase(mintdb.NewTable(app.db, stateTableId)))
}

func (app *Blockchain) CurrentState() *state.StateDB {
	return app.stateCheck
}

func (app *Blockchain) GetStateForHeight(height int) (*state.StateDB, error) {
	h := int64(height)
	result, err := app.tendermintRPC.Block(&h)

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

func GetBlockchain() *Blockchain {
	return blockchain
}
