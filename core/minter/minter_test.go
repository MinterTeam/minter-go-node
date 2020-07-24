package minter

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/developers"
	candidates2 "github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/upgrades"
	"github.com/tendermint/go-amino"
	tmConfig "github.com/tendermint/tendermint/config"
	log2 "github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	_ "github.com/tendermint/tendermint/types"
	types2 "github.com/tendermint/tendermint/types"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var pv *privval.FilePV
var cfg *tmConfig.Config
var tmCli *rpc.Local
var app *Blockchain
var privateKey *ecdsa.PrivateKey
var l sync.Mutex
var nonce = uint64(1)

func init() {
	l.Lock()
	go initNode()
	l.Lock()
}

func initNode() {
	utils.MinterHome = os.ExpandEnv(filepath.Join("$HOME", ".minter_test"))
	_ = os.RemoveAll(utils.MinterHome)

	if err := tmos.EnsureDir(utils.GetMinterHome()+"/tmdata/blockstore.db", 0777); err != nil {
		panic(err.Error())
	}

	minterCfg := config.GetConfig()
	logger := log.NewLogger(minterCfg)
	cfg = config.GetTmConfig(minterCfg)
	cfg.Consensus.TimeoutPropose = 0
	cfg.Consensus.TimeoutPrecommit = 0
	cfg.Consensus.TimeoutPrevote = 0
	cfg.Consensus.TimeoutCommit = 0
	cfg.Consensus.TimeoutPrecommitDelta = 0
	cfg.Consensus.TimeoutPrevoteDelta = 0
	cfg.Consensus.TimeoutProposeDelta = 0
	cfg.Consensus.SkipTimeoutCommit = true
	cfg.RPC.ListenAddress = ""
	cfg.P2P.ListenAddress = "0.0.0.0:25566"
	cfg.P2P.Seeds = ""
	cfg.P2P.PersistentPeers = ""
	cfg.DBBackend = "memdb"

	pv = privval.GenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())
	pv.Save()

	b, _ := hex.DecodeString("825ca965c34ef1c8343e8e377959108370c23ba6194d858452b63432456403f9")
	privateKey, _ = crypto.ToECDSA(b)

	app = NewMinterBlockchain(minterCfg)
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		panic(err)
	}

	node, err := tmNode.NewNode(
		cfg,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		getGenesis,
		tmNode.DefaultDBProvider,
		tmNode.DefaultMetricsProvider(cfg.Instrumentation),
		log2.NewTMLogger(os.Stdout),
	)

	if err != nil {
		panic(fmt.Sprintf("Failed to create a node: %v", err))
	}

	if err = node.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start node: %v", err))
	}

	logger.Info("Started node", "nodeInfo", node.Switch().NodeInfo())
	app.SetTmNode(node)
	tmCli = rpc.New(node)
	l.Unlock()
}

