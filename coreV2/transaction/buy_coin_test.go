package transaction

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"log"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/pkg/errors"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	db "github.com/tendermint/tm-db"
)

var (
	rnd             = rand.New(rand.NewSource(time.Now().Unix()))
	commissionPrice = commission.Price{
		Coin:                    types.GetBaseCoinID(),
		PayloadByte:             helpers.StringToBigInt("2000000000000000"),
		Send:                    helpers.StringToBigInt("10000000000000000"),
		BuyBancor:               helpers.StringToBigInt("100000000000000000"),
		SellBancor:              helpers.StringToBigInt("100000000000000000"),
		SellAllBancor:           helpers.StringToBigInt("100000000000000000"),
		BuyPoolBase:             helpers.StringToBigInt("100000000000000000"),
		BuyPoolDelta:            helpers.StringToBigInt("50000000000000000"),
		SellPoolBase:            helpers.StringToBigInt("100000000000000000"),
		SellPoolDelta:           helpers.StringToBigInt("50000000000000000"),
		SellAllPoolBase:         helpers.StringToBigInt("100000000000000000"),
		SellAllPoolDelta:        helpers.StringToBigInt("50000000000000000"),
		CreateTicker3:           helpers.StringToBigInt("1000000000000000000000000"),
		CreateTicker4:           helpers.StringToBigInt("100000000000000000000000"),
		CreateTicker5:           helpers.StringToBigInt("10000000000000000000000"),
		CreateTicker6:           helpers.StringToBigInt("1000000000000000000000"),
		CreateTicker7to10:       helpers.StringToBigInt("100000000000000000000"),
		CreateCoin:              helpers.StringToBigInt("0"),
		CreateToken:             helpers.StringToBigInt("0"),
		RecreateCoin:            helpers.StringToBigInt("10000000000000000000000"),
		RecreateToken:           helpers.StringToBigInt("10000000000000000000000"),
		DeclareCandidacy:        helpers.StringToBigInt("10000000000000000000"),
		Delegate:                helpers.StringToBigInt("200000000000000000"),
		Unbond:                  helpers.StringToBigInt("200000000000000000"),
		RedeemCheck:             helpers.StringToBigInt("30000000000000000"),
		SetCandidateOn:          helpers.StringToBigInt("100000000000000000"),
		SetCandidateOff:         helpers.StringToBigInt("100000000000000000"),
		CreateMultisig:          helpers.StringToBigInt("100000000000000000"),
		MultisendBase:           helpers.StringToBigInt("10000000000000000"),
		MultisendDelta:          helpers.StringToBigInt("5000000000000000"),
		EditCandidate:           helpers.StringToBigInt("10000000000000000000"),
		SetHaltBlock:            helpers.StringToBigInt("1000000000000000000"),
		EditTickerOwner:         helpers.StringToBigInt("10000000000000000000000"),
		EditMultisig:            helpers.StringToBigInt("1000000000000000000"),
		EditCandidatePublicKey:  helpers.StringToBigInt("100000000000000000000000"),
		CreateSwapPool:          helpers.StringToBigInt("1000000000000000000"),
		AddLiquidity:            helpers.StringToBigInt("100000000000000000"),
		RemoveLiquidity:         helpers.StringToBigInt("100000000000000000"),
		EditCandidateCommission: helpers.StringToBigInt("10000000000000000000"),
		BurnToken:               helpers.StringToBigInt("100000000000000000"),
		MintToken:               helpers.StringToBigInt("100000000000000000"),
		VoteCommission:          helpers.StringToBigInt("1000000000000000000"),
		VoteUpdate:              helpers.StringToBigInt("1000000000000000000"),
		More:                    nil,
	}
)

func getState() *state.State {
	s, err := state.NewState(0, db.NewMemDB(), nil, 1, 1, 0)
	if err != nil {
		panic(err)
	}

	s.Validators.Create(types.Pubkey{}, big.NewInt(1))
	s.Candidates.Create(types.Address{}, types.Address{}, types.Address{}, types.Pubkey{}, 10, 0, 0)
	s.Commission.SetNewCommissions(commissionPrice.Encode())
	return s
}

