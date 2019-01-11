package minter

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	"github.com/tendermint/tendermint/libs/pubsub/query"
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
	"testing"
	"time"
)

var pv *privval.FilePV
var cfg *tmConfig.Config
var client *rpc.Local
var privateKey *ecdsa.PrivateKey

func init() {
	*utils.MinterHome = os.ExpandEnv(filepath.Join("$HOME", ".minter_test"))
	_ = os.RemoveAll(*utils.MinterHome)

	cfg = config.GetTmConfig()
	cfg.Consensus.TimeoutPropose = 0
	cfg.Consensus.TimeoutPrecommit = 0
	cfg.Consensus.TimeoutPrevote = 0
	cfg.Consensus.SkipTimeoutCommit = true

	pv = privval.GenFilePV(cfg.PrivValidatorFile())
	pv.Save()

	b, _ := hex.DecodeString("825ca965c34ef1c8343e8e377959108370c23ba6194d858452b63432456403f9")
	privateKey, _ = crypto.ToECDSA(b)

	app := NewMinterBlockchain()
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
		log.With("module", "tendermint"),
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
}

func TestBlocksCreation(t *testing.T) {
	// Wait for blocks
	blocks := make(chan interface{})
	err := client.Subscribe(context.TODO(), "test-client", query.MustParse("tm.event = 'NewBlock'"), blocks)
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
		Nonce:         1,
		GasPrice:      big.NewInt(1),
		GasCoin:       types.GetBaseCoin(),
		Type:          transaction.TypeSend,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}

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

	txs := make(chan interface{})
	err = client.Subscribe(context.TODO(), "test-client", query.MustParse(fmt.Sprintf("tm.event = 'Tx'")), txs)
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

func getGenesis() (*types2.GenesisDoc, error) {
	validators := []types2.GenesisValidator{
		{
			PubKey: pv.PubKey,
			Power:  100000000,
		},
	}

	appHash := [32]byte{}

	appState := genesis.AppState{
		FirstValidatorAddress: crypto.PubkeyToAddress(privateKey.PublicKey),
		InitialBalances: []genesis.Account{
			{
				Address: crypto.PubkeyToAddress(privateKey.PublicKey),
				Balance: map[string]string{
					"MNT": helpers.BipToPip(big.NewInt(100000000)).String(),
				},
			},
		},
	}

	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return nil, err
	}

	genesisDoc := types2.GenesisDoc{
		ChainID:         "minter-test-network",
		GenesisTime:     time.Now(),
		ConsensusParams: nil,
		Validators:      validators,
		AppHash:         appHash[:],
		AppState:        json.RawMessage(appStateJSON),
	}

	err = genesisDoc.ValidateAndComplete()
	if err != nil {
		return nil, err
	}

	return &genesisDoc, nil
}
