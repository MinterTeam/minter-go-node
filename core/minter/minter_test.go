package minter

import (
	"context"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/genesis"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
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

func init() {
	*utils.MinterHome = os.ExpandEnv(filepath.Join("$HOME", ".minter_test"))
	_ = os.RemoveAll(*utils.MinterHome)

	cfg = config.GetTmConfig()

	pv = privval.GenFilePV(cfg.PrivValidatorFile())
	pv.Save()
}

func TestMinterBlockchain(t *testing.T) {
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
		t.Fatalf("Failed to create a node: %v", err)
	}

	if err = node.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	log.Info("Started node", "nodeInfo", node.Switch().NodeInfo())
	app.SetTmNode(node)
	client := rpc.NewLocal(node)

	// Wait for blocks
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	blocks := make(chan interface{})
	err = client.Subscribe(ctx, "test-client", query.MustParse("tm.event = 'NewBlock'"), blocks)
	if err != nil {
		panic(err)
	}

	select {
	case <-blocks:
		// got block
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for the first block")
	}

	err = node.Stop()
	if err != nil {
		panic(err)
	}
	app.Stop()
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
		FirstValidatorAddress: types.HexToAddress("Mxee81347211c72524338f9680072af90744333146"),
		InitialBalances: []genesis.Account{
			{
				Address: types.HexToAddress("Mxee81347211c72524338f9680072af90744333146"),
				Balance: map[string]string{
					"MNT": helpers.BipToPip(big.NewInt(100000000)).String(),
				},
			},
			{
				Address: types.HexToAddress("Mx184ac726059e43643e67290666f7b3195093f870"),
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
