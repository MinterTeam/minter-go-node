package minter

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"math/big"
	"testing"
	"time"

	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/developers"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/coreV2/statistics"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
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
)

func initTestNode(t *testing.T, initialHeight int64) (*Blockchain, *rpc.Local, *privval.FilePV, func()) {
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

	pv := privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())

	ctx, cancelFunc := context.WithCancel(context.Background())

	app := NewMinterBlockchain(storage, minterCfg, ctx, 120, 0, nil)
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		t.Fatal(err)
	}

	node, err := tmNode.NewNode(
		cfg,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		getTestGenesis(pv, storage.GetMinterHome(), initialHeight),
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

	tmCli := app.RpcClient()

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
			if err.Error() == "leveldb: closed" {
				t.Helper()
				t.Log(err)
				return
			}
			t.Error(err)
		}
	}
}

func TestBlockchain_UpdateCommission(t *testing.T) {
	blockchain, tmCli, pv, cancel := initTestNode(t, 100)
	defer cancel()

	txs, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'Tx'")
	if err != nil {
		t.Fatal(err)
	}

	data := transaction.VoteCommissionDataV3{
		PubKey:                  types.BytesToPubkey(pv.Key.PubKey.Bytes()[:]),
		Height:                  110,
		Coin:                    0,
		PayloadByte:             helpers.StringToBigInt("200000000000000000"),
		Send:                    helpers.StringToBigInt("1000000000000000000"),
		BuyBancor:               helpers.StringToBigInt("10000000000000000000"),
		SellBancor:              helpers.StringToBigInt("10000000000000000000"),
		SellAllBancor:           helpers.StringToBigInt("10000000000000000000"),
		BuyPoolBase:             helpers.StringToBigInt("10000000000000000000"),
		BuyPoolDelta:            helpers.StringToBigInt("5000000000000000000"),
		SellPoolBase:            helpers.StringToBigInt("10000000000000000000"),
		SellPoolDelta:           helpers.StringToBigInt("5000000000000000000"),
		SellAllPoolBase:         helpers.StringToBigInt("10000000000000000000"),
		SellAllPoolDelta:        helpers.StringToBigInt("5000000000000000000"),
		CreateTicker3:           helpers.StringToBigInt("100000000000000000000000000"),
		CreateTicker4:           helpers.StringToBigInt("10000000000000000000000000"),
		CreateTicker5:           helpers.StringToBigInt("1000000000000000000000000"),
		CreateTicker6:           helpers.StringToBigInt("100000000000000000000000"),
		CreateTicker7to10:       helpers.StringToBigInt("10000000000000000000000"),
		CreateCoin:              helpers.StringToBigInt("0"),
		CreateToken:             helpers.StringToBigInt("0"),
		RecreateCoin:            helpers.StringToBigInt("1000000000000000000000000"),
		RecreateToken:           helpers.StringToBigInt("1000000000000000000000000"),
		DeclareCandidacy:        helpers.StringToBigInt("1000000000000000000000"),
		Delegate:                helpers.StringToBigInt("20000000000000000000"),
		Unbond:                  helpers.StringToBigInt("20000000000000000000"),
		RedeemCheck:             helpers.StringToBigInt("3000000000000000000"),
		SetCandidateOn:          helpers.StringToBigInt("10000000000000000000"),
		SetCandidateOff:         helpers.StringToBigInt("10000000000000000000"),
		CreateMultisig:          helpers.StringToBigInt("10000000000000000000"),
		MultisendBase:           helpers.StringToBigInt("1000000000000000000"),
		MultisendDelta:          helpers.StringToBigInt("500000000000000000"),
		EditCandidate:           helpers.StringToBigInt("1000000000000000000000"),
		SetHaltBlock:            helpers.StringToBigInt("100000000000000000000"),
		EditTickerOwner:         helpers.StringToBigInt("1000000000000000000000000"),
		EditMultisig:            helpers.StringToBigInt("100000000000000000000"),
		EditCandidatePublicKey:  helpers.StringToBigInt("10000000000000000000000000"),
		CreateSwapPool:          helpers.StringToBigInt("100000000000000000000"),
		AddLiquidity:            helpers.StringToBigInt("10000000000000000000"),
		RemoveLiquidity:         helpers.StringToBigInt("10000000000000000000"),
		EditCandidateCommission: helpers.StringToBigInt("1000000000000000000000"),
		MintToken:               helpers.StringToBigInt("10000000000000000000"),
		BurnToken:               helpers.StringToBigInt("10000000000000000000"),
		VoteCommission:          helpers.StringToBigInt("100000000000000000000"),
		VoteUpdate:              helpers.StringToBigInt("100000000000000000000"),
		FailedTx:                helpers.StringToBigInt("1000000000000000000"),
		AddLimitOrder:           helpers.StringToBigInt("1000000000000000000"),
		RemoveLimitOrder:        helpers.StringToBigInt("1000000000000000000"),
		MoveStake:               helpers.StringToBigInt("1000000000000000000"),
		LockStake:               helpers.StringToBigInt("1000000000000000000"),
		Lock:                    helpers.StringToBigInt("1000000000000000000"),
		//More: []*big.Int{
		//	helpers.StringToBigInt("1000000000000000000"),
		//	helpers.StringToBigInt("1000000000000000000"),
		//	helpers.StringToBigInt("1000000000000000000"),
		//},
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
		Type:          transaction.TypeVoteCommission,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
	}
	nonce++

	if err := tx.Sign(getPrivateKey()); err != nil {
		t.Fatal(err)
	}

	txBytes, err := tx.Serialize()
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	res, err := tmCli.BroadcastTxSync(context.Background(), txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	if res.Code != 0 {
		t.Fatalf("CheckTx code is not 0: %d", res.Code)
	}
	<-txs

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

			events := blockchain.eventsDB.LoadEvents(uint32(height))
			if len(events) == 0 {
				t.Fatalf("not found events")
			}
			if events[0].Type() != eventsdb.TypeUpdateCommissionsEvent {
				t.Fatal("not changed")
			}
			return
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
			return
		}
	}
}