func TestBlocksCreation(t *testing.T) {
	// Wait for blocks
	blocks, err := tmCli.Subscribe(context.TODO(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		panic(err)
	}

	select {
	case <-blocks:
		// got block
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for the first block")
	}

	err = tmCli.UnsubscribeAll(context.TODO(), "test-client")
	if err != nil {
		panic(err)
	}
}

func TestSendTx(t *testing.T) {
	for blockchain.Height() < 2 {
		time.Sleep(time.Millisecond)
	}

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := transaction.SendData{
		Coin:  types.GetBaseCoinID(),
		To:    to,
		Value: value,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.Transaction{
		Nonce:         nonce,
		ChainID:       types.CurrentChainID,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeSend,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}
	nonce++

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	txBytes, _ := tx.Serialize()

	res, err := tmCli.BroadcastTxSync(txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	txs, err := tmCli.Subscribe(context.TODO(), "test-client", "tm.event = 'Tx'")
	if err != nil {
		panic(err)
	}

	select {
	case <-txs:
		// got tx
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for the tx to be committed")
	}

	err = tmCli.UnsubscribeAll(context.TODO(), "test-client")
	if err != nil {
		panic(err)
	}
}

// TODO: refactor
func TestSmallStakeValidator(t *testing.T) {
	for blockchain.Height() < 2 {
		time.Sleep(time.Millisecond)
	}

	pubkey := types.Pubkey{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	data := transaction.DeclareCandidacyData{
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		PubKey:     pubkey,
		Commission: 10,
		Coin:       types.GetBaseCoinID(),
		Stake:      big.NewInt(0),
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.Transaction{
		Nonce:         nonce,
		ChainID:       types.CurrentChainID,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeDeclareCandidacy,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}
	nonce++

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	txBytes, _ := tx.Serialize()
	res, err := tmCli.BroadcastTxSync(txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	time.Sleep(time.Second)

	setOnData := transaction.SetCandidateOnData{
		PubKey: pubkey,
	}

	encodedData, err = rlp.EncodeToBytes(setOnData)
	if err != nil {
		t.Fatal(err)
	}

	tx = transaction.Transaction{
		Nonce:         nonce,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeSetCandidateOnline,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}
	nonce++

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	txBytes, _ = tx.Serialize()
	res, err = tmCli.BroadcastTxSync(txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	status, _ := tmCli.Status()
	targetBlockHeight := status.SyncInfo.LatestBlockHeight - (status.SyncInfo.LatestBlockHeight % 120) + 150
	println("target block", targetBlockHeight)

	blocks, err := tmCli.Subscribe(context.TODO(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		panic(err)
	}

	ready := false
	for !ready {
		select {
		case block := <-blocks:
			if block.Data.(types2.EventDataNewBlock).Block.Height < targetBlockHeight {
				continue
			}

			vals, _ := tmCli.Validators(&targetBlockHeight, 1, 1000)

			if len(vals.Validators) > 1 {
				t.Errorf("There are should be 1 validator (has %d)", len(vals.Validators))
			}

			if len(app.stateDeliver.Validators.GetValidators()) > 1 {
				t.Errorf("There are should be 1 validator (has %d)", len(app.stateDeliver.Validators.GetValidators()))
			}

			ready = true
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for the block")
		}
	}
	err = tmCli.UnsubscribeAll(context.TODO(), "test-client")
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)

	encodedData, err = rlp.EncodeToBytes(setOnData)
	if err != nil {
		t.Fatal(err)
	}

	tx = transaction.Transaction{
		Nonce:         nonce,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeSetCandidateOnline,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}
	nonce++

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	txBytes, _ = tx.Serialize()
	res, err = tmCli.BroadcastTxSync(txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	status, _ = tmCli.Status()
	targetBlockHeight = status.SyncInfo.LatestBlockHeight - (status.SyncInfo.LatestBlockHeight % 120) + 120 + 5
	println("target block", targetBlockHeight)

	blocks, err = tmCli.Subscribe(context.TODO(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		panic(err)
	}

FORLOOP2:
	for {
		select {
		case block := <-blocks:
			if block.Data.(types2.EventDataNewBlock).Block.Height < targetBlockHeight {
				continue FORLOOP2
			}

			vals, _ := tmCli.Validators(&targetBlockHeight, 1, 100)

			if len(vals.Validators) > 1 {
				t.Errorf("There should be only 1 validator, got %d", len(vals.Validators))
			}

			mvals := app.stateDeliver.Validators.GetValidators()
			if len(mvals) > 1 {
				t.Errorf("There should be only 1 validator, got %d", len(mvals))
			}

			break FORLOOP2
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for the block")
		}
	}

	err = tmCli.UnsubscribeAll(context.TODO(), "test-client")
	if err != nil {
		panic(err)
	}
}

func TestStopNetworkByHaltBlocks(t *testing.T) {
	haltHeight := upgrades.UpgradeBlock4 + uint64(5)

	v1Pubkey := [32]byte{}
	v2Pubkey := [32]byte{}
	v3Pubkey := [32]byte{}

	rand.Read(v1Pubkey[:])
	rand.Read(v2Pubkey[:])
	rand.Read(v3Pubkey[:])

	app.stateDeliver.Validators.Create(v1Pubkey, helpers.BipToPip(big.NewInt(3)))
	app.stateDeliver.Validators.Create(v2Pubkey, helpers.BipToPip(big.NewInt(5)))
	app.stateDeliver.Validators.Create(v3Pubkey, helpers.BipToPip(big.NewInt(3)))

	v1Address := app.stateDeliver.Validators.GetValidators()[1].GetAddress()
	v2Address := app.stateDeliver.Validators.GetValidators()[2].GetAddress()
	v3Address := app.stateDeliver.Validators.GetValidators()[3].GetAddress()

	app.validatorsStatuses = map[types.TmAddress]int8{}
	app.validatorsStatuses[v1Address] = ValidatorPresent
	app.validatorsStatuses[v2Address] = ValidatorPresent
	app.validatorsStatuses[v3Address] = ValidatorPresent

	app.stateDeliver.Halts.AddHaltBlock(haltHeight, v1Pubkey)
	app.stateDeliver.Halts.AddHaltBlock(haltHeight, v3Pubkey)
	if app.isApplicationHalted(haltHeight) {
		t.Fatalf("Application halted at height %d", haltHeight)
	}

	haltHeight++
	app.stateDeliver.Halts.AddHaltBlock(haltHeight, v1Pubkey)
	app.stateDeliver.Halts.AddHaltBlock(haltHeight, v2Pubkey)
	if !app.isApplicationHalted(haltHeight) {
		t.Fatalf("Application not halted at height %d", haltHeight)
	}
}

func getGenesis() (*types2.GenesisDoc, error) {
	appHash := [32]byte{}

	validators, candidates := makeValidatorsAndCandidates([]string{base64.StdEncoding.EncodeToString(pv.Key.PubKey.Bytes()[5:])}, big.NewInt(10000000))

	appState := types.AppState{
		TotalSlashed: "0",
		Accounts: []types.Account{
			{
				Address: crypto.PubkeyToAddress(privateKey.PublicKey),
				Balance: []types.Balance{
					{
						Coin:  types.GetBaseCoinID(),
						Value: helpers.BipToPip(big.NewInt(1000000)).String(),
					},
				},
			},
		},
		Validators: validators,
		Candidates: candidates,
	}

	appStateJSON, err := amino.MarshalJSON(appState)
	if err != nil {
		return nil, err
	}

	genesisDoc := types2.GenesisDoc{
		ChainID:     "minter-test-network",
		GenesisTime: time.Now(),
		AppHash:     appHash[:],
		AppState:    json.RawMessage(appStateJSON),
	}

	err = genesisDoc.ValidateAndComplete()
	if err != nil {
		return nil, err
	}

	genesisFile := utils.GetMinterHome() + "/config/genesis.json"
	if err := genesisDoc.SaveAs(genesisFile); err != nil {
		panic(err)
	}

	return &genesisDoc, nil
}

func makeValidatorsAndCandidates(pubkeys []string, stake *big.Int) ([]types.Validator, []types.Candidate) {
	validators := make([]types.Validator, len(pubkeys))
	candidates := make([]types.Candidate, len(pubkeys))
	addr := developers.Address

	for i, val := range pubkeys {
		pkeyBytes, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			panic(err)
		}

		var pkey types.Pubkey
		copy(pkey[:], pkeyBytes)

		validators[i] = types.Validator{
			TotalBipStake: stake.String(),
			PubKey:        pkey,
			AccumReward:   big.NewInt(0).String(),
			AbsentTimes:   types.NewBitArray(24),
		}

		candidates[i] = types.Candidate{
			RewardAddress: addr,
			OwnerAddress:  addr,
			TotalBipStake: big.NewInt(1).String(),
			PubKey:        pkey,
			Commission:    100,
			Stakes: []types.Stake{
				{
					Owner:    addr,
					Coin:     types.GetBaseCoinID(),
					Value:    stake.String(),
					BipValue: stake.String(),
				},
			},
			Status: candidates2.CandidateStatusOnline,
		}
	}

	return validators, candidates
}