func getTestCoinSymbol() types.CoinSymbol {
	var coin types.CoinSymbol
	copy(coin[:], "TEST")

	return coin
}

func createTestCoin(stateDB *state.State) types.CoinID {
	volume := helpers.BipToPip(big.NewInt(100000))
	reserve := helpers.BipToPip(big.NewInt(100000))

	id := stateDB.App.GetNextCoinID()
	stateDB.Coins.Create(id, getTestCoinSymbol(), "TEST COIN", volume, 10, reserve,
		big.NewInt(0).Mul(volume, big.NewInt(10)), nil)
	stateDB.App.SetCoinsCount(id.Uint32())
	stateDB.Accounts.AddBalance(types.Address{}, id, volume)

	return id
}

func createTestCoinWithOwner(stateDB *state.State, owner types.Address) types.CoinID {
	volume := helpers.BipToPip(big.NewInt(100000))
	reserve := helpers.BipToPip(big.NewInt(100000))

	id := stateDB.App.GetNextCoinID()

	stateDB.Coins.Create(id, getTestCoinSymbol(), "TEST COIN", volume, 10, reserve,
		big.NewInt(0).Mul(volume, big.NewInt(10)), &owner)
	stateDB.App.SetCoinsCount(id.Uint32())
	stateDB.Accounts.AddBalance(types.Address{}, id, volume)

	_, _, err := stateDB.Tree().Commit(stateDB.Coins)
	if err != nil {
		log.Fatalf("failed to commit coins: %s", err)
	}

	return id
}

func checkState(cState *state.State) error {
	if _, err := cState.Commit(); err != nil {
		return err
	}

	exportedState := cState.Export()
	if err := exportedState.Verify(); err != nil {
		return errors.Wrapf(err, "error export version %d", cState.Tree().Version())
	}

	reloadExport := cState.ReloadFromDiskAndExport()
	if err := reloadExport.Verify(); err != nil {
		return errors.Wrapf(err, "error reload export version %d", cState.Tree().Version())
	}

	if bytes.Compare(
		getStateSha256Hash(exportedState),
		getStateSha256Hash(reloadExport),
	) != 0 {
		return errors.New("current state and real from disk state not equal")
	}

	return nil
}

func getStateSha256Hash(a types.AppState) []byte {
	bytes, err := tmjson.Marshal(a)
	if err != nil {
		panic(err)
	}

	h := sha256.New()
	h.Write(bytes)
	return h.Sum(nil)
}