func TestBlockchain_GetBlocksTimeDelta(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t, 100)
	defer cancel()

	blocks, err := tmCli.Subscribe(context.Background(), "test-client", "tm.event = 'NewBlock'")
	if err != nil {
		t.Fatal(err)
	}

	var startHeight int64
	for block := range blocks {
		height := block.Data.(types2.EventDataNewBlock).Block.Height
		if startHeight == 0 {
			startHeight = height
			continue
		}

		delta, count := blockchain.appDB.GetLastBlockTimeDelta()
		t.Log(delta, count, delta/count)
		if height > startHeight+10 {
			return
		}
	}

}

func TestBlockchain_Run(t *testing.T) {
	_, _, _, cancel := initTestNode(t, 0)
	cancel()
}

func TestBlockchain_InitialBlockHeight(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t, 100)
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

	txBytes, err := tx.Serialize()
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	res, err := tmCli.BroadcastTxCommit(context.Background(), txBytes)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	time.Sleep(time.Second)

	resultTx, err := tmCli.Tx(context.Background(), res.Hash.Bytes(), false)
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	_, err = blockchain.GetStateForHeight(uint64(resultTx.Height - 1))
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
}

func TestBlockchain_Height(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t, 100)
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

	blockchain.lockValidators.RLock()
	defer blockchain.lockValidators.RUnlock()
	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_SetStatisticData(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t, 0)
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

	blockchain.lockValidators.RLock()
	defer blockchain.lockValidators.RUnlock()

	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_IsApplicationHalted(t *testing.T) {
	//t.Skip("todo")
	blockchain, tmCli, pv, cancel := initTestNode(t, 0)
	defer cancel() // unexpected call to os.Exit(0) during test
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
			blockchain.lockValidators.RLock()
			exportedState := blockchain.CurrentState().Export()
			blockchain.lockValidators.RUnlock()
			if err := exportedState.Verify(); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
}

