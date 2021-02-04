package minter

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/developers"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/statistics"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/rlp"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmnet "github.com/tendermint/tendermint/libs/net"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	p2pmock "github.com/tendermint/tendermint/p2p/mock"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	types2 "github.com/tendermint/tendermint/types"
	"math/big"
	"testing"
	"time"
)

func initTestNode(t *testing.T) (*Blockchain, *rpc.Local, *privval.FilePV, func()) {
	storage := utils.NewStorage(t.TempDir(), "")
	minterCfg := config.GetConfig(storage.GetMinterHome())
	logger := log.NewLogger(minterCfg)
	cfg := config.GetTmConfig(minterCfg)
	cfg.Consensus.TimeoutPropose = 0
	cfg.Consensus.TimeoutPrecommit = 0
	cfg.Consensus.TimeoutPrevote = 0
	cfg.Consensus.TimeoutCommit = 0
	cfg.Consensus.TimeoutPrecommitDelta = 0
	cfg.Consensus.TimeoutPrevoteDelta = 0
	cfg.Consensus.TimeoutProposeDelta = 0
	cfg.Consensus.SkipTimeoutCommit = true
	cfg.RPC.ListenAddress = ""
	cfg.P2P.ListenAddress = fmt.Sprintf("tcp://127.0.0.1:%d", getPort())
	cfg.P2P.Seeds = ""
	cfg.P2P.PersistentPeers = ""
	cfg.DBBackend = "memdb"

	pv := privval.GenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())
	pv.Save()

	ctx, cancelFunc := context.WithCancel(context.Background())

	app := NewMinterBlockchain(storage, minterCfg, ctx)
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		t.Fatal(err)
	}

	node, err := tmNode.NewNode(
		cfg,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		getTestGenesis(pv, storage.GetMinterHome()),
		tmNode.DefaultDBProvider,
		tmNode.DefaultMetricsProvider(cfg.Instrumentation),
		logger,
		tmNode.CustomReactors(map[string]p2p.Reactor{
			// "PEX":        p2pmock.NewReactor(),
			"BLOCKCHAIN": p2pmock.NewReactor(),
		}),
	)
	if err != nil {
		t.Fatal(fmt.Sprintf("Failed to create a node: %v", err))
	}

	// logger.Info("Started node", "nodeInfo", node.Switch().NodeInfo())
	app.SetTmNode(node)

	if err = app.tmNode.Start(); err != nil {
		t.Fatal(fmt.Sprintf("Failed to start node: %v", err))
	}

	tmCli := rpc.New(app.tmNode)

	blocks, err := tmCli.Subscribe(context.Background(), "test-client", types2.EventQueryNewBlock.String())
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-blocks:
		err = tmCli.UnsubscribeAll(context.Background(), "test-client")
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for the first block")
	}

	return app, tmCli, pv, func() {
		cancelFunc()
		if err := app.WaitStop(); err != nil {
			t.Error(err)
		}
	}
}

func TestBlockchain_Run(t *testing.T) {
	_, _, _, cancel := initTestNode(t)
	cancel()
}

