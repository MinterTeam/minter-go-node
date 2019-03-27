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
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/genesis"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/rlp"
	tmConfig "github.com/tendermint/tendermint/config"
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
var client *rpc.Local
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
	*utils.MinterHome = os.ExpandEnv(filepath.Join("$HOME", ".minter_test"))
	_ = os.RemoveAll(*utils.MinterHome)

	cfg = config.GetTmConfig()
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

	pv = privval.GenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())
	pv.Save()

	b, _ := hex.DecodeString("825ca965c34ef1c8343e8e377959108370c23ba6194d858452b63432456403f9")
	privateKey, _ = crypto.ToECDSA(b)

	app = NewMinterBlockchain()
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
	client = rpc.NewLocal(node)
	l.Unlock()
}

func TestBlocksCreation(t *testing.T) {
	// Wait for blocks
	blocks, err := client.Subscribe(context.TODO(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		panic(err)
	}

	select {
	case <-blocks:
		// got block
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for the first block")
	}

	err = client.UnsubscribeAll(context.TODO(), "test-client")
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
		GasPrice:      big.NewInt(1),
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

	res, err := client.BroadcastTxSync(txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	txs, err := client.Subscribe(context.TODO(), "test-client", "tm.event = 'Tx'")
	if err != nil {
		panic(err)
	}

	select {
	case <-txs:
		// got tx
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for the tx to be committed")
	}

	err = client.UnsubscribeAll(context.TODO(), "test-client")
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
		GasPrice:      big.NewInt(1),
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
	res, err := client.BroadcastTxSync(txBytes)
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
		GasPrice:      big.NewInt(1),
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
	res, err = client.BroadcastTxSync(txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	status, _ := client.Status()
	targetBlockHeight := status.SyncInfo.LatestBlockHeight - (status.SyncInfo.LatestBlockHeight % 120) + 150
	println("target block", targetBlockHeight)

	blocks, err := client.Subscribe(context.TODO(), "test-client", "tm.event = 'NewBlock'")
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

			vals, _ := client.Validators(&targetBlockHeight)

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
	err = client.UnsubscribeAll(context.TODO(), "test-client")
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
		GasPrice:      big.NewInt(1),
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
	res, err = client.BroadcastTxSync(txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	status, _ = client.Status()
	targetBlockHeight = status.SyncInfo.LatestBlockHeight - (status.SyncInfo.LatestBlockHeight % 120) + 120 + 5
	println("target block", targetBlockHeight)

	blocks, err = client.Subscribe(context.TODO(), "test-client", "tm.event = 'NewBlock'")
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

			vals, _ := client.Validators(&targetBlockHeight)

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

	err = client.UnsubscribeAll(context.TODO(), "test-client")
	if err != nil {
		panic(err)
	}
}

func getGenesis() (*types2.GenesisDoc, error) {
	appHash := [32]byte{}

	validators, candidates := genesis.MakeValidatorsAndCandidates([]string{base64.StdEncoding.EncodeToString(pv.Key.PubKey.Bytes()[5:])}, big.NewInt(10000000))

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

	return &genesisDoc, nil
}
