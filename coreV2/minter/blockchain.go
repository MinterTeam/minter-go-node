package minter

import (
	"context"
	"github.com/cosmos/cosmos-sdk/snapshots"
	snapshottypes "github.com/cosmos/cosmos-sdk/snapshots/types"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"log"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"time"

	l "github.com/MinterTeam/minter-go-node/log"

	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/appdb"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/rewards"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/coreV2/statistics"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/upgrades"
	"github.com/MinterTeam/minter-go-node/version"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmNode "github.com/tendermint/tendermint/node"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
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

// Blockchain is a main structure of Minter
type Blockchain struct {
	abciTypes.BaseApplication

	logger tmlog.Logger

	executor      transaction.ExecutorTx
	statisticData *statistics.Data

	appDB                           *appdb.AppDB
	eventsDB                        eventsdb.IEventsDB
	stateDeliver                    *state.State
	stateCheck                      *state.CheckState
	height                          uint64   // current Blockchain height
	rewards                         *big.Int // Rewards pool
	validatorsStatuses              map[types.TmAddress]int8
	validatorsPowers                map[types.Pubkey]*big.Int
	totalPower                      *big.Int
	rewardsCounter                  *rewards.Reward
	updateStakesAndPayRewardsPeriod uint64
	expiredOrdersPeriod             uint64
	// local rpc client for Tendermint
	rpcClient *rpc.Local

	tmNode *tmNode.Node

	// currentMempool is responsive for prevent sending multiple transactions from one address in one block
	currentMempool *sync.Map

	lock         sync.RWMutex
	haltHeight   uint64
	cfg          *config.Config
	storages     *utils.Storage
	stopChan     context.Context
	stopped      bool
	grace        *upgrades.Grace
	knownUpdates map[string]struct{}
	stopOk       chan struct{}

	// manages snapshots, i.e. dumps of app state at certain intervals
	snapshotManager    *snapshots.Manager
	snapshotInterval   uint64 // block interval between state sync snapshots
	snapshotKeepRecent uint32 // recent state sync snapshots to keep
	snapshotter        snapshottypes.Snapshotter
	wgSnapshot         sync.WaitGroup
}

func (blockchain *Blockchain) GetCurrentRewards() *big.Int {
	return blockchain.rewards
}

// NewMinterBlockchain creates Minter Blockchain instance, should be only called once
func NewMinterBlockchain(storages *utils.Storage, cfg *config.Config, ctx context.Context, updateStakePeriod uint64, expiredOrdersPeriod uint64, logger tmlog.Logger) *Blockchain {
	// Initiate Application DB. Used for persisting data like current block, validators, etc.
	applicationDB := appdb.NewAppDB(storages.GetMinterHome(), cfg)
	applicationDB.SetStateDB(storages.StateDB())
	if ctx == nil {
		ctx = context.Background()
	}
	var eventsDB eventsdb.IEventsDB
	if !cfg.ValidatorMode {
		eventsDB = eventsdb.NewEventsStore(storages.EventDB())
	} else {
		eventsDB = &eventsdb.MockEvents{}
	}
	const updateStakesAndPayRewards = 720
	if updateStakePeriod == 0 {
		updateStakePeriod = updateStakesAndPayRewards
	}
	if expiredOrdersPeriod == 0 {
		expiredOrdersPeriod = types.GetExpireOrdersPeriod()
	}
	if logger == nil {
		logger = l.NewLogger(cfg)
	}
	app := &Blockchain{
		logger: logger,

		rewards:                         big.NewInt(0),
		rewardsCounter:                  rewards.NewReward(),
		appDB:                           applicationDB,
		storages:                        storages,
		eventsDB:                        eventsDB,
		currentMempool:                  &sync.Map{},
		cfg:                             cfg,
		stopChan:                        ctx,
		haltHeight:                      uint64(cfg.HaltHeight),
		updateStakesAndPayRewardsPeriod: updateStakePeriod,
		expiredOrdersPeriod:             expiredOrdersPeriod,
		stopOk:                          make(chan struct{}),
		knownUpdates: map[string]struct{}{
			"":   {}, // default version
			v230: {}, // add more for update
			v250: {}, // commissions and mempool
			v260: {}, // amm with orderbook
			v261: {}, // hothix
			v3:   {}, // tokenomics
		},
		executor: GetExecutor(""),
	}
	if applicationDB.GetStartHeight() != 0 {
		app.initState()
	}
	return app
}

func graceForUpdate(height uint64) *upgrades.GracePeriod {
	return upgrades.NewGracePeriod(height, height+120, false)
}

func GetExecutor(v string) transaction.ExecutorTx {
	switch v {
	case v3:
		return transaction.NewExecutorV250(transaction.GetDataV3)
	case v260, v261:
		return transaction.NewExecutorV250(transaction.GetDataV260)
	case v250:
		return transaction.NewExecutorV250(transaction.GetDataV250)
	case v230:
		return transaction.NewExecutor(transaction.GetDataV230)
	default:
		return transaction.NewExecutor(transaction.GetDataV1)
	}
}

const haltBlockV210 = 3431238
const updateBlockV240 = 4448826

const ( // known update versions
	v2   = ""     // default
	v230 = "v230" // remove liquidity bug
	v250 = "v250" // commissions and failed txs
	v260 = "v260" // orderbook
	v261 = "v261" // hotfix
	v3   = "v300" // tokenomics
)

func (blockchain *Blockchain) initState() {
	initialHeight := blockchain.appDB.GetStartHeight()
	currentHeight := blockchain.appDB.GetLastHeight()
	stateDeliver, err := state.NewState(currentHeight,
		blockchain.storages.StateDB(),
		blockchain.eventsDB,
		blockchain.cfg.StateCacheSize,
		blockchain.cfg.KeepLastStates,
		initialHeight)
	if err != nil {
		panic(err)
	}
	blockchain.appDB.SetState(stateDeliver.Tree())

	atomic.StoreUint64(&blockchain.height, currentHeight)
	blockchain.stateDeliver = stateDeliver
	blockchain.stateCheck = state.NewCheckState(stateDeliver)

	grace := upgrades.NewGrace()
	grace.AddGracePeriods(upgrades.NewGracePeriod(initialHeight, initialHeight+120, true),
		upgrades.NewGracePeriod(haltBlockV210, haltBlockV210+120, true),
		upgrades.NewGracePeriod(3612653, 3612653+120, true),
		upgrades.NewGracePeriod(updateBlockV240, updateBlockV240+120, true))

	for _, v := range blockchain.UpdateVersions() {
		grace.AddGracePeriods(graceForUpdate(v.Height))
		blockchain.executor = GetExecutor(v.Name)
	}

	blockchain.grace = grace
}

// InitChain initialize blockchain with validators and other info. Only called once.
func (blockchain *Blockchain) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {
	var genesisState types.AppState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	initialHeight := uint64(req.InitialHeight) - 1

	blockchain.appDB.SetStartHeight(initialHeight)
	blockchain.appDB.AddVersion(genesisState.Version, initialHeight)
	blockchain.initState()

	if err := blockchain.stateDeliver.Import(genesisState, genesisState.Version); err != nil {
		panic(err)
	}
	if err := blockchain.stateDeliver.Check(); err != nil {
		panic(err)
	}
	_, err := blockchain.stateDeliver.Commit()
	if err != nil {
		panic(err)
	}

	lastHeight := initialHeight
	blockchain.appDB.SetLastHeight(lastHeight)

	blockchain.appDB.SaveStartHeight()
	blockchain.appDB.SaveVersions()

	defer blockchain.appDB.FlushValidators()
	return abciTypes.ResponseInitChain{
		Validators: blockchain.updateValidators(),
	}
}

// BeginBlock signals the beginning of a block.
func (blockchain *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	if blockchain.stateDeliver == nil {
		blockchain.initState()
	}
	height := uint64(req.Header.Height)

	log.Println(blockchain.CurrentState().Swap().GetSwapper(0, 1993).Reserves())

	if h := blockchain.appDB.GetVersionHeight(v3); h > 0 && height > h {
		t, _, _ := blockchain.appDB.GetPrice()
		if t.Sub(req.Header.Time) > time.Hour*24 && req.Header.Time.Hour() > 12 {
			reserve0, reserve1 := blockchain.CurrentState().Swap().GetSwapper(0, 1993).Reserves()
			blockchain.appDB.SetPrice(req.Header.Time, reserve0, reserve1)
		}
	} else if h == height {
		reserve0, reserve1 := blockchain.CurrentState().Swap().GetSwapper(0, 1993).Reserves()
		blockchain.appDB.SetPrice(req.Header.Time, reserve0, reserve1)
	}

	blockchain.StatisticData().PushStartBlock(&statistics.StartRequest{Height: int64(height), Now: time.Now(), HeaderTime: req.Header.Time})

	// compute max gas
	maxGas := blockchain.calcMaxGas()
	blockchain.stateDeliver.App.SetMaxGas(maxGas)
	blockchain.appDB.AddBlocksTime(req.Header.Time)

	blockchain.rewards.SetInt64(0)

	// clear absent candidates
	blockchain.lock.Lock()
	blockchain.validatorsStatuses = map[types.TmAddress]int8{}
	// give penalty to absent validators
	for _, v := range req.LastCommitInfo.Votes {
		var address types.TmAddress
		copy(address[:], v.Validator.Address)

		if v.SignedLastBlock {
			blockchain.stateDeliver.Validators.SetValidatorPresent(height, address)
			blockchain.validatorsStatuses[address] = ValidatorPresent
		} else {
			blockchain.stateDeliver.Validators.SetValidatorAbsent(height, address, blockchain.grace)
			blockchain.validatorsStatuses[address] = ValidatorAbsent
		}
	}
	blockchain.lock.Unlock()

	blockchain.calculatePowers(blockchain.stateDeliver.Validators.GetValidators())

	if blockchain.isApplicationHalted(height) && !blockchain.grace.IsUpgradeBlock(height) {
		log.Printf("Application halted at height %d\n", height)
		blockchain.stop()
		return abciTypes.ResponseBeginBlock{}
	}

	versionName := blockchain.appDB.GetVersionName(height)
	if _, ok := blockchain.knownUpdates[versionName]; !ok {
		log.Printf("Update your node binary to the latest version: %s", versionName)
		blockchain.stop()
		return abciTypes.ResponseBeginBlock{}
	}

	if versionName == v230 && height > updateBlockV240 {
		blockchain.executor = transaction.NewExecutor(transaction.GetDataV240)
	}

	// give penalty to Byzantine validators
	for _, byzVal := range req.ByzantineValidators {
		var address types.TmAddress
		copy(address[:], byzVal.Validator.Address)

		// skip already offline candidates to prevent double punishing
		candidate := blockchain.stateDeliver.Candidates.GetCandidateByTendermintAddress(address)
		if candidate == nil || candidate.Status == candidates.CandidateStatusOffline || blockchain.stateDeliver.Validators.GetByTmAddress(address) == nil {
			continue
		}

		blockchain.stateDeliver.FrozenFunds.PunishFrozenFundsWithID(height, height+types.GetUnbondPeriod(), candidate.ID)
		blockchain.stateDeliver.Validators.PunishByzantineValidator(address)
		blockchain.stateDeliver.Candidates.PunishByzantineCandidate(height, address)
	}

	// apply frozen funds (used for unbond stakes)
	frozenFunds := blockchain.stateDeliver.FrozenFunds.GetFrozenFunds(height)
	if frozenFunds != nil {
		for _, item := range frozenFunds.List {
			amount := item.Value
			blockchain.eventsDB.AddEvent(&eventsdb.UnbondEvent{
				Address:         item.Address,
				Amount:          amount.String(),
				Coin:            uint64(item.Coin),
				ValidatorPubKey: item.CandidateKey,
			})
			blockchain.stateDeliver.Accounts.AddBalance(item.Address, item.Coin, amount)
		}

		// delete from db
		blockchain.stateDeliver.FrozenFunds.Delete(frozenFunds.Height())
	}

	blockchain.stateDeliver.Halts.Delete(height)

	return abciTypes.ResponseBeginBlock{}
}

// EndBlock signals the end of a block, returns changes to the validator set
func (blockchain *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	height := uint64(req.Height)
	atomic.StoreUint64(&blockchain.height, height)
	vals := blockchain.stateDeliver.Validators.GetValidators()

	hasDroppedValidators := false
	for _, val := range vals {
		if !val.IsToDrop() {
			continue
		}
		hasDroppedValidators = true

		// Move dropped validator's accum rewards back to pool
		blockchain.rewards.Add(blockchain.rewards, val.GetAccumReward())
		val.SetAccumReward(big.NewInt(0))
	}

	blockchain.calculatePowers(vals)

	// accumulate rewards
	var reward = big.NewInt(0)
	var emission *big.Int
	if h := blockchain.appDB.GetVersionHeight(v3); h > 0 && height > h {
		emission = blockchain.appDB.Emission()
		if emission.String() != rewards.TotalEmission {
			reward = blockchain.appDB.GetReward()
			if reward.Cmp(big.NewInt(1000)) == 1 {
				reward.Set(big.NewInt(1000))
			} else if diff := big.NewInt(0).Sub(blockchain.rewardsCounter.GetRewardForBlock(height), reward); diff.Sign() == 1 {
				blockchain.stateDeliver.Accounts.AddBalance([20]byte{}, 0, diff)
			}
			emission.Add(emission, reward)
			if emission.Cmp(rewards.TotalEmissionBig) == 1 {
				emission.Set(rewards.TotalEmissionBig)
			}
			blockchain.appDB.SetEmission(emission)
		}
	} else if h == height {
		reward = blockchain.rewardsCounter.GetRewardForBlock(height)
		emission = blockchain.rewardsCounter.GetBeforeBlock(height)
		emission.Add(emission, reward)
		blockchain.appDB.SetEmission(emission)
	} else {
		reward = blockchain.rewardsCounter.GetRewardForBlock(height)
	}

	blockchain.stateDeliver.Checker.AddCoinVolume(types.GetBaseCoinID(), reward)
	reward.Add(reward, blockchain.rewards)

	// compute remainder to keep total emission consist
	remainder := big.NewInt(0).Set(reward)

	for i, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || blockchain.GetValidatorStatus(val.GetAddress()) != ValidatorPresent {
			continue
		}

		r := big.NewInt(0).Set(reward)
		r.Mul(r, val.GetTotalBipStake())
		r.Div(r, blockchain.totalPower)

		remainder.Sub(remainder, r)
		vals[i].AddAccumReward(r)
	}

	// add remainder to total slashed
	blockchain.stateDeliver.App.AddTotalSlashed(remainder)

	// expire orders
	if height > blockchain.expiredOrdersPeriod && height%blockchain.updateStakesAndPayRewardsPeriod == blockchain.updateStakesAndPayRewardsPeriod/2 {
		blockchain.stateDeliver.Swap.ExpireOrders(height - blockchain.expiredOrdersPeriod)
	}

	// pay rewards
	if height%blockchain.updateStakesAndPayRewardsPeriod == 0 {
		if h := blockchain.appDB.GetVersionHeight(v3); h > 0 && height > h {
			blockchain.stateDeliver.Validators.PayRewardsV3()
		} else {
			blockchain.stateDeliver.Validators.PayRewards()
		}
	}

	{
		var updateCommissionsBlockPrices []byte
		if height < haltBlockV210 {
			updateCommissionsBlockPrices = blockchain.isUpdateCommissionsBlock(height)
		} else {
			updateCommissionsBlockPrices = blockchain.isUpdateCommissionsBlockV2(height)
		}
		if prices := updateCommissionsBlockPrices; len(prices) != 0 {
			blockchain.stateDeliver.Commission.SetNewCommissions(prices)
			price := blockchain.stateDeliver.Commission.GetCommissions()
			blockchain.eventsDB.AddEvent(&eventsdb.UpdateCommissionsEvent{
				Coin:                    uint64(price.Coin),
				PayloadByte:             price.PayloadByte.String(),
				Send:                    price.Send.String(),
				BuyBancor:               price.BuyBancor.String(),
				SellBancor:              price.SellBancor.String(),
				SellAllBancor:           price.SellAllBancor.String(),
				BuyPoolBase:             price.BuyPoolBase.String(),
				BuyPoolDelta:            price.BuyPoolDelta.String(),
				SellPoolBase:            price.SellPoolBase.String(),
				SellPoolDelta:           price.SellPoolDelta.String(),
				SellAllPoolBase:         price.SellAllPoolBase.String(),
				SellAllPoolDelta:        price.SellAllPoolDelta.String(),
				CreateTicker3:           price.CreateTicker3.String(),
				CreateTicker4:           price.CreateTicker4.String(),
				CreateTicker5:           price.CreateTicker5.String(),
				CreateTicker6:           price.CreateTicker6.String(),
				CreateTicker7_10:        price.CreateTicker7to10.String(),
				CreateCoin:              price.CreateCoin.String(),
				CreateToken:             price.CreateToken.String(),
				RecreateCoin:            price.RecreateCoin.String(),
				RecreateToken:           price.RecreateToken.String(),
				DeclareCandidacy:        price.DeclareCandidacy.String(),
				Delegate:                price.Delegate.String(),
				Unbond:                  price.Unbond.String(),
				RedeemCheck:             price.RedeemCheck.String(),
				SetCandidateOn:          price.SetCandidateOn.String(),
				SetCandidateOff:         price.SetCandidateOff.String(),
				CreateMultisig:          price.CreateMultisig.String(),
				MultisendBase:           price.MultisendBase.String(),
				MultisendDelta:          price.MultisendDelta.String(),
				EditCandidate:           price.EditCandidate.String(),
				SetHaltBlock:            price.SetHaltBlock.String(),
				EditTickerOwner:         price.EditTickerOwner.String(),
				EditMultisig:            price.EditMultisig.String(),
				EditCandidatePublicKey:  price.EditCandidatePublicKey.String(),
				CreateSwapPool:          price.CreateSwapPool.String(),
				AddLiquidity:            price.AddLiquidity.String(),
				RemoveLiquidity:         price.RemoveLiquidity.String(),
				EditCandidateCommission: price.EditCandidateCommission.String(),
				MintToken:               price.MintToken.String(),
				BurnToken:               price.BurnToken.String(),
				VoteCommission:          price.VoteCommission.String(),
				VoteUpdate:              price.VoteUpdate.String(),
				FailedTx:                price.FailedTxPrice().String(),
				AddLimitOrder:           price.AddLimitOrderPrice().String(),
				RemoveLimitOrder:        price.RemoveLimitOrderPrice().String(),
			})
		}
		blockchain.stateDeliver.Commission.Delete(height)
	}

	{
		if v, ok := blockchain.isUpdateNetworkBlockV2(height); ok {
			blockchain.appDB.AddVersion(v, height)
			blockchain.eventsDB.AddEvent(&eventsdb.UpdateNetworkEvent{
				Version: v,
			})
			blockchain.grace.AddGracePeriods(graceForUpdate(height))
			blockchain.executor = GetExecutor(v)
		}
		blockchain.stateDeliver.Updates.Delete(height)
	}

	hasChangedPublicKeys := false
	if blockchain.stateDeliver.Candidates.IsChangedPublicKeys() {
		blockchain.stateDeliver.Candidates.ResetIsChangedPublicKeys()
		hasChangedPublicKeys = true
	}

	// update validators
	var updates []abciTypes.ValidatorUpdate
	if height%blockchain.updateStakesAndPayRewardsPeriod == 0 || hasDroppedValidators || hasChangedPublicKeys {
		updates = blockchain.updateValidators()
	}

	defer func() {
		blockchain.StatisticData().PushEndBlock(&statistics.EndRequest{TimeEnd: time.Now(), Height: int64(height)})
	}()

	return abciTypes.ResponseEndBlock{
		ValidatorUpdates: updates,
		ConsensusParamUpdates: &abciTypes.ConsensusParams{
			Block: &abciTypes.BlockParams{
				MaxBytes: blockMaxBytes,
				MaxGas:   int64(blockchain.stateDeliver.App.GetMaxGas()),
			},
		},
	}
}