func TestBlockchain_Height(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t)
	defer cancel()

	blocks, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = tmCli.UnsubscribeAll(context.Background(), "test-client")
		if err != nil {
			t.Fatal(err)
		}
	}()

	block := <-blocks
	if block.Data.(types2.EventDataNewBlock).Block.Height != int64(blockchain.Height()) {
		t.Fatal("invalid blockchain height")
	}

	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_SetStatisticData(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t)
	defer cancel()

	ch := make(chan struct{})
	blockchain.stateDeliver.Lock()
	go func() {
		close(ch)
		blockchain.SetStatisticData(statistics.New()).Statistic(context.Background())
	}()
	<-ch
	time.Sleep(time.Second)
	blockchain.stateDeliver.Unlock()

	blocks, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = tmCli.UnsubscribeAll(context.Background(), "test-client")
		if err != nil {
			t.Fatal(err)
		}
	}()

	<-blocks
	<-blocks
	<-blocks
	block := <-blocks
	if block.Data.(types2.EventDataNewBlock).Block.Header.Time.Nanosecond() != blockchain.StatisticData().BlockEnd.LastBlockInfo.HeaderTimestamp.Nanosecond() {
		t.Fatal("statistic last block and event event last block header time not equal")
	}

	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()

	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_IsApplicationHalted(t *testing.T) {
	blockchain, tmCli, pv, cancel := initTestNode(t)
	defer cancel()
	data := transaction.SetHaltBlockData{
		PubKey: types.BytesToPubkey(pv.Key.PubKey.Bytes()[:]),
		Height: 5,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.Transaction{
		Nonce:         1,
		ChainID:       types.CurrentChainID,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeSetHaltBlock,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}

	if err := tx.Sign(getPrivateKey()); err != nil {
		t.Fatal(err)
	}

	txBytes, _ := tx.Serialize()
	res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d, %s", res.Code, res.Log)
	}

	blocks, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = tmCli.UnsubscribeAll(context.Background(), "test-client")
		if err != nil {
			t.Fatal(err)
		}
	}()

	for {
		select {
		case block := <-blocks:
			height := block.Data.(types2.EventDataNewBlock).Block.Height
			if height < int64(data.Height) {
				continue
			}

			t.Fatalf("don't stop on block %d", height)
			return
		case <-time.After(2 * time.Second):
			blockchain.lock.RLock()
			exportedState := blockchain.CurrentState().Export()
			blockchain.lock.RUnlock()
			if err := exportedState.Verify(); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
}

func TestBlockchain_GetStateForHeightAndDeleteStateVersions(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t)
	defer cancel()

	symbol := types.StrToCoinSymbol("AAA123")
	data := transaction.CreateCoinData{
		Name:                 "nAAA123",
		Symbol:               symbol,
		InitialAmount:        helpers.BipToPip(big.NewInt(1000000)),
		InitialReserve:       helpers.BipToPip(big.NewInt(10000)),
		ConstantReserveRatio: 70,
		MaxSupply:            big.NewInt(0).Exp(big.NewInt(10), big.NewInt(15+18), nil),
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.Transaction{
		Nonce:         1,
		ChainID:       types.CurrentChainID,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeCreateCoin,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}

	if err := tx.Sign(getPrivateKey()); err != nil {
		t.Fatal(err)
	}

	txBytes, _ := tx.Serialize()
	res, err := tmCli.BroadcastTxCommit(context.Background(), txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	time.Sleep(time.Second)

	resultTx, err := tmCli.Tx(context.Background(), res.Hash.Bytes(), false)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	checkState := blockchain.CurrentState()

	if !checkState.Coins().ExistsBySymbol(symbol) {
		t.Fatalf("Failed: %s", "state invalid")
	}

	checkState, err = blockchain.GetStateForHeight(uint64(resultTx.Height - 1))
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	if checkState.Coins().ExistsBySymbol(symbol) {
		t.Fatalf("Failed: %s", "state invalid")
	}

	err = blockchain.DeleteStateVersions(0, resultTx.Height)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	_, err = blockchain.GetStateForHeight(uint64(resultTx.Height - 1))
	if err == nil {
		t.Fatalf("Failed: %s", "state not deleted")
	}

	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_SendTx(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t)
	defer cancel()

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

	nonce := uint64(1)
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

	if err := tx.Sign(getPrivateKey()); err != nil {
		t.Fatal(err)
	}

	txBytes, _ := tx.Serialize()

	res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}

	txs, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'Tx'")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = tmCli.UnsubscribeAll(context.Background(), "test-client")
		if err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case <-txs:
		// got tx
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for the tx to be committed")
	}

	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_FrozenFunds(t *testing.T) {
	blockchain, tmCli, pv, cancel := initTestNode(t)
	defer cancel()

	targetHeight := uint64(10)
	value := helpers.BipToPip(big.NewInt(1000))
	pubkey := types.BytesToPubkey(pv.Key.PubKey.Bytes()[:])
	blockchain.stateDeliver.RLock()
	blockchain.stateDeliver.Candidates.SubStake(developers.Address, pubkey, 0, big.NewInt(0).Set(value))
	blockchain.stateDeliver.FrozenFunds.AddFund(targetHeight, developers.Address, pubkey, blockchain.stateDeliver.Candidates.ID(pubkey), 0, big.NewInt(0).Set(value), nil)
	blockchain.stateDeliver.RUnlock()

	blocks, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		t.Fatal(err)
	}

	for block := range blocks {
		if block.Data.(types2.EventDataNewBlock).Block.Height < int64(targetHeight) {
			continue
		}
		break
	}

	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}

	events := blockchain.GetEventsDB().LoadEvents(uint32(targetHeight))

	if len(events) == 0 {
		t.Errorf("empty events for %d block", targetHeight)
	}
	if events[0].Type() != eventsdb.TypeUnbondEvent {
		t.Fatal("event is not StakeMoveEvent")
	}
	if events[0].(eventsdb.Stake).AddressString() != developers.Address.String() {
		t.Error("event address invalid")
	}
	if events[0].(eventsdb.Stake).ValidatorPubKeyString() != pubkey.String() {
		t.Error("event validator pubkey invalid")
	}

}

