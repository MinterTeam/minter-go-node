package minter

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/cosmos/cosmos-sdk/snapshots"
	snapshottypes "github.com/cosmos/cosmos-sdk/snapshots/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	tmlog "github.com/tendermint/tendermint/libs/log"

	l "github.com/MinterTeam/minter-go-node/log"

	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/appdb"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/rewards"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
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

	appDB        *appdb.AppDB
	eventsDB     eventsdb.IEventsDB
	stateDeliver *state.State
	stateCheck   *state.CheckState
	height       uint64   // current Blockchain height
	rewards      *big.Int // Rewards pool

	lockValidators     sync.RWMutex
	validatorsStatuses map[types.TmAddress]int8
	validatorsPowers   map[types.Pubkey]*big.Int
	totalPower         *big.Int

	rewardsCounter                  *rewards.Reward
	updateStakesAndPayRewardsPeriod uint64
	expiredOrdersPeriod             uint64
	// local rpc client for Tendermint
	rpcClient *rpc.Local

	tmNode *tmNode.Node

	// currentMempool is responsive for prevent sending multiple transactions from one address in one block
	currentMempool *sync.Map

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

func (blockchain *Blockchain) Executor() transaction.ExecutorTx {
	return blockchain.executor
}

func (blockchain *Blockchain) UpdateStakesAndPayRewardsPeriod() uint64 {
	return blockchain.updateStakesAndPayRewardsPeriod
}

func (blockchain *Blockchain) ExpiredOrdersPeriod() uint64 {
	return blockchain.expiredOrdersPeriod
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
			V3:   {}, // tokenomics
			V310: {}, // hotfix
			V320: {},
			V330: {},
			V340: {}, // TODO: Only for release version
		},
		executor: GetExecutor(V3),
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
	//case V3:
	//	return transaction.NewExecutorV3(transaction.GetDataV3)
	//case v260, v261, v262:
	//	return transaction.NewExecutorV250(transaction.GetDataV260)
	//case v250:
	//	return transaction.NewExecutorV250(transaction.GetDataV250)
	//case v230:
	//	return transaction.NewExecutor(transaction.GetDataV230)
	default:
		return transaction.NewExecutorV3(transaction.GetDataV3)
	}
}

const ( // known update versions
	V3   = "v300" // tokenomics
	V310 = "v310" // hotfix
	V320 = "v320" // hotfix
	V330 = "v330" // hotfix
	V340 = "v340" // hotfix
)

func (blockchain *Blockchain) initState() {
	initialHeight := blockchain.appDB.GetStartHeight()
	currentHeight := blockchain.appDB.GetLastHeight()

	var stateDeliver *state.State
	var err error
	stateDeliver, err = state.NewStateV3(currentHeight,
		blockchain.storages.StateDB(),
		blockchain.eventsDB,
		blockchain.cfg.StateCacheSize,
		blockchain.cfg.KeepLastStates,
		initialHeight)

	if err != nil {
		panic(err)
	}
	blockchain.appDB.SetState(stateDeliver.Tree())

	height := currentHeight
	if height == 0 {
		height = initialHeight
	}
	atomic.StoreUint64(&blockchain.height, height)
	blockchain.stateDeliver = stateDeliver
	blockchain.stateCheck = state.NewCheckState(stateDeliver)

	blockchain.grace = upgrades.NewGrace()
	blockchain.grace.AddGracePeriods(upgrades.NewGracePeriod(initialHeight, initialHeight+120, true))

	for _, v := range blockchain.UpdateVersions() {
		blockchain.grace.AddGracePeriods(graceForUpdate(v.Height))
		blockchain.executor = GetExecutor(v.Name)
	}

}

// InitChain initialize blockchain with validators and other info. Only called once.
func (blockchain *Blockchain) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {
	var genesisState types.AppState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	initialHeight := uint64(req.InitialHeight) - 1

	blockchain.appDB.SetStartHeight(initialHeight)
	if len(genesisState.Versions) == 0 {
		blockchain.appDB.AddVersion(genesisState.Version, initialHeight)
	} else {
		for _, history := range genesisState.Versions {
			blockchain.appDB.AddVersion(history.Name, history.Height)
		}
	}
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

	blockchain.appDB.SetEmission(helpers.StringToBigInt(genesisState.Emission))

	blockchain.appDB.SetPrice(
		time.Unix(0, int64(genesisState.PrevReward.Time)).UTC(),
		helpers.StringToBigInt(genesisState.PrevReward.AmountBIP),
		helpers.StringToBigInt(genesisState.PrevReward.AmountUSDT),
		helpers.StringToBigInt(genesisState.PrevReward.Reward),
		genesisState.PrevReward.Off)

	blockchain.appDB.SaveStartHeight()
	blockchain.appDB.SaveVersions()
	blockchain.appDB.SaveEmission()
	blockchain.appDB.SavePrice()

	defer blockchain.appDB.FlushValidators()
	return abciTypes.ResponseInitChain{
		Validators: blockchain.updateValidators(),
	}
}