func TestBuyCoinTxBaseToCustom(t *testing.T) {
	t.Parallel()
	cState := getState()

	coinToBuyID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	initBalance := helpers.BipToPip(big.NewInt(1000000))
	cState.Accounts.AddBalance(addr, coin, initBalance)
	cState.Coins.AddVolume(coin, initBalance)

	toBuy := helpers.BipToPip(big.NewInt(10))
	maxValToSell, _ := big.NewInt(0).SetString("159374246010000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          coinToBuyID,
		ValueToBuy:         toBuy,
		CoinToSell:         coin,
		MaximumValueToSell: maxValToSell,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999899854987997899747979", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	testBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	if testBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", getTestCoinSymbol(), toBuy, testBalance)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxInsufficientFunds(t *testing.T) {
	t.Parallel()
	cState := getState()

	coinToBuyID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))

	toBuy := helpers.BipToPip(big.NewInt(10))
	maxValToSell, _ := big.NewInt(0).SetString("159374246010000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          coinToBuyID,
		ValueToBuy:         toBuy,
		CoinToSell:         coin,
		MaximumValueToSell: maxValToSell,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:    1,
		GasPrice: 1,

		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxEqualCoins(t *testing.T) {
	t.Parallel()
	cState := getState()

	coinID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  coinID,
		ValueToBuy: big.NewInt(0),
		CoinToSell: coinID,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.CrossConvert {
		t.Fatalf("Response code is not %d. Error %s", code.CrossConvert, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxNotExistsBuyCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  cState.App.GetNextCoinID(),
		ValueToBuy: big.NewInt(0),
		CoinToSell: types.GetBaseCoinID(),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxNotExistsSellCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  types.GetBaseCoinID(),
		ValueToBuy: big.NewInt(0),
		CoinToSell: cState.App.GetNextCoinID(),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxNotExistsGasCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	coinToSellID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  types.GetBaseCoinID(),
		ValueToBuy: big.NewInt(0),
		CoinToSell: coinToSellID,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       cState.App.GetNextCoinID(),
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxNotGasCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	coinToSellID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	initBal := helpers.BipToPip(big.NewInt(1000))
	cState.Accounts.AddBalance(addr, coinToSellID, initBal)
	cState.Coins.AddVolume(coinToSellID, initBal)

	data := BuyCoinData{
		CoinToBuy:          types.GetBaseCoinID(),
		ValueToBuy:         big.NewInt(1),
		CoinToSell:         coinToSellID,
		MaximumValueToSell: big.NewInt(10100004545002879),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coinToSellID,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxCustomToBase(t *testing.T) {
	t.Parallel()
	cState := getState()

	coinToSellID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	initBal := helpers.BipToPip(big.NewInt(10000000))
	cState.Accounts.AddBalance(addr, coinToSellID, initBal)
	cState.Coins.AddVolume(coinToSellID, initBal)

	toBuy := helpers.BipToPip(big.NewInt(10))
	maxValToSell, _ := big.NewInt(0).SetString("159374246010000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          types.GetBaseCoinID(),
		ValueToBuy:         toBuy,
		CoinToSell:         coinToSellID,
		MaximumValueToSell: maxValToSell,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coinToSellID,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("9999897985363348906133282", 10)
	balance := cState.Accounts.GetBalance(addr, coinToSellID)
	if balance.Cmp(targetBalance) != 0 {
		t.Errorf("Target %s balance is not correct. Expected %s, got %s", coinToSellID.String(), targetBalance, balance)
	}

	baseBalance := cState.Accounts.GetBalance(addr, types.GetBaseCoinID())
	if baseBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", types.GetBaseCoinID(), toBuy, baseBalance)
	}

	coinData := cState.Coins.GetCoin(coinToSellID)

	targetReserve, _ := big.NewInt(0).SetString("99989900000000000000000", 10)
	if coinData.Reserve().Cmp(targetReserve) != 0 {
		t.Fatalf("Target %s reserve is not correct. Expected %s, got %s", coinToSellID.String(), targetBalance, coinData.Reserve())
	}

	targetVolume, _ := big.NewInt(0).SetString("10099897985363348906133282", 10)
	if coinData.Volume().Cmp(targetVolume) != 0 {
		t.Fatalf("Target %s volume is not correct. Expected %s, got %s", coinToSellID.String(), targetVolume, coinData.Volume())
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState := getState()

	coinToSellID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	initBal := helpers.BipToPip(big.NewInt(10000000))
	cState.Accounts.AddBalance(addr, coinToSellID, initBal)
	cState.Coins.AddVolume(coinToSellID, initBal)

	toBuy := helpers.BipToPip(big.NewInt(99000))
	maxValToSell, _ := big.NewInt(0).SetString("3727394550309723608045536", 10)
	data := BuyCoinData{
		CoinToBuy:          types.GetBaseCoinID(),
		ValueToBuy:         toBuy,
		CoinToSell:         coinToSellID,
		MaximumValueToSell: maxValToSell,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coinToSellID,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.CoinReserveUnderflow {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveUnderflow, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxBaseToCustomBaseCommission(t *testing.T) {
	t.Parallel()
	// sell_coin: MNT
	// buy_coin: TEST
	// gas_coin: MNT

	coinToSell := types.GetBaseCoinID()
	coinToBuy := types.StrToCoinSymbol("TEST")
	gasCoin := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToBuyID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()

	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)
	cState.Coins.AddVolume(coinToSell, initialBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuyID, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins + commission
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	commissions := cState.Commission.GetCommissions()
	commissionInBaseCoin := tx.MulGasPrice(tx.Price(commissions))
	if !commissions.Coin.IsBaseCoin() {
		commissionInBaseCoin = cState.Swap.GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
	}
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, commissionInBaseCoin)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, toBuy))
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToBuyID)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Add(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, toBuy))
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Add(estimatedSupply, toBuy)
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxCustomToBaseBaseCommission(t *testing.T) {
	t.Parallel()
	// sell_coin: TEST
	// buy_coin: MNT
	// gas_coin: MNT

	coinToSell := types.StrToCoinSymbol("TEST")
	coinToBuy := types.GetBaseCoinID()
	gasCoin := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance, _ := big.NewInt(0).SetString("100000000000000000", 10)
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToSell)

	privateKey, addr := getAccount()

	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume.Add(initialVolume, initialBalance)

	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)
	cState.Coins.AddVolume(gasCoin, initialGasBalance)

	tx := createBuyCoinTx(coinToSellID, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, formula.CalculateSaleAmount(initialVolume, initialReserve, crr, toBuy))
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToSellID)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Sub(estimatedReserve, toBuy)
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume, initialReserve, crr, toBuy))
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxCustomToCustomBaseCommission(t *testing.T) {
	t.Parallel()
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: MNT

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST12")
	gasCoin := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance, _ := big.NewInt(0).SetString("100000000000000000", 10)

	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	coinToBuyID, initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume1.Add(initialVolume1, initialBalance)

	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)

	tx := createBuyCoinTx(coinToSellID, coinToBuyID, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	toSellBaseCoin := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)
	toSell := formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, toSellBaseCoin)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToSellID)

	estimatedReserve := big.NewInt(0).Set(initialReserve1)
	estimatedReserve.Sub(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume1)
	estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)))
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxBaseToCustomCustomCommission(t *testing.T) {
	t.Parallel()
	// sell_coin: MNT
	// buy_coin: TEST
	// gas_coin: TEST

	coinToSell := types.GetBaseCoinID()
	coinToBuy := types.StrToCoinSymbol("TEST")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToBuyID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)
	cState.Accounts.AddBalance(addr, coinToBuyID, initialGasBalance)
	cState.Coins.AddVolume(coinToBuyID, initialGasBalance)
	initialVolume.Add(initialVolume, initialGasBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuyID, coinToBuyID, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	commissions := cState.Commission.GetCommissions()
	commissionInBaseCoin := tx.MulGasPrice(tx.Price(commissions))
	if !commissions.Coin.IsBaseCoin() {
		commissionInBaseCoin = cState.Swap.GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
	}
	commission := formula.CalculateSaleAmount(initialVolume, initialReserve, crr, commissionInBaseCoin)

	{
		// check received coins + commission
		buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
		estimatedBuyCoinBalance := big.NewInt(0).Set(toBuy)
		estimatedBuyCoinBalance.Add(estimatedBuyCoinBalance, initialGasBalance)
		// toReserve := formula.CalculatePurchaseAmount(big.NewInt(0).Sub(initialVolume, commission), big.NewInt(0).Sub(initialReserve, commissionInBaseCoin), crr, toBuy)

		estimatedBuyCoinBalance.Sub(estimatedBuyCoinBalance, commission)
		if buyCoinBalance.Cmp(estimatedBuyCoinBalance) != 0 {
			t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyCoinBalance.String(), buyCoinBalance.String())
		}
	}

	{
		// check sold coins
		sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
		estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
		// amountCom := formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, commission)
		amount := formula.CalculatePurchaseAmount(big.NewInt(0).Sub(initialVolume, commission), big.NewInt(0).Sub(initialReserve, commissionInBaseCoin), crr, toBuy)
		// amount.Add(amount, amountCom)
		estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, amount)
		if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
			t.Fatalf("Sell coin balance want %s, got %s", estimatedSellCoinBalance, sellCoinBalance)
		}

	}
	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToBuyID)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Add(estimatedReserve, formula.CalculatePurchaseAmount(big.NewInt(0).Sub(initialVolume, commission), big.NewInt(0).Sub(initialReserve, commissionInBaseCoin), crr, toBuy))
	estimatedReserve.Sub(estimatedReserve, commissionInBaseCoin)
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Add(estimatedSupply, toBuy)
	estimatedSupply.Sub(estimatedSupply, commission)
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply. Expected %s, got %s", estimatedSupply.String(), coinData.Volume().String())
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxCustomToBaseCustomCommission(t *testing.T) {
	t.Parallel()
	// sell_coin: TEST
	// buy_coin: MNT
	// gas_coin: TEST

	coinToSell := types.StrToCoinSymbol("TEST")
	coinToBuy := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToSell)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume.Add(initialVolume, initialBalance)

	tx := createBuyCoinTx(coinToSellID, coinToBuy, coinToSellID, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	commissions := cState.Commission.GetCommissions()
	commissionInBaseCoin := tx.MulGasPrice(tx.Price(commissions))
	if !commissions.Coin.IsBaseCoin() {
		commissionInBaseCoin = cState.Swap.GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
	}
	coms := formula.CalculateSaleAmount(initialVolume, initialReserve, crr, commissionInBaseCoin)
	shouldGive := formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume, coms), big.NewInt(0).Sub(initialReserve, commissionInBaseCoin), crr, toBuy)
	editedSupply := big.NewInt(0).Add(shouldGive, coms)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, editedSupply)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Errorf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSellID)

		estimatedReserve := big.NewInt(0).Set(initialReserve)
		estimatedReserve.Sub(estimatedReserve, toBuy)
		estimatedReserve.Sub(estimatedReserve, commissionInBaseCoin)
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume)
		estimatedSupply.Sub(estimatedSupply, editedSupply)
		// estimatedSupply.Sub(estimatedSupply, commission)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply. Expected %s, got %s", estimatedSupply.String(), coinData.Volume().String())
		}
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxCustomToCustomCustom1Commission(t *testing.T) {
	t.Parallel()
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: TEST1

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST2")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	coinToBuyID, initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume1.Add(initialVolume1, initialBalance)

	tx := createBuyCoinTx(coinToSellID, coinToBuyID, coinToSellID, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	toSellBaseCoin := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)
	commissions := cState.Commission.GetCommissions()
	commissionInBaseCoin := tx.MulGasPrice(tx.Price(commissions))
	if !commissions.Coin.IsBaseCoin() {
		commissionInBaseCoin = cState.Swap.GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
	}
	com := formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, commissionInBaseCoin)
	toSell := formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume1, com), big.NewInt(0).Sub(initialReserve1, commissionInBaseCoin), crr1, toSellBaseCoin)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, big.NewInt(0).Add(toSell, com))
	// commission := formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume1, toSell), big.NewInt(0).Sub(initialReserve1, toSellBaseCoin), crr1, tx.CommissionInBaseCoin())
	// estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, commission)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Errorf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSellID)

		estimatedReserve := big.NewInt(0).Set(initialReserve1)
		estimatedReserve.Sub(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
		estimatedReserve.Sub(estimatedReserve, commissionInBaseCoin)
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume1)
		spend := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)
		estimatedSupply.Sub(estimatedSupply, com)
		estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume1, com), big.NewInt(0).Sub(initialReserve1, commissionInBaseCoin), crr1, spend))
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply want %s, got %s", estimatedSupply, coinData.Volume())
		}
	}

	{
		coinData := cState.Coins.GetCoin(coinToBuyID)

		estimatedReserve := big.NewInt(0).Set(initialReserve2)
		estimatedReserve.Add(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume2)
		estimatedSupply.Add(estimatedSupply, toBuy)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxCustomToCustomCustom2Commission(t *testing.T) {
	t.Parallel()
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: TEST2

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST2")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	coinToBuyID, initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume1.Add(initialVolume1, initialBalance)

	cState.Accounts.AddBalance(addr, coinToBuyID, initialGasBalance)
	cState.Coins.AddVolume(coinToBuyID, initialGasBalance)
	initialVolume2.Add(initialVolume2, initialGasBalance)

	tx := createBuyCoinTx(coinToSellID, coinToBuyID, coinToBuyID, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)

	commissions := cState.Commission.GetCommissions()
	commissionInBaseCoin := tx.MulGasPrice(tx.Price(commissions))
	if !commissions.Coin.IsBaseCoin() {
		commissionInBaseCoin = cState.Swap.GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
	}

	commission := formula.CalculateSaleAmount(initialVolume2, initialReserve2, crr2, commissionInBaseCoin)
	// shouldGive := formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume2, coms), big.NewInt(0).Sub(initialReserve2, commissionInBaseCoin), crr, toBuy)
	buyCoinBalance.Add(buyCoinBalance, commission)

	buyCoinBalance.Sub(buyCoinBalance, initialGasBalance)
	// commission := formula.CalculateSaleAmount(big.NewInt(0).Add(initialVolume2, toBuy), big.NewInt(0).Add(initialReserve2, toReserve), crr2, commissionInBaseCoin)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", toBuy.String(), buyCoinBalance.String())
	}

	{
		// check received coins
		// toReserve := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)

	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	toSellBaseCoin := formula.CalculatePurchaseAmount(big.NewInt(0).Sub(initialVolume2, commission), big.NewInt(0).Sub(initialReserve2, commissionInBaseCoin), crr2, toBuy)
	toSell := formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, toSellBaseCoin)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	calculatePurchaseAmount := formula.CalculatePurchaseAmount(big.NewInt(0).Sub(initialVolume2, commission), big.NewInt(0).Sub(initialReserve2, commissionInBaseCoin), crr2, toBuy)
	{
		coinData := cState.Coins.GetCoin(coinToSellID)

		estimatedReserve := big.NewInt(0).Set(initialReserve1)
		estimatedReserve.Sub(estimatedReserve, calculatePurchaseAmount)
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume1)
		estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, calculatePurchaseAmount))
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	{
		coinData := cState.Coins.GetCoin(coinToBuyID)

		estimatedReserve := big.NewInt(0).Set(initialReserve2)
		estimatedReserve.Add(estimatedReserve, calculatePurchaseAmount)
		estimatedReserve.Sub(estimatedReserve, commissionInBaseCoin)
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume2)
		estimatedSupply.Add(estimatedSupply, toBuy)
		estimatedSupply.Sub(estimatedSupply, commission)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxToCoinSupplyOverflow(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID, sellCoinID := createTestCoin(cState), types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, sellCoinID, helpers.BipToPip(big.NewInt(5000000)))

	tx := createBuyCoinTx(sellCoinID, coinToBuyID, sellCoinID, helpers.BipToPip(big.NewInt(1000001)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CoinSupplyOverflow {
		t.Fatalf("Response code is not %d. Error %s", code.CoinSupplyOverflow, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxToMaximumValueToSellReached(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID, sellCoinID := createTestCoin(cState), types.GetBaseCoinID()

	valueToBuy := big.NewInt(2e18)
	// cState.Accounts.AddBalance(addr, sellCoinID, valueToBuy)
	cState.Coins.AddVolume(sellCoinID, valueToBuy)

	data := BuyCoinData{
		CoinToBuy:          coinToBuyID,
		ValueToBuy:         valueToBuy,
		CoinToSell:         sellCoinID,
		MaximumValueToSell: big.NewInt(1e18),
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx := &Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       sellCoinID,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
		decodedData:   data,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.MaximumValueToSellReached {
		t.Fatalf("Response code is not %d. Error %d %s", code.MaximumValueToSellReached, response.Code, response.Log)
	}

	cState.Accounts.AddBalance(addr, coinToBuyID, helpers.BipToPip(big.NewInt(100000)))
	cState.Coins.AddVolume(coinToBuyID, helpers.BipToPip(big.NewInt(100000)))

	data.CoinToBuy = sellCoinID
	data.CoinToSell = coinToBuyID
	data.MaximumValueToSell = big.NewInt(1)
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx.Data = encodedData
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.MaximumValueToSellReached {
		t.Fatalf("Response code is not %d. Error %s", code.MaximumValueToSellReached, response.Log)
	}

	cState.Coins.Create(
		cState.App.GetNextCoinID(),
		types.StrToCoinSymbol("TEST9"),
		"TEST COIN",
		helpers.BipToPip(big.NewInt(100000)),
		10,
		helpers.BipToPip(big.NewInt(100000)),
		helpers.BipToPip(big.NewInt(1000000)),
		nil,
	)

	coinToSellID := cState.App.GetNextCoinID()
	cState.App.SetCoinsCount(coinToSellID.Uint32())

	cState.Accounts.AddBalance(types.Address{0}, coinToSellID, helpers.BipToPip(big.NewInt(100000-2)))
	cState.Accounts.AddBalance(addr, coinToSellID, helpers.BipToPip(big.NewInt(2)))

	data.CoinToBuy = coinToBuyID
	data.CoinToSell = coinToSellID
	data.MaximumValueToSell = big.NewInt(1)
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx.Data = encodedData
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.MaximumValueToSellReached {
		t.Fatalf("Response code is not %d. Error %s", code.MaximumValueToSellReached, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	data.MaximumValueToSell = big.NewInt(1000090908227683847)
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx.Data = encodedData
	tx.GasCoin = data.CoinToSell
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.MaximumValueToSellReached {
		t.Fatalf("Response code is not %d. Error %d %s", code.MaximumValueToSellReached, response.Code, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestBuyCoinTxToCoinReserveNotSufficient(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID := createTestCoin(cState)

	cState.Coins.Create(
		cState.App.GetNextCoinID(),
		types.StrToCoinSymbol("TEST9"),
		"TEST COIN",
		helpers.BipToPip(big.NewInt(5000000)),
		10,
		helpers.BipToPip(big.NewInt(100000)),
		helpers.BipToPip(big.NewInt(10000000)),
		nil,
	)

	coinToSellID := cState.App.GetNextCoinID()
	cState.App.SetCoinsCount(coinToSellID.Uint32())

	cState.Accounts.AddBalance(addr, coinToSellID, helpers.BipToPip(big.NewInt(5000000)))

	tx := createBuyCoinTx(coinToSellID, coinToBuyID, coinToBuyID, helpers.BipToPip(big.NewInt(10000)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CoinReserveNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	// gas coin == coin to buy

	cState.Coins.SubReserve(tx.GasCoin, helpers.BipToPip(big.NewInt(100000)))

	tx = createBuyCoinTx(coinToSellID, coinToBuyID, coinToBuyID, helpers.BipToPip(big.NewInt(1)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %d %s", code.CommissionCoinNotSufficient, response.Code, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	// gas coin == coin to sell

	tx = createBuyCoinTx(coinToBuyID, coinToSellID, coinToBuyID, helpers.BipToPip(big.NewInt(1)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %d %s", code.CommissionCoinNotSufficient, response.Code, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	// gas coin == coin to buy
	// sell coin == base coin

	tx = createBuyCoinTx(types.GetBaseCoinID(), coinToBuyID, coinToBuyID, big.NewInt(1), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	tx.GasPrice = 5000
	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %d %s", code.CommissionCoinNotSufficient, response.Code, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func createBuyCoinTx(sellCoin, buyCoin, gasCoin types.CoinID, valueToBuy *big.Int, nonce uint64) *Transaction {
	maxValToSell, _ := big.NewInt(0).SetString("100000000000000000000000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          buyCoin,
		ValueToBuy:         valueToBuy,
		CoinToSell:         sellCoin,
		MaximumValueToSell: maxValToSell,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		panic(err)
	}

	return &Transaction{
		Nonce:         nonce,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       gasCoin,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,

		decodedData: data,
	}
}

func getAccount() (*ecdsa.PrivateKey, types.Address) {
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	return privateKey, addr
}

func createTestCoinWithSymbol(stateDB *state.State, symbol types.CoinSymbol) (types.CoinID, *big.Int, *big.Int, uint32) {
	volume := helpers.BipToPip(big.NewInt(100000))
	reserve := helpers.BipToPip(big.NewInt(100000))
	volume.Mul(volume, big.NewInt(int64(rnd.Intn(9))+1))
	reserve.Mul(reserve, big.NewInt(int64(rnd.Intn(9))+1))

	crr := uint32(10 + rnd.Intn(90))

	id := stateDB.App.GetNextCoinID()
	stateDB.Coins.Create(id, symbol, "TEST COIN", volume, crr, reserve, big.NewInt(0).Mul(volume, big.NewInt(10)), nil)
	stateDB.App.SetCoinsCount(id.Uint32())
	stateDB.Accounts.AddBalance(types.Address{}, id, volume)

	return id, big.NewInt(0).Set(volume), big.NewInt(0).Set(reserve), crr
}
