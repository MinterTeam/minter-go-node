package minter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/appdb"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/genesis"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/danil-lashin/tendermint/rpc/lib/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	tmNode "github.com/tendermint/tendermint/node"
	types2 "github.com/tendermint/tendermint/types"
	"math/big"
	"sync"
	"sync/atomic"
)

const (
	// Global validator's statuses
	ValidatorPresent = 1
	ValidatorAbsent  = 2

	BlockMaxBytes = 10000000

	DefaultMaxGas = 100000
	MinMaxGas     = 5000
)

var (
	blockchain *Blockchain
)

// Main structure of Minter Blockchain
type Blockchain struct {
	abciTypes.BaseApplication

	stateDB             db.DB
	appDB               *appdb.AppDB
	stateDeliver        *state.StateDB
	stateCheck          *state.StateDB
	height              int64    // current Blockchain height
	lastCommittedHeight int64    // Blockchain.height updated in the at begin of block processing, while lastCommittedHeight updated at the end of block processing
	rewards             *big.Int // Rewards pool
	validatorsStatuses  map[[20]byte]int8

	// local rpc client for Tendermint
	tmNode *tmNode.Node

	// currentMempool is responsive for prevent sending multiple transactions from one address in one block
	currentMempool map[types.Address]struct{}

	lock    sync.RWMutex
	wg      sync.WaitGroup // wg is used for graceful node shutdown
	stopped uint32
}

// Creates Minter Blockchain instance, should be only called once
func NewMinterBlockchain() *Blockchain {
	ldb, err := db.NewGoLevelDB("state", utils.GetMinterHome()+"/data")
	if err != nil {
		panic(err)
	}

	// Initiate Application DB. Used for persisting data like current block, validators, etc.
	applicationDB := appdb.NewAppDB()

	blockchain = &Blockchain{
		stateDB:             ldb,
		appDB:               applicationDB,
		height:              applicationDB.GetLastHeight(),
		lastCommittedHeight: applicationDB.GetLastHeight(),
		currentMempool:      map[types.Address]struct{}{},
	}

	// Set stateDeliver and stateCheck
	blockchain.stateDeliver, err = state.New(int64(blockchain.height), blockchain.stateDB)
	if err != nil {
		panic(err)
	}

	blockchain.stateCheck = state.NewForCheck(blockchain.stateDeliver)

	return blockchain
}

// Initialize blockchain with validators and other info. Only called once.
func (app *Blockchain) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {
	var genesisState genesis.AppState
	err := json.Unmarshal(req.AppStateBytes, &genesisState)
	if err != nil {
		panic(err)
	}

	// Filling genesis accounts with given amount of coins
	for _, account := range genesisState.InitialBalances {
		for coin, value := range account.Balance {
			bigIntValue, success := big.NewInt(0).SetString(value, 10)
			if !success {
				panic(fmt.Sprintf("%s is not a corrent int", value))
			}

			coinSymbol := types.StrToCoinSymbol(coin)
			app.stateDeliver.SetBalance(account.Address, coinSymbol, bigIntValue)
		}
	}

	// Set initial Blockchain validators
	commission := uint(100)
	currentBlock := uint(1)
	initialStake := helpers.BipToPip(big.NewInt(1000000)) // 1 mln bip
	for _, validator := range req.Validators {
		app.stateDeliver.CreateCandidate(genesisState.FirstValidatorAddress, genesisState.FirstValidatorAddress, validator.PubKey.Data, commission, currentBlock, types.GetBaseCoin(), initialStake)
		app.stateDeliver.CreateValidator(genesisState.FirstValidatorAddress, validator.PubKey.Data, commission, currentBlock, types.GetBaseCoin(), initialStake)
		app.stateDeliver.SetCandidateOnline(validator.PubKey.Data)
	}

	app.stateDeliver.SetMaxGas(DefaultMaxGas)

	return abciTypes.ResponseInitChain{
		ConsensusParams: &abciTypes.ConsensusParams{
			BlockSize: &abciTypes.BlockSizeParams{
				MaxBytes: BlockMaxBytes,
				MaxGas:   DefaultMaxGas,
			},
			Evidence: &abciTypes.EvidenceParams{
				MaxAge: 1000,
			},
			Validator: &abciTypes.ValidatorParams{
				PubKeyTypes: []string{types2.ABCIPubKeyTypeEd25519},
			},
		},
	}
}

