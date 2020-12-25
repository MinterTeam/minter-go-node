package minter

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/appdb"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/statistics"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tendermint/go-amino"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tm-db"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Statuses of validators
const (
	ValidatorPresent = 1
	ValidatorAbsent  = 2
)

// Block params
const (
	blockMaxBytes = 10000000
	defaultMaxGas = 100000
	minMaxGas     = 5000
)

const votingPowerConsensus = 2. / 3.

var (
	blockchain *Blockchain
)

// Blockchain is a main structure of Minter
type Blockchain struct {
	abciTypes.BaseApplication

	statisticData *statistics.Data

	stateDB            db.DB
	appDB              *appdb.AppDB
	eventsDB           eventsdb.IEventsDB
	stateDeliver       *state.State
	stateCheck         *state.CheckState
	height             uint64   // current Blockchain height
	rewards            *big.Int // Rewards pool
	validatorsStatuses map[types.TmAddress]int8

	// local rpc client for Tendermint
	tmNode *tmNode.Node

	// currentMempool is responsive for prevent sending multiple transactions from one address in one block
	currentMempool *sync.Map

	lock sync.RWMutex

	haltHeight uint64
	cfg        *config.Config
}

// NewMinterBlockchain creates Minter Blockchain instance, should be only called once
func NewMinterBlockchain(cfg *config.Config) *Blockchain {
	var err error

	ldb, err := db.NewGoLevelDBWithOpts("state", utils.GetMinterHome()+"/data", getDbOpts(cfg.StateMemAvailable))
	if err != nil {
		panic(err)
	}

	// Initiate Application DB. Used for persisting data like current block, validators, etc.
	applicationDB := appdb.NewAppDB(cfg)

	edb, err := db.NewGoLevelDBWithOpts("events", utils.GetMinterHome()+"/data", getDbOpts(1024))
	if err != nil {
		panic(err)
	}

	lastHeight := applicationDB.GetLastHeight()
	initialHeight := applicationDB.GetStartHeight()
	height := lastHeight
	if lastHeight < initialHeight {
		height = initialHeight
	}
	blockchain = &Blockchain{
		stateDB:        ldb,
		appDB:          applicationDB,
		height:         height,
		eventsDB:       eventsdb.NewEventsStore(edb),
		currentMempool: &sync.Map{},
		cfg:            cfg,
	}

	// Set stateDeliver and stateCheck
	blockchain.stateDeliver, err = state.NewState(lastHeight, blockchain.stateDB, blockchain.eventsDB, cfg.StateCacheSize, cfg.KeepLastStates, initialHeight)
	if err != nil {
		panic(err)
	}

	blockchain.stateCheck = state.NewCheckState(blockchain.stateDeliver)

	// Set start height for rewards and validators
	rewards.SetStartHeight(initialHeight)

	blockchain.haltHeight = uint64(cfg.HaltHeight)

	return blockchain
}