func TestBlockchain_RecalculateStakes_andRemoveValidator(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t)
	defer cancel()

	txs, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'Tx'")
	if err != nil {
		t.Fatal(err)
	}
	symbol := types.StrToCoinSymbol("AAA123")
	nonce := uint64(1)
	{
		data := transaction.CreateCoinData{
			Name:                 "nAAA123",
			Symbol:               symbol,
			InitialAmount:        helpers.BipToPip(big.NewInt(1000000)),
			InitialReserve:       helpers.BipToPip(big.NewInt(10000)),
			ConstantReserveRatio: 70,
			MaxSupply:            big.NewInt(0).Exp(big.NewInt(10), big.NewInt(15+18), nil),
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
			Type:          transaction.TypeCreateCoin,
			Data:          encodedData,
			SignatureType: transaction.SigTypeSingle,
		}

		if err := tx.Sign(getPrivateKey()); err != nil {
			t.Fatal(err)
		}

		txBytes, _ := tx.Serialize()
		res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}
		if res.Code != 0 {
			t.Fatalf("CheckTx code is not 0: %d", res.Code)
		}
	}
	<-txs
	nonce++
	blockchain.lock.RLock()
	coinID := blockchain.CurrentState().Coins().GetCoinBySymbol(symbol, 0).ID()
	blockchain.lock.RUnlock()
	{
		buyCoinData := transaction.BuyCoinData{
			CoinToBuy:          coinID,
			ValueToBuy:         helpers.BipToPip(big.NewInt(10000000)),
			CoinToSell:         0,
			MaximumValueToSell: helpers.BipToPip(big.NewInt(10000000000000000)),
		}

		encodedData, err := rlp.EncodeToBytes(buyCoinData)
		if err != nil {
			t.Fatal(err)
		}

		tx := transaction.Transaction{
			Nonce:         nonce,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          transaction.TypeBuyCoin,
			Data:          encodedData,
			SignatureType: transaction.SigTypeSingle,
		}

		if err := tx.Sign(getPrivateKey()); err != nil {
			t.Fatal(err)
		}

		txBytes, _ := tx.Serialize()
		res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}
		if res.Code != 0 {
			t.Fatalf("CheckTx code is not 0: %d", res.Code)
		}
	}
	<-txs
	nonce++
	{
		data := transaction.DeclareCandidacyData{
			Address:    types.Address{1},
			PubKey:     types.Pubkey{1},
			Commission: 10,
			Coin:       0,
			Stake:      helpers.BipToPip(big.NewInt(1000000)),
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

		if err := tx.Sign(getPrivateKey()); err != nil {
			t.Fatal(err)
		}

		txBytes, _ := tx.Serialize()

		res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}

		if res.Code != 0 {
			t.Fatalf("CheckTx code is not 0: %d, %s", res.Code, res.Log)
		}
	}
	<-txs
	nonce++
	{
		data := transaction.DelegateData{
			PubKey: types.Pubkey{1},
			Coin:   coinID,
			Value:  helpers.BipToPip(big.NewInt(9000000)),
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
			Type:          transaction.TypeDelegate,
			Data:          encodedData,
			SignatureType: transaction.SigTypeSingle,
		}

		if err := tx.Sign(getPrivateKey()); err != nil {
			t.Fatal(err)
		}

		txBytes, _ := tx.Serialize()

		res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}

		if res.Code != 0 {
			t.Fatalf("CheckTx code is not 0: %d, %s", res.Code, res.Log)
		}
	}
	<-txs
	nonce++
	{
		data := transaction.SetCandidateOnData{
			PubKey: types.Pubkey{1},
		}

		encodedData, err := rlp.EncodeToBytes(data)
		if err != nil {
			t.Fatal(err)
		}

		tx := transaction.Transaction{
			Nonce:         nonce,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          transaction.TypeSetCandidateOnline,
			Data:          encodedData,
			SignatureType: transaction.SigTypeSingle,
		}

		if err := tx.Sign(getPrivateKey()); err != nil {
			t.Fatal(err)
		}

		txBytes, _ := tx.Serialize()
		res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}
		if res.Code != 0 {
			t.Fatalf("CheckTx code is not 0: %d", res.Code)
		}
	}
	<-txs

	err = tmCli.Unsubscribe(context.Background(), "test-client", "tm.event = 'Tx'")
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		t.Fatal(err)
	}

	var targetHeight int64
	func() {
		for {
			select {
			case block := <-blocks:
				h := block.Data.(types2.EventDataNewBlock).Block.Height
				if targetHeight == 0 {
					targetHeight = 135
				}
				if h > targetHeight {
					return
				}
			case <-time.After(time.Second * 10):
				t.Fatal("block empty")
			}
		}
	}()
	blockchain.lock.RLock()
	events := blockchain.eventsDB.LoadEvents(135)
	if len(events) == 0 {
		t.Error("no slashes")
	}
	candidate := blockchain.CurrentState().Candidates().GetCandidate(types.Pubkey{1})
	if candidate == nil {
		t.Fatal("candidate not found")
	}
	blockchain.lock.RUnlock()

	if candidate.Status == candidates.CandidateStatusOnline {
		t.Fatal("candidate not Offline")
	}

	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()

	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestStopNetworkByHaltBlocks(t *testing.T) {
	blockchain, _, _, cancel := initTestNode(t)
	cancel()

	haltHeight := uint64(50)

	v1Pubkey := types.Pubkey{1}
	v2Pubkey := types.Pubkey{2}
	v3Pubkey := types.Pubkey{3}

	blockchain.validatorsStatuses = map[types.TmAddress]int8{}
	blockchain.validatorsPowers[v1Pubkey] = helpers.BipToPip(big.NewInt(3))
	blockchain.validatorsPowers[v2Pubkey] = helpers.BipToPip(big.NewInt(5))
	blockchain.validatorsPowers[v3Pubkey] = helpers.BipToPip(big.NewInt(3))
	blockchain.totalPower = helpers.BipToPip(big.NewInt(11))

	blockchain.stateDeliver.Halts.AddHaltBlock(haltHeight, v1Pubkey)
	blockchain.stateDeliver.Halts.AddHaltBlock(haltHeight, v3Pubkey)
	if blockchain.isApplicationHalted(haltHeight) {
		t.Fatalf("Application halted at height %d", haltHeight)
	}

	haltHeight++
	blockchain.stateDeliver.Halts.AddHaltBlock(haltHeight, v1Pubkey)
	blockchain.stateDeliver.Halts.AddHaltBlock(haltHeight, v2Pubkey)
	if !blockchain.isApplicationHalted(haltHeight) {
		t.Fatalf("Application not halted at height %d", haltHeight)
	}
}

