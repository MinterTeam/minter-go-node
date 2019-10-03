package minter

import (
	"bytes"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/appdb"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/eventsdb/events"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/version"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	tmNode "github.com/tendermint/tendermint/node"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
	types2 "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tm-db"
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
	stateDeliver        *state.State
	stateCheck          *state.State
	height              uint64 // current Blockchain height
	lastCommittedHeight uint64 // Blockchain.height updated in the at begin of block processing, while
	// lastCommittedHeight updated at the end of block processing
	rewards            *big.Int // Rewards pool
	validatorsStatuses map[[20]byte]int8

	// local rpc client for Tendermint
	tmNode *tmNode.Node

	// currentMempool is responsive for prevent sending multiple transactions from one address in one block
	currentMempool sync.Map

	lock    sync.RWMutex
	wg      sync.WaitGroup // wg is used for graceful node shutdown
	stopped uint32
}

// Creates Minter Blockchain instance, should be only called once
func NewMinterBlockchain(cfg *config.Config) *Blockchain {
	dbType := db.DBBackendType(cfg.DBBackend)
	ldb := db.NewDB("state", dbType, utils.GetMinterHome()+"/data")

	// Initiate Application DB. Used for persisting data like current block, validators, etc.
	applicationDB := appdb.NewAppDB(cfg)

	blockchain = &Blockchain{
		stateDB:             ldb,
		appDB:               applicationDB,
		height:              applicationDB.GetLastHeight(),
		lastCommittedHeight: applicationDB.GetLastHeight(),
		currentMempool:      sync.Map{},
	}

	// Set stateDeliver and stateCheck
	var err error
	blockchain.stateDeliver, err = state.NewState(blockchain.height, blockchain.stateDB)
	if err != nil {
		panic(err)
	}

	blockchain.stateCheck = state.NewCheckState(blockchain.stateDeliver)

	// Set start height for rewards and validators
	rewards.SetStartHeight(applicationDB.GetStartHeight())
	validators.SetStartHeight(applicationDB.GetStartHeight())

	return blockchain
}

// Initialize blockchain with validators and other info. Only called once.
func (app *Blockchain) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {
	var genesisState types.AppState
	if err := amino.UnmarshalJSON(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	if err := app.stateDeliver.Import(genesisState); err != nil {
		panic(err)
	}

	totalPower := big.NewInt(0)
	for _, val := range genesisState.Validators {
		totalPower.Add(totalPower, val.TotalBipStake)
	}

	vals := make([]abciTypes.ValidatorUpdate, len(genesisState.Validators))
	for i, val := range genesisState.Validators {
		var validatorPubKey ed25519.PubKeyEd25519
		copy(validatorPubKey[:], val.PubKey[:])
		pkey, err := cryptoAmino.PubKeyFromBytes(validatorPubKey.Bytes())
		if err != nil {
			panic(err)
		}

		vals[i] = abciTypes.ValidatorUpdate{
			PubKey: types2.TM2PB.PubKey(pkey),
			Power: big.NewInt(0).Div(big.NewInt(0).Mul(val.TotalBipStake,
				big.NewInt(100000000)), totalPower).Int64(),
		}
	}

	app.appDB.SetStartHeight(genesisState.StartHeight)
	app.appDB.SaveValidators(vals)
	rewards.SetStartHeight(genesisState.StartHeight)
	validators.SetStartHeight(genesisState.StartHeight)

	return abciTypes.ResponseInitChain{
		Validators: vals,
	}
}

// Signals the beginning of a block.
func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.wg.Add(1)
	if atomic.LoadUint32(&app.stopped) == 1 {
		panic("Application stopped")
	}

	height := uint64(req.Header.Height)
	// Check invariants
	if height%720 == 0 {
		if err := state.NewCheckState(app.stateCheck).CheckForInvariants(); err != nil {
			log.With("module", "invariants").Error("Invariants error", "msg", err.Error(), "height", app.height)
		}
	}

	// compute max gas
	app.updateBlocksTimeDelta(height, 3)
	maxGas := app.calcMaxGas(height)
	app.stateDeliver.App.SetMaxGas(maxGas)

	atomic.StoreUint64(&app.height, height)
	app.rewards = big.NewInt(0)

	// clear absent candidates
	app.validatorsStatuses = map[[20]byte]int8{}

	// give penalty to absent validators
	for _, v := range req.LastCommitInfo.Votes {
		var address [20]byte
		copy(address[:], v.Validator.Address)

		if v.SignedLastBlock {
			app.stateDeliver.Validators.SetValidatorPresent(address)
			app.validatorsStatuses[address] = ValidatorPresent
		} else {
			app.stateDeliver.Validators.SetValidatorAbsent(address)
			app.validatorsStatuses[address] = ValidatorAbsent
		}
	}

	// give penalty to Byzantine validators
	for _, byzVal := range req.ByzantineValidators {
		var address types.TmAddress
		copy(address[:], byzVal.Validator.Address)

		// skip already offline candidates to prevent double punishing
		candidate := app.stateDeliver.Candidates.GetCandidateByTendermintAddress(address)
		if candidate == nil || candidate.Status == candidates.CandidateStatusOffline {
			continue
		}

		app.stateDeliver.FrozenFunds.PunishFrozenFundsWithAddress(height, height+candidates.UnbondPeriod, address)
		app.stateDeliver.Validators.PunishByzantineValidator(address)
		app.stateDeliver.Candidates.PunishByzantineCandidate(address)
	}

	// apply frozen funds (used for unbond stakes)
	frozenFunds := app.stateDeliver.FrozenFunds.GetFrozenFunds(uint64(req.Header.Height))
	if frozenFunds != nil {
		for _, item := range frozenFunds.List {
			eventsdb.GetCurrent().AddEvent(uint64(req.Header.Height), events.UnbondEvent{
				Address:         item.Address,
				Amount:          item.Value.Bytes(),
				Coin:            item.Coin,
				ValidatorPubKey: *item.CandidateKey,
			})
			app.stateDeliver.Accounts.AddBalance(item.Address, item.Coin, item.Value)
		}

		// delete from db
		frozenFunds.Delete()
	}

	return abciTypes.ResponseBeginBlock{}
}

