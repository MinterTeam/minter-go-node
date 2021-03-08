package state

import (
	"github.com/MinterTeam/minter-go-node/coreV2/check"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	db "github.com/tendermint/tm-db"
	"log"
	"math/big"
	"math/rand"
	"testing"
)

func TestStateExport(t *testing.T) {
	t.Parallel()
	height := uint64(0)

	state, err := NewState(height, db.NewMemDB(), &eventsdb.MockEvents{}, 1, 2, 0)
	if err != nil {
		log.Panic("Cannot create state")
	}
	commissionPrice := commission.Price{
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
	state.Commission.SetNewCommissions(commissionPrice.Encode())

	coinTest := types.StrToCoinSymbol("TEST")
	coinTest2 := types.StrToCoinSymbol("TEST2")

	coinTestID := state.App.GetNextCoinID()
	coinTest2ID := coinTestID + 1

	state.Coins.Create(
		coinTestID,
		coinTest,
		"TEST",
		helpers.BipToPip(big.NewInt(701)),
		10,
		helpers.BipToPip(big.NewInt(100)),
		helpers.BipToPip(big.NewInt(100)),
		nil,
	)

	state.Coins.Create(
		coinTest2ID,
		coinTest2,
		"TEST2",
		helpers.BipToPip(big.NewInt(1202)),
		50,
		helpers.BipToPip(big.NewInt(200)),
		helpers.BipToPip(big.NewInt(200)),
		nil,
	)

	state.App.SetCoinsCount(coinTest2ID.Uint32())

	privateKey1, _ := crypto.GenerateKey()
	address1 := crypto.PubkeyToAddress(privateKey1.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	address2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	state.Accounts.AddBalance(address1, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(1)))
	state.Accounts.AddBalance(address1, coinTestID, helpers.BipToPip(big.NewInt(100)))
	state.Accounts.AddBalance(address2, coinTest2ID, helpers.BipToPip(big.NewInt(200)))

	candidatePubKey1 := types.Pubkey{}
	rand.Read(candidatePubKey1[:])

	candidatePubKey2 := types.Pubkey{}
	rand.Read(candidatePubKey2[:])

	state.Candidates.Create(address1, address1, address1, candidatePubKey1, 10, 0)
	state.Candidates.Create(address2, address2, address2, candidatePubKey2, 30, 0)
	state.Validators.Create(candidatePubKey1, helpers.BipToPip(big.NewInt(1)))
	state.FrozenFunds.AddFund(height+110, address1, &candidatePubKey1, state.Candidates.ID(candidatePubKey1), coinTestID, helpers.BipToPip(big.NewInt(100)), nil)
	state.FrozenFunds.AddFund(height+120, address1, &candidatePubKey1, state.Candidates.ID(candidatePubKey1), types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(3)), nil)
	state.FrozenFunds.AddFund(height+140, address2, &candidatePubKey1, state.Candidates.ID(candidatePubKey1), coinTestID, helpers.BipToPip(big.NewInt(500)), nil)
	state.FrozenFunds.AddFund(height+150, address2, &candidatePubKey1, state.Candidates.ID(candidatePubKey1), coinTest2ID, helpers.BipToPip(big.NewInt(1000)), nil)

	newCheck0 := &check.Check{
		Nonce:    []byte("test nonce"),
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coinTestID,
		Value:    helpers.BipToPip(big.NewInt(100)),
		GasCoin:  coinTest2ID,
	}

	err = newCheck0.Sign(privateKey1)
	if err != nil {
		log.Panicf("Cannot sign check: %s", err)
	}

	state.Checks.UseCheck(newCheck0)

	newCheck := &check.Check{
		Nonce:    []byte("test nonce 1"),
		ChainID:  types.CurrentChainID,
		DueBlock: 999999,
		Coin:     coinTestID,
		Value:    helpers.BipToPip(big.NewInt(100)),
		GasCoin:  coinTest2ID,
	}

	err = newCheck.Sign(privateKey1)
	if err != nil {
		log.Panicf("Cannot sign check: %s", err)
	}

	state.Checks.UseCheck(newCheck)

	state.Halts.AddHaltBlock(height, types.Pubkey{0})
	state.Halts.AddHaltBlock(height+1, types.Pubkey{1})
	state.Halts.AddHaltBlock(height+2, types.Pubkey{2})

	wlAddr1 := types.StringToAddress("1")
	wlAddr2 := types.StringToAddress("2")

	state.Waitlist.AddWaitList(wlAddr1, candidatePubKey1, coinTestID, big.NewInt(1e18))
	state.Waitlist.AddWaitList(wlAddr2, candidatePubKey2, coinTest2ID, big.NewInt(2e18))

	_, err = state.Commit()
	if err != nil {
		log.Panicf("Cannot commit state: %s", err)
	}

	newState := state.Export()
	if err := newState.Verify(); err != nil {
		t.Error(err)
	}

	if newState.MaxGas != state.App.GetMaxGas() {
		t.Fatalf("Wrong new state max gas. Expected %d, got %d", state.App.GetMaxGas(), newState.MaxGas)
	}

	if newState.TotalSlashed != state.App.GetTotalSlashed().String() {
		t.Fatalf("Wrong new state total slashes. Expected %d, got %s", state.App.GetMaxGas(), newState.TotalSlashed)
	}

	if len(newState.Coins) != 2 {
		t.Fatalf("Wrong new state coins size. Expected %d, got %d", 2, len(newState.Coins))
	}

	newStateCoin := newState.Coins[0]
	newStateCoin1 := newState.Coins[1]

	if newStateCoin.Name != "TEST" ||
		newStateCoin.Symbol != coinTest ||
		newStateCoin.Volume != helpers.BipToPip(big.NewInt(701)).String() ||
		newStateCoin.Reserve != helpers.BipToPip(big.NewInt(100)).String() ||
		newStateCoin.MaxSupply != helpers.BipToPip(big.NewInt(100)).String() ||
		newStateCoin.Crr != 10 {
		t.Fatalf("Wrong new state coin data")
	}

	if newStateCoin1.Name != "TEST2" ||
		newStateCoin1.Symbol != coinTest2 ||
		newStateCoin1.Volume != helpers.BipToPip(big.NewInt(1202)).String() ||
		newStateCoin1.Reserve != helpers.BipToPip(big.NewInt(200)).String() ||
		newStateCoin1.MaxSupply != helpers.BipToPip(big.NewInt(200)).String() ||
		newStateCoin1.Crr != 50 {
		t.Fatalf("Wrong new state coin data")
	}

	if len(newState.FrozenFunds) != 4 {
		t.Fatalf("Wrong new state frozen funds size. Expected %d, got %d", 4, len(newState.FrozenFunds))
	}

	funds := newState.FrozenFunds[0]
	funds1 := newState.FrozenFunds[1]
	funds2 := newState.FrozenFunds[2]
	funds3 := newState.FrozenFunds[3]

	if funds.Height != height+110 ||
		funds.Address != address1 ||
		funds.Coin != uint64(coinTestID) ||
		*funds.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds.Value != helpers.BipToPip(big.NewInt(100)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if funds1.Height != height+120 ||
		funds1.Address != address1 ||
		funds1.Coin != uint64(types.GetBaseCoinID()) ||
		*funds1.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds1.Value != helpers.BipToPip(big.NewInt(3)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if funds2.Height != height+140 ||
		funds2.Address != address2 ||
		funds2.Coin != uint64(coinTestID) ||
		*funds2.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds2.Value != helpers.BipToPip(big.NewInt(500)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if funds3.Height != height+150 ||
		funds3.Address != address2 ||
		funds3.Coin != uint64(coinTest2ID) ||
		*funds3.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds3.Value != helpers.BipToPip(big.NewInt(1000)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if len(newState.UsedChecks) != 2 {
		t.Fatalf("Wrong new state used checks size. Expected %d, got %d", 1, len(newState.UsedChecks))
	}

	if string("Mx"+newState.UsedChecks[0]) != newCheck.Hash().String() {
		t.Fatal("Wrong new state used check data")
	}

	if len(newState.Accounts) != 2 {
		t.Fatalf("Wrong new state accounts size. Expected %d, got %d", 2, len(newState.Accounts))
	}

	var account1, account2 types.Account

	if newState.Accounts[0].Address == address1 {
		account1 = newState.Accounts[0]
		account2 = newState.Accounts[1]
	}

	if newState.Accounts[0].Address == address2 {
		account1 = newState.Accounts[1]
		account2 = newState.Accounts[0]
	}

	if account1.Address != address1 || account2.Address != address2 {
		t.Fatal("Wrong new state account addresses")
	}

	if len(account1.Balance) != 2 || len(account2.Balance) != 1 {
		t.Fatal("Wrong new state account balances size")
	}

	if account1.Balance[1].Coin != uint64(coinTestID) || account1.Balance[1].Value != helpers.BipToPip(big.NewInt(100)).String() {
		t.Fatal("Wrong new state account balance data")
	}

	if account1.Balance[0].Coin != uint64(types.GetBaseCoinID()) || account1.Balance[0].Value != helpers.BipToPip(big.NewInt(1)).String() {
		t.Fatal("Wrong new state account balance data")
	}

	if account2.Balance[0].Coin != uint64(coinTest2ID) || account2.Balance[0].Value != helpers.BipToPip(big.NewInt(200)).String() {
		t.Fatal("Wrong new state account balance data")
	}

	if len(newState.Validators) != 1 {
		t.Fatal("Wrong new state validators size")
	}

	if newState.Validators[0].PubKey != candidatePubKey1 || newState.Validators[0].TotalBipStake != helpers.BipToPip(big.NewInt(1)).String() {
		t.Fatal("Wrong new state validator data")
	}

	if len(newState.Candidates) != 2 {
		t.Fatal("Wrong new state candidates size")
	}

	newStateCandidate1 := newState.Candidates[1]
	newStateCandidate2 := newState.Candidates[0]

	if newStateCandidate1.PubKey != candidatePubKey1 ||
		newStateCandidate1.OwnerAddress != address1 ||
		newStateCandidate1.RewardAddress != address1 ||
		newStateCandidate1.Commission != 10 {
		t.Fatal("Wrong new state candidate data")
	}

	if newStateCandidate2.PubKey != candidatePubKey2 ||
		newStateCandidate2.OwnerAddress != address2 ||
		newStateCandidate2.RewardAddress != address2 ||
		newStateCandidate2.Commission != 30 {
		t.Fatal("Wrong new state candidate data")
	}

	if len(newState.HaltBlocks) != 3 {
		t.Fatalf("Invalid amount of halts: %d. Expected 3", len(newState.HaltBlocks))
	}

	pubkey := types.Pubkey{0}
	if newState.HaltBlocks[0].Height != height || !newState.HaltBlocks[0].CandidateKey.Equals(pubkey) {
		t.Fatal("Wrong new state halt blocks")
	}

	pubkey = types.Pubkey{1}
	if newState.HaltBlocks[1].Height != height+1 || !newState.HaltBlocks[1].CandidateKey.Equals(pubkey) {
		t.Fatal("Wrong new state halt blocks")
	}

	pubkey = types.Pubkey{2}
	if newState.HaltBlocks[2].Height != height+2 || !newState.HaltBlocks[2].CandidateKey.Equals(pubkey) {
		t.Fatal("Wrong new state halt blocks")
	}

	if len(newState.Waitlist) != 2 {
		t.Fatalf("Invalid amount of waitlist: %d. Expected 2", len(newState.Waitlist))
	}

	if newState.Waitlist[0].Coin != uint64(coinTest2ID) || newState.Waitlist[0].Value != big.NewInt(2e18).String() || newState.Waitlist[0].Owner.Compare(wlAddr2) != 0 {
		t.Fatal("Invalid waitlist data")
	}

	if newState.Waitlist[1].Coin != uint64(coinTestID) || newState.Waitlist[1].Value != big.NewInt(1e18).String() || newState.Waitlist[1].Owner.Compare(wlAddr1) != 0 {
		t.Fatal("Invalid waitlist data")
	}
}