func getPrivateKey() *ecdsa.PrivateKey {
	b, _ := hex.DecodeString("825ca965c34ef1c8343e8e377959108370c23ba6194d858452b63432456403f9")
	privateKey, _ := crypto.ToECDSA(b)
	return privateKey
}

func makeTestValidatorsAndCandidates(pubkeys []string, stake *big.Int) ([]types.Validator, []types.Candidate) {
	vals := make([]types.Validator, 0, len(pubkeys))
	cands := make([]types.Candidate, 0, len(pubkeys))

	for i, val := range pubkeys {
		// pkeyBytes, err := base64.StdEncoding.DecodeString(val)
		pkeyBytes := []byte(val)
		// if err != nil {
		// 	panic(err)
		// }

		var pkey types.Pubkey
		copy(pkey[:], pkeyBytes)
		addr := developers.Address

		vals = append(vals, types.Validator{
			TotalBipStake: stake.String(),
			PubKey:        pkey,
			AccumReward:   big.NewInt(0).String(),
			AbsentTimes:   types.NewBitArray(24),
		})

		cands = append(cands, types.Candidate{
			ID:             uint64(i) + 1,
			RewardAddress:  addr,
			OwnerAddress:   crypto.PubkeyToAddress(getPrivateKey().PublicKey),
			ControlAddress: addr,
			TotalBipStake:  stake.String(),
			PubKey:         pkey,
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    addr,
					Coin:     uint64(types.GetBaseCoinID()),
					Value:    stake.String(),
					BipValue: stake.String(),
				},
			},
			Status: candidates.CandidateStatusOnline,
		})
	}

	return vals, cands
}

func getTestGenesis(pv *privval.FilePV, home string) func() (*types2.GenesisDoc, error) {
	return func() (*types2.GenesisDoc, error) {
		validators, candidates := makeTestValidatorsAndCandidates([]string{string(pv.Key.PubKey.Bytes()[:])}, helpers.BipToPip(big.NewInt(12444011)))

		appState := types.AppState{
			// StartHeight: 100, // FIXME
			TotalSlashed: "0",
			Accounts: []types.Account{
				{
					Address: crypto.PubkeyToAddress(getPrivateKey().PublicKey),
					Balance: []types.Balance{
						{
							Coin:  uint64(types.GetBaseCoinID()),
							Value: helpers.BipToPip(big.NewInt(9223372036854775807)).String(),
						},
					},
				},
			},
			Validators: validators,
			Candidates: candidates,
		}

		appStateJSON, err := tmjson.Marshal(appState)
		if err != nil {
			return nil, err
		}

		genesisDoc := types2.GenesisDoc{
			ChainID:     "minter-test-network",
			GenesisTime: time.Now(),
			AppHash:     nil,
			AppState:    json.RawMessage(appStateJSON),
		}

		err = genesisDoc.ValidateAndComplete()
		if err != nil {
			return nil, err
		}

		genesisFile := home + "/config/genesis.json"
		if err := genesisDoc.SaveAs(genesisFile); err != nil {
			panic(err)
		}

		return &genesisDoc, nil
	}
}

func getPort() int {
	port, err := tmnet.GetFreePort()
	if err != nil {
		panic(err)
	}
	return port
}