func TestBlockchain_GetStateForHeightAndDeleteStateVersions(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t, 100)
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

	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_SendTx(t *testing.T) {
	blockchain, tmCli, _, cancel := initTestNode(t, 0)
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

	blockchain.lockValidators.RLock()
	defer blockchain.lockValidators.RUnlock()
	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestBlockchain_FrozenFunds(t *testing.T) {
	blockchain, tmCli, pv, cancel := initTestNode(t, 0)
	defer cancel()

	targetHeight := uint64(10)
	value := helpers.BipToPip(big.NewInt(1000))
	pubkey := types.BytesToPubkey(pv.Key.PubKey.Bytes()[:])
	blockchain.stateDeliver.RLock()
	blockchain.stateDeliver.Candidates.SubStake(developers.Address, pubkey, 0, big.NewInt(0).Set(value))
	blockchain.stateDeliver.FrozenFunds.AddFund(targetHeight, developers.Address, &pubkey, blockchain.stateDeliver.Candidates.ID(pubkey), 0, big.NewInt(0).Set(value), 0)
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

	blockchain.lockValidators.RLock()
	defer blockchain.lockValidators.RUnlock()
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
	blockchain, tmCli, _, cancel := initTestNode(t, 0)
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
	blockchain.lockValidators.RLock()
	coinID := blockchain.CurrentState().Coins().GetCoinBySymbol(symbol, 0).ID()
	blockchain.lockValidators.RUnlock()
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
					targetHeight = int64(15 + blockchain.updateStakesAndPayRewardsPeriod)
				}
				if h > targetHeight {
					return
				}
			case <-time.After(time.Second * 10):
				t.Fatal("block empty")
			}
		}
	}()
	blockchain.lockValidators.RLock()
	events := blockchain.eventsDB.LoadEvents(135)
	if len(events) == 0 {
		t.Error("no jail")
	}
	candidate := blockchain.CurrentState().Candidates().GetCandidate(types.Pubkey{1})
	if candidate == nil {
		t.Fatal("candidate not found")
	}
	blockchain.lockValidators.RUnlock()

	if candidate.Status == candidates.CandidateStatusOnline {
		t.Fatal("candidate not Offline")
	}

	blockchain.lockValidators.RLock()
	defer blockchain.lockValidators.RUnlock()

	exportedState := blockchain.CurrentState().Export()
	if err := exportedState.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestStopNetworkByHaltBlocks(t *testing.T) {
	blockchain, _, _, cancel := initTestNode(t, 0)
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

func getTestGenesis(pv *privval.FilePV, home string, initialState int64) func() (*types2.GenesisDoc, error) {
	return func() (*types2.GenesisDoc, error) {
		validators, candidates := makeTestValidatorsAndCandidates([]string{string(pv.Key.PubKey.Bytes()[:])}, helpers.BipToPip(big.NewInt(12444011)))

		appState := types.AppState{
			Note:                "",
			Validators:          validators,
			Candidates:          candidates,
			BlockListCandidates: nil,
			DeletedCandidates:   nil,
			Waitlist:            nil,
			Pools: []types.Pool{{
				Coin0:    0,
				Coin1:    1993,
				Reserve0: "350000000000000000000000000000000",
				Reserve1: "1000000000000000000000000000000",
				ID:       1,
				Orders:   nil,
			}},
			NextOrderID: 0,
			Accounts: []types.Account{
				{
					Address: crypto.PubkeyToAddress(getPrivateKey().PublicKey),
					Balance: []types.Balance{
						{
							Coin:  uint64(types.GetBaseCoinID()),
							Value: helpers.BipToPip(big.NewInt(9223372036854775807)).String(),
						},
						{
							Coin:  uint64(types.USDTID),
							Value: "1000000000000000000000000000000",
						},
					},
				},
			},
			Coins: []types.Coin{{
				ID:           1993,
				Name:         "USDT Eth",
				Symbol:       types.StrToCoinSymbol("USDTE"),
				Volume:       "2000000000000000000000000000000",
				Crr:          0,
				Reserve:      "0",
				MaxSupply:    coins.MaxCoinSupply().String(),
				Version:      0,
				OwnerAddress: nil,
				Mintable:     false,
				Burnable:     false,
			}},
			FrozenFunds: nil,
			HaltBlocks:  nil,
			Commission: types.Commission{
				Coin:                    0,
				PayloadByte:             "2000000000000000",
				Send:                    "10000000000000000",
				BuyBancor:               "100000000000000000",
				SellBancor:              "100000000000000000",
				SellAllBancor:           "100000000000000000",
				BuyPoolBase:             "100000000000000000",
				BuyPoolDelta:            "50000000000000000",
				SellPoolBase:            "100000000000000000",
				SellPoolDelta:           "50000000000000000",
				SellAllPoolBase:         "100000000000000000",
				SellAllPoolDelta:        "50000000000000000",
				CreateTicker3:           "1000000000000000000000000",
				CreateTicker4:           "100000000000000000000000",
				CreateTicker5:           "10000000000000000000000",
				CreateTicker6:           "1000000000000000000000",
				CreateTicker7_10:        "100000000000000000000",
				CreateCoin:              "0",
				CreateToken:             "0",
				RecreateCoin:            "10000000000000000000000",
				RecreateToken:           "10000000000000000000000",
				DeclareCandidacy:        "10000000000000000000",
				Delegate:                "200000000000000000",
				Unbond:                  "200000000000000000",
				RedeemCheck:             "30000000000000000",
				SetCandidateOn:          "100000000000000000",
				SetCandidateOff:         "100000000000000000",
				CreateMultisig:          "100000000000000000",
				MultisendBase:           "10000000000000000",
				MultisendDelta:          "5000000000000000",
				EditCandidate:           "10000000000000000000",
				SetHaltBlock:            "1000000000000000000",
				EditTickerOwner:         "10000000000000000000000",
				EditMultisig:            "1000000000000000000",
				EditCandidatePublicKey:  "100000000000000000000000",
				CreateSwapPool:          "1000000000000000000",
				AddLiquidity:            "100000000000000000",
				RemoveLiquidity:         "100000000000000000",
				EditCandidateCommission: "10000000000000000000",
				MintToken:               "100000000000000000",
				BurnToken:               "100000000000000000",
				VoteCommission:          "1000000000000000000",
				VoteUpdate:              "1000000000000000000",
				FailedTx:                "10000000000000000",
				AddLimitOrder:           "10000000000000000",
				RemoveLimitOrder:        "10000000000000000",
				MoveStake:               "10000000000000000",
				LockStake:               "10000000000000000",
				Lock:                    "10000000000000000",
			},
			CommissionVotes: nil,
			UpdateVotes:     nil,
			UsedChecks:      nil,
			MaxGas:          0,
			TotalSlashed:    "0",
			Emission:        "1111",
			PrevReward: types.RewardPrice{
				Time:       0,
				AmountBIP:  "350",
				AmountUSDT: "1",
				Off:        false,
				Reward:     "79",
			},
			Version:  V3,
			Versions: nil,
		}

		appStateJSON, err := tmjson.Marshal(appState)
		if err != nil {
			return nil, err
		}

		genesisDoc := types2.GenesisDoc{
			ChainID:       "minter-test-network",
			InitialHeight: initialState,
			GenesisTime:   time.Now(),
			AppHash:       nil,
			AppState:      json.RawMessage(appStateJSON),
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