// InitChain initialize blockchain with validators and other info. Only called once.
func (app *Blockchain) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {
	var genesisState types.AppState
	if err := amino.UnmarshalJSON(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	if genesisState.StartHeight > app.height {
		app.appDB.SetStartHeight(genesisState.StartHeight)
		app.Stop()
		*app = *NewMinterBlockchain(app.cfg)
	}
	if err := app.stateDeliver.Import(genesisState); err != nil {
		panic(err)
	}

	vals := app.updateValidators()

	app.appDB.SetStartHeight(genesisState.StartHeight)
	app.appDB.FlushValidators()

	rewards.SetStartHeight(genesisState.StartHeight)

	return abciTypes.ResponseInitChain{
		Validators: vals,
	}
}

// BeginBlock signals the beginning of a block.
func (app *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	height := uint64(req.Header.Height)

	app.StatisticData().PushStartBlock(&statistics.StartRequest{Height: int64(height), Now: time.Now(), HeaderTime: req.Header.Time})

	app.stateDeliver.Lock()

	// compute max gas
	app.updateBlocksTimeDelta(height, 3)
	maxGas := app.calcMaxGas(height)
	app.stateDeliver.App.SetMaxGas(maxGas)

	atomic.StoreUint64(&app.height, height)
	app.rewards = big.NewInt(0)

	// clear absent candidates
	app.lock.Lock()
	app.validatorsStatuses = map[types.TmAddress]int8{}

	// give penalty to absent validators
	for _, v := range req.LastCommitInfo.Votes {
		var address types.TmAddress
		copy(address[:], v.Validator.Address)

		if v.SignedLastBlock {
			app.stateDeliver.Validators.SetValidatorPresent(height, address)
			app.validatorsStatuses[address] = ValidatorPresent
		} else {
			app.stateDeliver.Validators.SetValidatorAbsent(height, address)
			app.validatorsStatuses[address] = ValidatorAbsent
		}
	}
	app.lock.Unlock()

	if app.isApplicationHalted(height) {
		panic(fmt.Sprintf("Application halted at height %d", height))
	}

	// give penalty to Byzantine validators
	for _, byzVal := range req.ByzantineValidators {
		var address types.TmAddress
		copy(address[:], byzVal.Validator.Address)

		// skip already offline candidates to prevent double punishing
		candidate := app.stateDeliver.Candidates.GetCandidateByTendermintAddress(address)
		if candidate == nil || candidate.Status == candidates.CandidateStatusOffline || app.stateDeliver.Validators.GetByTmAddress(address) == nil {
			continue
		}

		app.stateDeliver.FrozenFunds.PunishFrozenFundsWithID(height, height+candidates.UnbondPeriod, candidate.ID)
		app.stateDeliver.Validators.PunishByzantineValidator(address)
		app.stateDeliver.Candidates.PunishByzantineCandidate(height, address)
	}

	// apply frozen funds (used for unbond stakes)
	frozenFunds := app.stateDeliver.FrozenFunds.GetFrozenFunds(uint64(req.Header.Height))
	if frozenFunds != nil {
		for _, item := range frozenFunds.List {
			amount := item.Value
			if item.MoveToCandidate == nil {
				app.eventsDB.AddEvent(uint32(req.Header.Height), &eventsdb.UnbondEvent{
					Address:         item.Address,
					Amount:          amount.String(),
					Coin:            uint64(item.Coin),
					ValidatorPubKey: *item.CandidateKey,
				})
				app.stateDeliver.Accounts.AddBalance(item.Address, item.Coin, amount)
			} else {
				newCandidate := app.stateDeliver.Candidates.PubKey(*item.MoveToCandidate)
				value := big.NewInt(0).Set(amount)
				if wl := app.stateDeliver.Waitlist.Get(item.Address, newCandidate, item.Coin); wl != nil {
					value.Add(value, wl.Value)
					app.stateDeliver.Waitlist.Delete(item.Address, newCandidate, item.Coin)
				}
				if app.stateDeliver.Candidates.IsDelegatorStakeSufficient(item.Address, newCandidate, item.Coin, value) {
					app.stateDeliver.Candidates.Delegate(item.Address, newCandidate, item.Coin, value, big.NewInt(0))
				} else {
					app.stateDeliver.Waitlist.AddWaitList(item.Address, newCandidate, item.Coin, value)
				}
			}
		}

		// delete from db
		app.stateDeliver.FrozenFunds.Delete(frozenFunds.Height())
	}

	app.stateDeliver.Halts.Delete(height)

	return abciTypes.ResponseBeginBlock{}
}

// EndBlock signals the end of a block, returns changes to the validator set
func (app *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	height := uint64(req.Height)

	vals := app.stateDeliver.Validators.GetValidators()

	hasDroppedValidators := false
	for _, val := range vals {
		if !val.IsToDrop() {
			continue
		}
		hasDroppedValidators = true

		// Move dropped validator's accum rewards back to pool
		app.rewards.Add(app.rewards, val.GetAccumReward())
		val.SetAccumReward(big.NewInt(0))
	}

	// calculate total power of validators
	totalPower := big.NewInt(0)
	for _, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || app.GetValidatorStatus(val.GetAddress()) != ValidatorPresent {
			continue
		}

		totalPower.Add(totalPower, val.GetTotalBipStake())
	}

	if totalPower.Sign() == 0 {
		totalPower = big.NewInt(1)
	}

	// accumulate rewards
	reward := rewards.GetRewardForBlock(height)
	app.stateDeliver.Checker.AddCoinVolume(types.GetBaseCoinID(), reward)
	reward.Add(reward, app.rewards)

	// compute remainder to keep total emission consist
	remainder := big.NewInt(0).Set(reward)

	for i, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || app.GetValidatorStatus(val.GetAddress()) != ValidatorPresent {
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
	if height%120 == 0 {
		app.stateDeliver.Validators.PayRewards(height)
	}

	hasChangedPublicKeys := false
	if app.stateDeliver.Candidates.IsChangedPublicKeys() {
		app.stateDeliver.Candidates.ResetIsChangedPublicKeys()
		hasChangedPublicKeys = true
	}

	// update validators
	var updates []abciTypes.ValidatorUpdate
	if height%120 == 0 || hasDroppedValidators || hasChangedPublicKeys {
		updates = app.updateValidators()
	}

	defer func() {
		app.StatisticData().PushEndBlock(&statistics.EndRequest{TimeEnd: time.Now(), Height: int64(app.Height())})
	}()

	return abciTypes.ResponseEndBlock{
		ValidatorUpdates: updates,
		ConsensusParamUpdates: &abciTypes.ConsensusParams{
			Block: &abciTypes.BlockParams{
				MaxBytes: blockMaxBytes,
				MaxGas:   int64(app.stateDeliver.App.GetMaxGas()),
			},
		},
	}
}

func (app *Blockchain) updateValidators() []abciTypes.ValidatorUpdate {
	height := app.Height()
	app.stateDeliver.Candidates.RecalculateStakes(height)

	valsCount := validators.GetValidatorsCountForBlock(height)
	newCandidates := app.stateDeliver.Candidates.GetNewCandidates(valsCount)
	if len(newCandidates) < valsCount {
		valsCount = len(newCandidates)
	}

	newValidators := make([]abciTypes.ValidatorUpdate, 0, valsCount)

	// calculate total power
	totalPower := big.NewInt(0)
	for _, candidate := range newCandidates {
		totalPower.Add(totalPower, app.stateDeliver.Candidates.GetTotalStake(candidate.PubKey))
	}

	for _, newCandidate := range newCandidates {
		power := big.NewInt(0).Div(big.NewInt(0).Mul(app.stateDeliver.Candidates.GetTotalStake(newCandidate.PubKey),
			big.NewInt(100000000)), totalPower).Int64()

		if power == 0 {
			power = 1
		}

		newValidators = append(newValidators, abciTypes.Ed25519ValidatorUpdate(newCandidate.PubKey.Bytes(), power))
	}

	sort.SliceStable(newValidators, func(i, j int) bool {
		return newValidators[i].Power > newValidators[j].Power
	})

	// update validators in state
	app.stateDeliver.Validators.SetNewValidators(newCandidates)

	activeValidators := app.appDB.GetValidators()

	app.appDB.SetValidators(newValidators)

	updates := newValidators

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
	return updates
}

// Info return application info. Used for synchronization between Tendermint and Minter
func (app *Blockchain) Info(_ abciTypes.RequestInfo) (resInfo abciTypes.ResponseInfo) {
	return abciTypes.ResponseInfo{
		Version:          version.Version,
		AppVersion:       version.AppVer,
		LastBlockHeight:  int64(app.appDB.GetLastHeight()),
		LastBlockAppHash: app.appDB.GetLastBlockHash(),
	}
}

// DeliverTx deliver a tx for full processing
func (app *Blockchain) DeliverTx(req abciTypes.RequestDeliverTx) abciTypes.ResponseDeliverTx {
	response := transaction.RunTx(app.stateDeliver, req.Tx, app.rewards, app.Height(), &sync.Map{}, 0)

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

// CheckTx validates a tx for the mempool
func (app *Blockchain) CheckTx(req abciTypes.RequestCheckTx) abciTypes.ResponseCheckTx {
	response := transaction.RunTx(app.CurrentState(), req.Tx, nil, app.height, app.currentMempool, app.MinGasPrice())

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
	if err := app.stateDeliver.Check(); err != nil {
		panic(err)
	}

	// Committing Minter Blockchain state
	hash, err := app.stateDeliver.Commit()
	if err != nil {
		panic(err)
	}

	// Flush events db
	err = app.eventsDB.CommitEvents()
	if err != nil {
		panic(err)
	}

	// Persist application hash and height
	app.appDB.SetLastBlockHash(hash)
	app.appDB.SetLastHeight(app.Height())
	app.appDB.FlushValidators()

	app.stateDeliver.Unlock()

	// Resetting check state to be consistent with current height
	app.resetCheckState()

	// Clear mempool
	app.currentMempool = &sync.Map{}

	return abciTypes.ResponseCommit{
		Data: hash,
	}
}

// Query Unused method, required by Tendermint
func (app *Blockchain) Query(_ abciTypes.RequestQuery) abciTypes.ResponseQuery {
	return abciTypes.ResponseQuery{}
}

// SetOption Unused method, required by Tendermint
func (app *Blockchain) SetOption(_ abciTypes.RequestSetOption) abciTypes.ResponseSetOption {
	return abciTypes.ResponseSetOption{}
}

// Stop gracefully stopping Minter Blockchain instance
func (app *Blockchain) Stop() {
	app.appDB.Close()
	if err := app.stateDB.Close(); err != nil {
		panic(err)
	}
	if err := app.eventsDB.Close(); err != nil {
		panic(err)
	}
}

// CurrentState returns immutable state of Minter Blockchain
func (app *Blockchain) CurrentState() *state.CheckState {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return app.stateCheck
}

// AvailableVersions returns all available versions in ascending order
func (app *Blockchain) AvailableVersions() []int {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return app.stateDeliver.Tree().AvailableVersions()
}

// GetStateForHeight returns immutable state of Minter Blockchain for given height
func (app *Blockchain) GetStateForHeight(height uint64) (*state.CheckState, error) {
	if height > 0 {
		s, err := state.NewCheckStateAtHeight(height, app.stateDB)
		if err != nil {
			return nil, err
		}
		return s, nil
	}
	return blockchain.CurrentState(), nil
}

// Height returns current height of Minter Blockchain
func (app *Blockchain) Height() uint64 {
	return atomic.LoadUint64(&app.height)
}

// SetTmNode sets Tendermint node
func (app *Blockchain) SetTmNode(node *tmNode.Node) {
	app.tmNode = node
}

// MinGasPrice returns minimal acceptable gas price
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

// SetBlocksTimeDelta sets current blocks time delta
func (app *Blockchain) SetBlocksTimeDelta(height uint64, value int) {
	app.appDB.SetLastBlocksTimeDelta(height, value)
}

// GetBlocksTimeDelta returns current blocks time delta
func (app *Blockchain) GetBlocksTimeDelta(height, _ uint64) (int, error) {
	return app.appDB.GetLastBlocksTimeDelta(height)
}

func (app *Blockchain) calcMaxGas(height uint64) uint64 {
	const targetTime = 7
	const blockDelta = 3

	// skip first 20 blocks
	if height <= 20 {
		return defaultMaxGas
	}

	// get current max gas
	newMaxGas := app.stateCheck.App().GetMaxGas()

	// check if blocks are created in time
	if delta, _ := app.GetBlocksTimeDelta(height, blockDelta); delta > targetTime*blockDelta {
		newMaxGas = newMaxGas * 7 / 10 // decrease by 30%
	} else {
		newMaxGas = newMaxGas * 105 / 100 // increase by 5%
	}

	// check if max gas is too high
	if newMaxGas > defaultMaxGas {
		newMaxGas = defaultMaxGas
	}

	// check if max gas is too low
	if newMaxGas < minMaxGas {
		newMaxGas = minMaxGas
	}

	return newMaxGas
}

// GetEventsDB returns current EventsDB
func (app *Blockchain) GetEventsDB() eventsdb.IEventsDB {
	return app.eventsDB
}

// SetStatisticData used for collection statistics about blockchain operations
func (app *Blockchain) SetStatisticData(statisticData *statistics.Data) *statistics.Data {
	app.statisticData = statisticData
	return app.statisticData
}

// StatisticData used for collection statistics about blockchain operations
func (app *Blockchain) StatisticData() *statistics.Data {
	return app.statisticData
}

// GetValidatorStatus returns given validator's status
func (app *Blockchain) GetValidatorStatus(address types.TmAddress) int8 {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return app.validatorsStatuses[address]
}

// DeleteStateVersions deletes states in given range
func (app *Blockchain) DeleteStateVersions(from, to int64) error {
	app.lock.RLock()
	defer app.lock.RUnlock()

	return app.stateDeliver.Tree().DeleteVersionsRange(from, to)
}

func (app *Blockchain) isApplicationHalted(height uint64) bool {
	if app.haltHeight > 0 && height >= app.haltHeight {
		return true
	}

	halts := app.stateDeliver.Halts.GetHaltBlocks(height)
	if halts != nil {
		// calculate total power of validators
		vals := app.stateDeliver.Validators.GetValidators()
		totalPower, totalVotedPower := big.NewInt(0), big.NewInt(0)
		for _, val := range vals {
			// skip if candidate is not present
			if val.IsToDrop() || app.validatorsStatuses[val.GetAddress()] != ValidatorPresent {
				continue
			}

			for _, halt := range halts.List {
				if halt.Pubkey == val.PubKey {
					totalVotedPower.Add(totalVotedPower, val.GetTotalBipStake())
				}
			}

			totalPower.Add(totalPower, val.GetTotalBipStake())
		}

		if totalPower.Cmp(types.Big0) == 0 {
			totalPower = big.NewInt(1)
		}

		votingResult := new(big.Float).Quo(
			new(big.Float).SetInt(totalVotedPower),
			new(big.Float).SetInt(totalPower),
		)

		if votingResult.Cmp(big.NewFloat(votingPowerConsensus)) == 1 {
			return true
		}
	}

	return false
}

func getDbOpts(memLimit int) *opt.Options {
	if memLimit < 1024 {
		panic(fmt.Sprintf("Not enough memory given to StateDB. Expected >1024M, given %d", memLimit))
	}
	return &opt.Options{
		OpenFilesCacheCapacity: memLimit,
		BlockCacheCapacity:     memLimit / 2 * opt.MiB,
		WriteBuffer:            memLimit / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	}
}