// Signals the end of a block, returns changes to the validator set
func (app *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	height := uint64(req.Height)

	var updates []abciTypes.ValidatorUpdate

	vals := app.stateDeliver.Validators.GetValidators()

	hasDroppedValidators := false
	for _, val := range vals {
		if val.IsToDrop() {
			hasDroppedValidators = true

			// Move dropped validator's accum rewards back to pool
			app.rewards.Add(app.rewards, val.GetAccumReward())
			val.SetAccumReward(big.NewInt(0))
			break
		}
	}

	// calculate total power of validators
	totalPower := big.NewInt(0)
	for _, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || app.validatorsStatuses[val.GetAddress()] != ValidatorPresent {
			continue
		}

		totalPower.Add(totalPower, val.GetTotalBipStake())
	}

	if totalPower.Cmp(types.Big0) == 0 {
		totalPower = big.NewInt(1)
	}

	// accumulate rewards
	reward := rewards.GetRewardForBlock(height)
	reward.Add(reward, app.rewards)

	// compute remainder to keep total emission consist
	remainder := big.NewInt(0).Set(reward)

	for i, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || app.validatorsStatuses[val.GetAddress()] != ValidatorPresent {
			continue
		}

		r := big.NewInt(0).Set(reward)
		r.Mul(r, val.GetTotalBipStake())
		r.Div(r, totalPower)

		remainder.Sub(remainder, r)
		vals[i].AddAccumReward(r)
	}

	// add remainder to total slashed
	app.stateDeliver.App.AddTotalSlashed(remainder)

	// pay rewards
	if req.Height%12 == 0 {
		app.stateDeliver.Validators.PayRewards()
	}

	// update validators
	if req.Height%120 == 0 || hasDroppedValidators {
		app.stateDeliver.Candidates.RecalculateTotalStakeValues()

		valsCount := validators.GetValidatorsCountForBlock(height)

		newCandidates := app.stateDeliver.Candidates.GetNewCandidates(valsCount, req.Height)

		// remove candidates with 0 total stake
		var tempCandidates []candidates.Candidate
		for _, candidate := range newCandidates {
			if app.stateDeliver.Candidates.GetTotalStake(candidate.PubKey).Cmp(big.NewInt(0)) != 1 {
				continue
			}

			tempCandidates = append(tempCandidates, candidate)
		}
		newCandidates = tempCandidates

		if len(newCandidates) < valsCount {
			valsCount = len(newCandidates)
		}

		newValidators := make([]abciTypes.ValidatorUpdate, valsCount)

		// calculate total power
		totalPower := big.NewInt(0)
		for _, candidate := range newCandidates {
			totalPower.Add(totalPower, app.stateDeliver.Candidates.GetTotalStake(candidate.PubKey))
		}

		for i := range newCandidates {
			power := big.NewInt(0).Div(big.NewInt(0).Mul(app.stateDeliver.Candidates.GetTotalStake(newCandidates[i].PubKey),
				big.NewInt(100000000)), totalPower).Int64()

			if power == 0 {
				power = 1
			}

			newValidators[i] = abciTypes.Ed25519ValidatorUpdate(newCandidates[i].PubKey[:], power)
		}

		// update validators in state
		app.stateDeliver.Validators.SetNewValidators(newCandidates)

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

	return abciTypes.ResponseEndBlock{
		ValidatorUpdates: updates,
		ConsensusParamUpdates: &abciTypes.ConsensusParams{
			Block: &abciTypes.BlockParams{
				MaxBytes: BlockMaxBytes,
				MaxGas:   int64(app.stateDeliver.App.GetMaxGas()),
			},
		},
	}
}

// Return application info. Used for synchronization between Tendermint and Minter
func (app *Blockchain) Info(req abciTypes.RequestInfo) (resInfo abciTypes.ResponseInfo) {
	return abciTypes.ResponseInfo{
		Version:          version.Version,
		AppVersion:       version.AppVer,
		LastBlockHeight:  int64(app.appDB.GetLastHeight()),
		LastBlockAppHash: app.appDB.GetLastBlockHash(),
	}
}

