package minter

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/rlp"
	tmConfig "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/common"
	log2 "github.com/tendermint/tendermint/libs/log"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client"
	_ "github.com/tendermint/tendermint/types"
	types2 "github.com/tendermint/tendermint/types"
	"math/big"
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

	if err := common.EnsureDir(utils.GetMinterHome()+"/tmdata/blockstore.db", 0777); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	minterCfg := config.GetConfig()
	log.InitLog(minterCfg)
	eventsdb.InitDB(minterCfg)
	cfg = config.GetTmConfig(minterCfg)
	cfg.Consensus.TimeoutPropose = 0
	cfg.Consensus.TimeoutPrecommit = 0
	cfg.Consensus.TimeoutPrevote = 0
	cfg.Consensus.TimeoutCommit = 0
	cfg.Consensus.TimeoutPrecommitDelta = 0
	cfg.Consensus.TimeoutPrevoteDelta = 0
	cfg.Consensus.TimeoutProposeDelta = 0
	cfg.Consensus.SkipTimeoutCommit = true
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

	log.Info("Started node", "nodeInfo", node.Switch().NodeInfo())
	app.SetTmNode(node)
	tmCli = rpc.NewLocal(node)
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
		Coin:  types.GetBaseCoin(),
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
		GasCoin:       types.GetBaseCoin(),
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
		Coin:       types.GetBaseCoin(),
		Stake:      big.NewInt(1),
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.Transaction{
		Nonce:         nonce,
		ChainID:       types.CurrentChainID,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoin(),
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
		GasCoin:       types.GetBaseCoin(),
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

			vals, _ := tmCli.Validators(&targetBlockHeight)

			if len(vals.Validators) > 1 {
				t.Errorf("There are should be 1 validator (has %d)", len(vals.Validators))
			}

			if len(app.stateDeliver.GetStateValidators().Data()) > 1 {
				t.Errorf("There are should be 1 validator (has %d)", len(app.stateDeliver.GetStateValidators().Data()))
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
		GasCoin:       types.GetBaseCoin(),
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

			vals, _ := tmCli.Validators(&targetBlockHeight)

			if len(vals.Validators) > 1 {
				t.Errorf("There are should be only 1 validator")
			}

			if len(app.stateDeliver.GetStateValidators().Data()) > 1 {
				t.Errorf("There are should be only 1 validator")
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

func getGenesis() (*types2.GenesisDoc, error) {
	appHash := [32]byte{}

	validators, candidates := makeValidatorsAndCandidates([]string{base64.StdEncoding.EncodeToString(pv.Key.PubKey.Bytes()[5:])}, big.NewInt(10000000))

	appState := types.AppState{
		Accounts: []types.Account{
			{
				Address: crypto.PubkeyToAddress(privateKey.PublicKey),
				Balance: []types.Balance{
					{
						Coin:  types.GetBaseCoin(),
						Value: helpers.BipToPip(big.NewInt(1000000)),
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
		pkey, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			panic(err)
		}

		validators[i] = types.Validator{
			RewardAddress: addr,
			TotalBipStake: stake,
			PubKey:        pkey,
			Commission:    100,
			AccumReward:   big.NewInt(0),
			AbsentTimes:   types.NewBitArray(24),
		}

		candidates[i] = types.Candidate{
			RewardAddress: addr,
			OwnerAddress:  addr,
			TotalBipStake: big.NewInt(1),
			PubKey:        pkey,
			Commission:    100,
			Stakes: []types.Stake{
				{
					Owner:    addr,
					Coin:     types.GetBaseCoin(),
					Value:    stake,
					BipValue: stake,
				},
			},
			CreatedAtBlock: 1,
			Status:         state.CandidateStatusOnline,
		}
	}

	return validators, candidates
}