// Signals the beginning of a block.
func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.wg.Add(1)
	if atomic.LoadUint32(&app.stopped) == 1 {
		panic("Application stopped")
	}

	app.updateBlocksTimeDelta(req.Header.Height, 3)

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

	// apply frozen funds (used for unbond stakes)
	frozenFunds := app.stateDeliver.GetStateFrozenFunds(uint64(req.Header.Height))
	if frozenFunds != nil {
		for _, item := range frozenFunds.List() {
			eventsdb.GetCurrent().AddEvent(req.Header.Height, eventsdb.UnbondEvent{
				Address:         item.Address,
				Amount:          item.Value.Bytes(),
				Coin:            item.Coin,
				ValidatorPubKey: item.CandidateKey,
			})
			app.stateDeliver.AddBalance(item.Address, item.Coin, item.Value)
		}

		// delete from db
		frozenFunds.Delete()
	}

	return abciTypes.ResponseBeginBlock{}
}

// Signals the end of a block, returns changes to the validator set
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

	// compute max gas
	maxGas := app.calcMaxGas(req.Height)
	app.stateDeliver.SetMaxGas(maxGas)

	return abciTypes.ResponseEndBlock{
		ValidatorUpdates: updates,
		ConsensusParamUpdates: &abciTypes.ConsensusParams{
			BlockSize: &abciTypes.BlockSizeParams{
				MaxBytes: BlockMaxBytes,
				MaxGas:   int64(maxGas),
			},
		},
	}
}

// Return application info. Used for synchronization between Tendermint and Minter
func (app *Blockchain) Info(req abciTypes.RequestInfo) (resInfo abciTypes.ResponseInfo) {
	return abciTypes.ResponseInfo{
		Version:          version.Version,
		AppVersion:       version.AppVer,
		LastBlockHeight:  app.appDB.GetLastHeight(),
		LastBlockAppHash: app.appDB.GetLastBlockHash(),
	}
}

// Deliver a tx for full processing
func (app *Blockchain) DeliverTx(rawTx []byte) abciTypes.ResponseDeliverTx {
	response := transaction.RunTx(app.stateDeliver, false, rawTx, app.rewards, app.height, nil)

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

// Validate a tx for the mempool
func (app *Blockchain) CheckTx(rawTx []byte) abciTypes.ResponseCheckTx {
	response := transaction.RunTx(app.stateCheck, true, rawTx, nil, app.height, app.currentMempool)

	if response.Code == code.OK && response.GasPrice.Cmp(app.MinGasPrice()) == -1 {
		return abciTypes.ResponseCheckTx{
			Code: code.TooLowGasPrice,
			Log:  fmt.Sprintf("Gas price of tx is too low to be included in mempool. Expected %s", app.MinGasPrice().String()),
		}
	}

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

// Commit the state and return the application Merkle root hash
func (app *Blockchain) Commit() abciTypes.ResponseCommit {
	// Committing Minter Blockchain state
	hash, _, err := app.stateDeliver.Commit()
	if err != nil {
		panic(err)
	}

	// Persist application hash and height
	app.appDB.SetLastBlockHash(hash)
	app.appDB.SetLastHeight(app.height)

	// Resetting check state to be consistent with current height
	app.resetCheckState()

	// Update LastCommittedHeight
	atomic.StoreInt64(&app.lastCommittedHeight, app.Height())

	// Clear mempool
	app.currentMempool = map[types.Address]struct{}{}

	// Releasing wg
	app.wg.Done()

	return abciTypes.ResponseCommit{
		Data: hash,
	}
}

// Unused method, required by Tendermint
func (app *Blockchain) Query(reqQuery abciTypes.RequestQuery) abciTypes.ResponseQuery {
	return abciTypes.ResponseQuery{}
}

// Unused method, required by Tendermint
func (app *Blockchain) SetOption(req abciTypes.RequestSetOption) abciTypes.ResponseSetOption {
	return abciTypes.ResponseSetOption{}
}

// Gracefully stopping Minter Blockchain instance
func (app *Blockchain) Stop() {
	atomic.StoreUint32(&app.stopped, 1)
	app.wg.Wait()

	app.appDB.Close()
	app.stateDB.Close()
}

// Get immutable state of Minter Blockchain
func (app *Blockchain) CurrentState() *state.StateDB {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return state.NewForCheck(app.stateCheck)
}

// Get immutable state of Minter Blockchain for given height
func (app *Blockchain) GetStateForHeight(height int) (*state.StateDB, error) {
	app.lock.RLock()
	defer app.lock.RUnlock()

	s, err := state.New(int64(height), app.stateDB)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "State at given height not found", Data: err.Error()}
	}

	return s, nil
}