// BeginBlock signals the beginning of a block.
func (blockchain *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	height := uint64(req.Header.Height)
	if blockchain.stateDeliver == nil {
		blockchain.initState()
	}

	if emission := blockchain.appDB.Emission(); emission.Cmp(blockchain.rewardsCounter.TotalEmissionBig()) == -1 {
		t, _, _, _, _ := blockchain.appDB.GetPrice()
		if height%blockchain.updateStakesAndPayRewardsPeriod == 1 && (t.IsZero() || (req.Header.Time.Hour() >= 12 && req.Header.Time.Hour() <= 14) && req.Header.Time.Sub(t) > 3*time.Hour) {
			reserve0, reserve1 := blockchain.stateCheck.Swap().GetSwapper(0, types.USDTID).Reserves()
			funcUpdatePrice := blockchain.appDB.UpdatePriceBug
			if h := blockchain.appDB.GetVersionHeight(V320); h > 0 && height > h {
				funcUpdatePrice = blockchain.appDB.UpdatePriceFix
			}
			newRewards, safeReward := funcUpdatePrice(req.Header.Time, reserve0, reserve1)
			blockchain.stateDeliver.App.SetReward(newRewards, safeReward)
			blockchain.eventsDB.AddEvent(&eventsdb.UpdatedBlockRewardEvent{Value: newRewards.String(), ValueLockedStakeRewards: new(big.Int).Mul(safeReward, big.NewInt(3)).String()})
		}
	} else {
		blockchain.stateDeliver.App.SetReward(big.NewInt(0), big.NewInt(0))
	}

	blockchain.StatisticData().PushStartBlock(&statistics.StartRequest{Height: int64(height), Now: time.Now(), HeaderTime: req.Header.Time})

	// compute max gas
	maxGas := blockchain.calcMaxGas()
	blockchain.stateDeliver.App.SetMaxGas(maxGas)
	blockchain.appDB.AddBlocksTime(req.Header.Time)

	blockchain.rewards.SetInt64(0)

	// clear absent candidates
	blockchain.lockValidators.Lock()
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
	blockchain.lockValidators.Unlock()

	blockchain.calculatePowers(blockchain.stateDeliver.Validators.GetValidators())

	if blockchain.isApplicationHalted(height) && !blockchain.grace.IsUpgradeBlock(height) {
		log.Printf("Application halted at height %d\n", height)
		blockchain.stop()
		return abciTypes.ResponseBeginBlock{}
	}

	versionName := blockchain.appDB.GetVersionName(height)
	if _, ok := blockchain.knownUpdates[versionName]; !ok {
		log.Printf("Update your node binary to the latest version: %s, height: %d", versionName, height)
		blockchain.stop()
		return abciTypes.ResponseBeginBlock{}
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
			if item.GetMoveToCandidateID() == 0 {
				if item.CandidateKey != nil {
					blockchain.eventsDB.AddEvent(&eventsdb.UnbondEvent{
						Address:         item.Address,
						Amount:          amount.String(),
						Coin:            uint64(item.Coin),
						ValidatorPubKey: item.CandidateKey,
					})
				} else {
					blockchain.eventsDB.AddEvent(&eventsdb.UnlockEvent{
						Address: item.Address,
						Amount:  amount.String(),
						Coin:    uint64(item.Coin),
					})
				}
				blockchain.stateDeliver.Accounts.AddBalance(item.Address, item.Coin, amount)
			} else {
				moveTo := blockchain.stateDeliver.Candidates.PubKey(item.GetMoveToCandidateID())
				blockchain.eventsDB.AddEvent(&eventsdb.StakeMoveEvent{
					Address:           item.Address,
					Amount:            amount.String(),
					Coin:              uint64(item.Coin),
					CandidatePubKey:   *item.CandidateKey,
					ToCandidatePubKey: moveTo,
				})
				blockchain.stateDeliver.Candidates.Delegate(item.Address, moveTo, item.Coin, amount, big.NewInt(0))
			}
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
	var heightIsMaxIfIssueIsOverOrNotDynamic uint64 = math.MaxUint64
	emission := blockchain.appDB.Emission()
	if emission.Cmp(blockchain.rewardsCounter.TotalEmissionBig()) == -1 {
		reward, _ = blockchain.stateDeliver.App.Reward()
		heightIsMaxIfIssueIsOverOrNotDynamic = height
	}

	{
		rewardWithTxs := big.NewInt(0).Add(reward, blockchain.rewards)

		// compute remainder to keep total emission consist
		remainder := big.NewInt(0).Set(rewardWithTxs)

		for i, val := range vals {
			// skip if candidate is not present
			if val.IsToDrop() || blockchain.GetValidatorStatus(val.GetAddress()) != ValidatorPresent {
				continue
			}

			r := big.NewInt(0).Set(rewardWithTxs)
			r.Mul(r, val.GetTotalBipStake())
			r.Div(r, blockchain.totalPower)

			remainder.Sub(remainder, r)
			vals[i].AddAccumReward(r)
		}

		// add remainder to total slashed
		blockchain.stateDeliver.App.AddTotalSlashed(remainder)
	}

	// expire orders
	if height > blockchain.expiredOrdersPeriod && height%blockchain.updateStakesAndPayRewardsPeriod == blockchain.updateStakesAndPayRewardsPeriod/2 {
		blockchain.stateDeliver.Swapper().ExpireOrders(height - blockchain.expiredOrdersPeriod)
	}

	// pay rewards
	var moreRewards = big.NewInt(0)
	if height%blockchain.updateStakesAndPayRewardsPeriod == 0 {
		PayRewards := blockchain.stateDeliver.Validators.PayRewardsV3

		if h := blockchain.appDB.GetVersionHeight(V340); h > 0 && height > h {
			PayRewards = blockchain.stateDeliver.Validators.PayRewardsV5Fix2
		} else if h := blockchain.appDB.GetVersionHeight(V330); h > 0 && height > h {
			if height < h+blockchain.updateStakesAndPayRewardsPeriod && types.CurrentChainID == types.ChainMainnet {
				excess := blockchain.stateDeliver.Candidates.FixStakesAfter10509400()
				blockchain.appDB.SetEmission(big.NewInt(0).Sub(blockchain.appDB.Emission(), excess))
				log.Println("fixEmission", blockchain.appDB.Emission())
			}
			PayRewards = blockchain.stateDeliver.Validators.PayRewardsV5Fix
		} else if h := blockchain.appDB.GetVersionHeight(V320); h > 0 && height > h {
			PayRewards = blockchain.stateDeliver.Validators.PayRewardsV5Bug
		} else if h := blockchain.appDB.GetVersionHeight(V310); h > 0 && height > h {
			PayRewards = blockchain.stateDeliver.Validators.PayRewardsV4
		}

		moreRewards = PayRewards(heightIsMaxIfIssueIsOverOrNotDynamic, int64(blockchain.updateStakesAndPayRewardsPeriod))
		blockchain.appDB.SetEmission(big.NewInt(0).Add(blockchain.appDB.Emission(), moreRewards))
		blockchain.stateDeliver.Checker.AddCoinVolume(types.GetBaseCoinID(), moreRewards)

	}

	if heightIsMaxIfIssueIsOverOrNotDynamic != math.MaxUint64 {
		_, rewardForBlock := blockchain.CurrentState().App().Reward()
		blockchain.appDB.SetEmission(big.NewInt(0).Add(blockchain.appDB.Emission(), rewardForBlock))
		if diff := big.NewInt(0).Sub(rewardForBlock, reward); diff.Sign() == 1 {
			blockchain.stateDeliver.Accounts.AddBalance([20]byte{}, 0, diff)
			reward.Add(reward, diff)
		}
	}

	blockchain.stateDeliver.Checker.AddCoinVolume(types.GetBaseCoinID(), reward)

	{
		updateCommissionsBlockPrices := blockchain.isUpdateCommissionsBlockV2(height)
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
				FailedTx:                price.FailedTx.String(),
				AddLimitOrder:           price.AddLimitOrder.String(),
				RemoveLimitOrder:        price.RemoveLimitOrder.String(),
				MoveStake:               price.MoveStake.String(),
				LockStake:               price.LockStake.String(),
				Lock:                    price.Lock.String(),
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
	height := blockchain.Height()

	if err := blockchain.stateDeliver.Check(); err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("height %d", height)))
	}

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

	{ // Persist application hash and height
		blockchain.appDB.SetLastBlockHash(hash)
		blockchain.appDB.SetLastHeight(height)

		blockchain.appDB.FlushValidators()
		blockchain.appDB.SaveBlocksTime()
		blockchain.appDB.SaveVersions()
		blockchain.appDB.SaveEmission()
		blockchain.appDB.SavePrice()
	}

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