// Deliver a tx for full processing
func (app *Blockchain) DeliverTx(req abciTypes.RequestDeliverTx) abciTypes.ResponseDeliverTx {
	response := transaction.RunTx(app.stateDeliver, false, req.Tx, app.rewards, app.height, sync.Map{}, 0)

	return abciTypes.ResponseDeliverTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Events: []abciTypes.Event{
			{
				Type:       "tags",
				Attributes: response.Tags,
			},
		},
	}
}

// Validate a tx for the mempool
func (app *Blockchain) CheckTx(req abciTypes.RequestCheckTx) abciTypes.ResponseCheckTx {
	response := transaction.RunTx(app.stateCheck, true, req.Tx, nil, app.height, app.currentMempool, app.MinGasPrice())

	return abciTypes.ResponseCheckTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Events: []abciTypes.Event{
			{
				Type:       "tags",
				Attributes: response.Tags,
			},
		},
	}
}

// Commit the state and return the application Merkle root hash
func (app *Blockchain) Commit() abciTypes.ResponseCommit {
	// Committing Minter Blockchain state
	hash, err := app.stateDeliver.Commit()
	if err != nil {
		panic(err)
	}

	// Flush events db
	_ = eventsdb.GetCurrent().FlushEvents()

	// Persist application hash and height
	app.appDB.SetLastBlockHash(hash)
	app.appDB.SetLastHeight(app.height)

	// Resetting check state to be consistent with current height
	app.resetCheckState()

	// Update LastCommittedHeight
	atomic.StoreUint64(&app.lastCommittedHeight, app.Height())

	// Clear mempool
	app.currentMempool = sync.Map{}

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
func (app *Blockchain) CurrentState() *state.State {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return state.NewCheckState(app.stateCheck)
}

// Get immutable state of Minter Blockchain for given height
func (app *Blockchain) GetStateForHeight(height uint64) (*state.State, error) {
	app.lock.RLock()
	defer app.lock.RUnlock()

	s, err := state.NewCheckStateAtHeight(height, app.stateDB)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "State at given height not found", Data: err.Error()}
	}

	return s, nil
}

// Get current height of Minter Blockchain
func (app *Blockchain) Height() uint64 {
	return atomic.LoadUint64(&app.height)
}

// Get last committed height of Minter Blockchain
func (app *Blockchain) LastCommittedHeight() uint64 {
	return atomic.LoadUint64(&app.lastCommittedHeight)
}

// Set Tendermint node
func (app *Blockchain) SetTmNode(node *tmNode.Node) {
	app.tmNode = node
}

// Get minimal acceptable gas price
func (app *Blockchain) MinGasPrice() uint32 {
	mempoolSize := app.tmNode.Mempool().Size()

	if mempoolSize > 5000 {
		return 50
	}

	if mempoolSize > 1000 {
		return 10
	}

	if mempoolSize > 500 {
		return 5
	}

	if mempoolSize > 100 {
		return 2
	}

	return 1
}

func (app *Blockchain) resetCheckState() {
	app.lock.Lock()
	defer app.lock.Unlock()

	app.stateCheck = state.NewCheckState(app.stateDeliver)
}

func (app *Blockchain) getCurrentValidators() abciTypes.ValidatorUpdates {
	return app.appDB.GetValidators()
}

func (app *Blockchain) saveCurrentValidators(vals abciTypes.ValidatorUpdates) {
	app.appDB.SaveValidators(vals)
}

func (app *Blockchain) updateBlocksTimeDelta(height uint64, count int64) {
	// should do this because tmNode is unavailable during Tendermint's replay mode
	if app.tmNode == nil {
		return
	}

	if int64(height)-count-1 < 1 {
		return
	}

	blockStore := app.tmNode.BlockStore()

	blockA := blockStore.LoadBlockMeta(int64(height) - count - 1)
	blockB := blockStore.LoadBlockMeta(int64(height) - 1)

	delta := int(blockB.Header.Time.Sub(blockA.Header.Time).Seconds())
	app.appDB.SetLastBlocksTimeDelta(height, delta)
}

func (app *Blockchain) SetBlocksTimeDelta(height uint64, value int) {
	app.appDB.SetLastBlocksTimeDelta(height, value)
}

func (app *Blockchain) GetBlocksTimeDelta(height, count uint64) (int, error) {
	return app.appDB.GetLastBlocksTimeDelta(height)
}

func (app *Blockchain) calcMaxGas(height uint64) uint64 {
	const targetTime = 7
	const blockDelta = 3

	// skip first 20 blocks
	if height <= 20 {
		return DefaultMaxGas
	}

	// get current max gas
	newMaxGas := app.stateCheck.App.GetMaxGas()

	// check if blocks are created in time
	if delta, _ := app.GetBlocksTimeDelta(height, blockDelta); delta > targetTime*blockDelta {
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