// Get current height of Minter Blockchain
func (app *Blockchain) Height() int64 {
	return atomic.LoadInt64(&app.height)
}

// Get last committed height of Minter Blockchain
func (app *Blockchain) LastCommittedHeight() int64 {
	return atomic.LoadInt64(&app.lastCommittedHeight)
}

// Set Tendermint node
func (app *Blockchain) SetTmNode(node *tmNode.Node) {
	app.tmNode = node
}

// Get minimal acceptable gas price
func (app *Blockchain) MinGasPrice() *big.Int {
	mempoolSize := app.tmNode.MempoolReactor().Mempool.Size()

	if mempoolSize > 5000 {
		return big.NewInt(50)
	}

	if mempoolSize > 1000 {
		return big.NewInt(10)
	}

	if mempoolSize > 500 {
		return big.NewInt(5)
	}

	if mempoolSize > 100 {
		return big.NewInt(2)
	}

	return big.NewInt(1)
}

func (app *Blockchain) resetCheckState() {
	app.lock.Lock()
	defer app.lock.Unlock()

	app.stateCheck = state.NewForCheck(app.stateDeliver)
}

func (app *Blockchain) getCurrentValidators() abciTypes.ValidatorUpdates {
	return app.appDB.GetValidators()
}

func (app *Blockchain) saveCurrentValidators(vals abciTypes.ValidatorUpdates) {
	app.appDB.SaveValidators(vals)
}

func (app *Blockchain) updateBlocksTimeDelta(height, count int64) {
	// should do this because tmNode is unavailable during Tendermint's replay mode
	if app.tmNode == nil {
		return
	}

	if height-count-1 < 1 {
		return
	}

	blockStore := app.tmNode.BlockStore()

	blockA := blockStore.LoadBlockMeta(height - count - 1)
	blockB := blockStore.LoadBlockMeta(height - 1)

	delta := int(blockB.Header.Time.Sub(blockA.Header.Time).Seconds())
	app.appDB.SetLastBlocksTimeDelta(height, delta)
}

func (app *Blockchain) getBlocksTimeDelta(height, count int64) int {
	return app.appDB.GetLastBlocksTimeDelta(height)
}

func (app *Blockchain) calcMaxGas(height int64) uint64 {
	const targetTime = 7
	const blockDelta = 3

	// skip first 20 blocks
	if height <= 20 {
		return DefaultMaxGas
	}

	// get current max gas
	newMaxGas := app.stateCheck.GetCurrentMaxGas()

	// check if blocks are created in time
	if app.getBlocksTimeDelta(height, blockDelta) > targetTime*blockDelta {
		newMaxGas = newMaxGas * 7 / 10 // decrease by 30%
	} else {
		newMaxGas = newMaxGas * 105 / 100 // increase by 5%
	}

	// check if max gas is too high
	if newMaxGas > DefaultMaxGas {
		newMaxGas = DefaultMaxGas
	}

	// check if max gas is too low
	if newMaxGas < MinMaxGas {
		newMaxGas = MinMaxGas
	}

	return newMaxGas
}