// Info return application info. Used for synchronization between Tendermint and Minter
func (blockchain *Blockchain) Info(_ abciTypes.RequestInfo) (resInfo abciTypes.ResponseInfo) {
	hash := blockchain.appDB.GetLastBlockHash()
	height := int64(blockchain.appDB.GetLastHeight())
	return abciTypes.ResponseInfo{
		Version:          version.Version,
		AppVersion:       version.AppVer,
		LastBlockHeight:  height,
		LastBlockAppHash: hash,
	}
}

// DeliverTx deliver a tx for full processing
func (blockchain *Blockchain) DeliverTx(req abciTypes.RequestDeliverTx) abciTypes.ResponseDeliverTx {
	response := blockchain.executor.RunTx(blockchain.stateDeliver, req.Tx, blockchain.rewards, blockchain.Height()+1, &sync.Map{}, 0, blockchain.cfg.ValidatorMode)

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
func (blockchain *Blockchain) CheckTx(req abciTypes.RequestCheckTx) abciTypes.ResponseCheckTx {
	response := blockchain.executor.RunTx(blockchain.CurrentState(), req.Tx, nil, blockchain.Height()+1, blockchain.currentMempool, blockchain.MinGasPrice(), true)

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
func (blockchain *Blockchain) Commit() abciTypes.ResponseCommit {

	panic(1)

	if blockchain.stopped {
		blockchain.wgSnapshot.Wait()
		select {
		case <-time.After(10 * time.Second):
			blockchain.Close()
			os.Exit(0)
		case <-blockchain.stopOk:
			os.Exit(0)
		}
	}
	if err := blockchain.stateDeliver.Check(); err != nil {
		panic(err)
	}

	height := blockchain.Height()

	// Flush events db
	err := blockchain.eventsDB.CommitEvents(uint32(height))
	if err != nil {
		panic(err)
	}

	// Committing Minter Blockchain state
	hash, err := blockchain.stateDeliver.Commit()
	if err != nil {
		panic(err)
	}

	// Persist application hash and height
	blockchain.appDB.SetLastBlockHash(hash)
	blockchain.appDB.SetLastHeight(height)

	blockchain.appDB.FlushValidators()
	blockchain.appDB.SaveBlocksTime()
	blockchain.appDB.SaveVersions()
	blockchain.appDB.SaveEmission()
	// todo
	//blockchain.appDB.SaveReward()
	//blockchain.appDB.SavePrice()

	// Clear mempool
	blockchain.currentMempool = &sync.Map{}

	if blockchain.checkStop() {
		return abciTypes.ResponseCommit{Data: hash}
	}

	if blockchain.snapshotInterval > 0 && height%blockchain.snapshotInterval == 0 && blockchain.snapshotManager != nil {
		blockchain.appDB.WG.Add(1)
		go blockchain.snapshot(int64(height))
	}

	return abciTypes.ResponseCommit{
		Data:         hash,
		RetainHeight: 0,
	}
}

// Query Unused method, required by Tendermint
func (blockchain *Blockchain) Query(_ abciTypes.RequestQuery) abciTypes.ResponseQuery {
	return abciTypes.ResponseQuery{}
}

// SetOption Unused method, required by Tendermint
func (blockchain *Blockchain) SetOption(_ abciTypes.RequestSetOption) abciTypes.ResponseSetOption {
	return abciTypes.ResponseSetOption{}
}

// Close closes db connections
func (blockchain *Blockchain) Close() error {
	if err := blockchain.appDB.Close(); err != nil {
		return err
	}
	if err := blockchain.storages.StateDB().Close(); err != nil {
		return err
	}
	if err := blockchain.storages.EventDB().Close(); err != nil {
		return err
	}
	if err := blockchain.storages.SnapshotDB().Close(); err != nil {
		return err
	}
	return nil
}
