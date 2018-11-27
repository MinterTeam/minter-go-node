package minter

import (
	"bytes"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/appdb"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/genesis"
	"github.com/MinterTeam/minter-go-node/helpers"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	"math/big"
	"sync"
	"sync/atomic"
)

type Blockchain struct {
	abciTypes.BaseApplication

	stateDB             db.DB
	appDB               *appdb.AppDB
	stateDeliver        *state.StateDB
	stateCheck          *state.StateDB
	height              int64
	lastCommittedHeight int64
	rewards             *big.Int
	validatorsStatuses  map[[20]byte]int8

	lock sync.RWMutex
	wg   sync.WaitGroup
}

const (
	ValidatorPresent = 1
	ValidatorAbsent  = 2
)

var (
	blockchain *Blockchain
)

func NewMinterBlockchain() *Blockchain {
	ldb, err := db.NewGoLevelDB("state", utils.GetMinterHome()+"/data")

	if err != nil {
		panic(err)
	}

	applicationDB := appdb.NewAppDB()

	blockchain = &Blockchain{
		stateDB:             ldb,
		appDB:               applicationDB,
		height:              applicationDB.GetLastHeight(),
		lastCommittedHeight: applicationDB.GetLastHeight(),
	}

	blockchain.stateDeliver, err = state.New(int64(blockchain.height), blockchain.stateDB)

	if err != nil {
		panic(err)
	}

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
			app.stateDeliver.SetBalance(account.Address, coin, bigIntValue)
		}
	}

	for _, validator := range req.Validators {
		app.stateDeliver.CreateCandidate(genesisState.FirstValidatorAddress, validator.PubKey.Data, 100, 1, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1000000)))
		app.stateDeliver.CreateValidator(genesisState.FirstValidatorAddress, validator.PubKey.Data, 100, 1, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1000000)))
		app.stateDeliver.SetCandidateOnline(validator.PubKey.Data)
	}

	return abciTypes.ResponseInitChain{}
}

func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.wg.Add(1)

	atomic.StoreInt64(&app.height, req.Header.Height)
	app.rewards = big.NewInt(0)

	// clear absent candidates
	app.validatorsStatuses = map[[20]byte]int8{}

	// give penalty to absent validators
	for _, v := range req.LastCommitInfo.Votes {
		var address [20]byte
		copy(address[:], v.Validator.Address)

		if v.SignedLastBlock {
			app.stateDeliver.SetValidatorPresent(req.Header.Height, address)
			app.validatorsStatuses[address] = ValidatorPresent
		} else {
			app.stateDeliver.SetValidatorAbsent(req.Header.Height, address)
			app.validatorsStatuses[address] = ValidatorAbsent
		}
	}

	// give penalty to Byzantine validators
	for i := range req.ByzantineValidators {
		v := &req.ByzantineValidators[i]
		var address [20]byte
		copy(address[:], v.Validator.Address)

		app.stateDeliver.PunishByzantineValidator(req.Header.Height, address)
		app.stateDeliver.PunishFrozenFundsWithAddress(uint64(req.Header.Height), uint64(req.Header.Height+518400), address)
	}

	// apply frozen funds
	frozenFunds := app.stateDeliver.GetStateFrozenFunds(uint64(req.Header.Height))
	if frozenFunds != nil {
		for _, item := range frozenFunds.List() {
			app.stateDeliver.AddBalance(item.Address, item.Coin, item.Value)
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

		reward := rewards.GetRewardForBlock(uint64(req.Height))

		reward.Add(reward, app.rewards)

		reward.Mul(reward, val.TotalBipStake)
		reward.Div(reward, totalPower)

		vals[i].AccumReward.Add(vals[i].AccumReward, reward)
	}

	stateValidators.SetData(vals)
	app.stateDeliver.SetStateValidators(stateValidators)

	// pay rewards
	if req.Height%12 == 0 {
		app.stateDeliver.PayRewards(req.Height)
	}

	hasDroppedValidators := false
	for _, val := range vals {
		if val.IsToDrop() {
			hasDroppedValidators = true
			break
		}
	}

	// update validators
	if req.Height%120 == 0 || hasDroppedValidators {
		app.stateDeliver.RecalculateTotalStakeValues()

		app.stateDeliver.ClearCandidates(req.Height)
		app.stateDeliver.ClearStakes(req.Height)

		valsCount := validators.GetValidatorsCountForBlock(req.Height)

		newCandidates := app.stateDeliver.GetCandidates(valsCount, req.Height)

		if len(newCandidates) < valsCount {
			valsCount = len(newCandidates)
		}

		newValidators := make([]abciTypes.ValidatorUpdate, valsCount)

		// calculate total power
		totalPower := big.NewInt(0)
		for _, candidate := range newCandidates {
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

		activeValidators := app.getCurrentValidators()

		app.saveCurrentValidators(newValidators)

		updates = newValidators

		for _, validator := range activeValidators {
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

	_ = eventsdb.GetCurrent().FlushEvents(req.Height)

	return abciTypes.ResponseEndBlock{
		ValidatorUpdates: updates,
	}
}

func (app *Blockchain) Info(req abciTypes.RequestInfo) (resInfo abciTypes.ResponseInfo) {
	return abciTypes.ResponseInfo{
		LastBlockHeight:  app.appDB.GetLastHeight(),
		LastBlockAppHash: app.appDB.GetLastBlockHash(),
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
	hash, _, err := app.stateDeliver.Commit(false)

	if err != nil {
		panic(err)
	}

	app.appDB.SetLastBlockHash(hash)
	app.appDB.SetLastHeight(app.height)

	app.updateCurrentState()

	atomic.StoreInt64(&app.lastCommittedHeight, app.Height())

	app.wg.Done()

	return abciTypes.ResponseCommit{
		Data: hash,
	}
}

func (app *Blockchain) Query(reqQuery abciTypes.RequestQuery) abciTypes.ResponseQuery {
	return abciTypes.ResponseQuery{}
}

func (app *Blockchain) Stop() {
	app.wg.Wait()

	app.appDB.Close()
	app.stateDB.Close()
}

func (app *Blockchain) updateCurrentState() {
	app.lock.Lock()
	defer app.lock.Unlock()

	app.stateCheck = state.NewForCheck(app.stateDeliver)
}

func (app *Blockchain) CurrentState() *state.StateDB {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return state.NewForCheck(app.stateCheck)
}

func (app *Blockchain) GetStateForHeight(height int) (*state.StateDB, error) {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return state.New(int64(height), app.stateDB)
}

func (app *Blockchain) Height() int64 {
	return atomic.LoadInt64(&app.height)
}

func (app *Blockchain) LastCommittedHeight() int64 {
	return atomic.LoadInt64(&app.lastCommittedHeight)
}

func (app *Blockchain) getCurrentValidators() abciTypes.ValidatorUpdates {
	return app.appDB.GetValidators()
}

func (app *Blockchain) saveCurrentValidators(vals abciTypes.ValidatorUpdates) {
	app.appDB.SaveValidators(vals)
}
