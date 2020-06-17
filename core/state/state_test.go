package state

import (
	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	db "github.com/tendermint/tm-db"
	"log"
	"math/big"
	"math/rand"
	"testing"
)

func TestStateExport(t *testing.T) {
	height := uint64(0)

	state, err := NewState(height, db.NewMemDB(), emptyEvents{}, 1, 1, 0, 1)
	if err != nil {
		log.Panic("Cannot create state")
	}

	coinTest := types.StrToCoinSymbol("TEST")
	coinTest2 := types.StrToCoinSymbol("TEST2")

	state.Coins.Create(
		coinTest,
		"TEST",
		helpers.BipToPip(big.NewInt(1)),
		10,
		helpers.BipToPip(big.NewInt(100)),
		helpers.BipToPip(big.NewInt(100)),
	)

	state.Coins.Create(
		coinTest2,
		"TEST2",
		helpers.BipToPip(big.NewInt(2)),
		50,
		helpers.BipToPip(big.NewInt(200)),
		helpers.BipToPip(big.NewInt(200)),
	)

	privateKey1, _ := crypto.GenerateKey()
	address1 := crypto.PubkeyToAddress(privateKey1.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	address2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	state.Accounts.AddBalance(address1, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1)))
	state.Accounts.AddBalance(address1, coinTest, helpers.BipToPip(big.NewInt(1)))
	state.Accounts.AddBalance(address2, coinTest2, helpers.BipToPip(big.NewInt(2)))

	candidatePubKey1 := [32]byte{}
	rand.Read(candidatePubKey1[:])

	candidatePubKey2 := [32]byte{}
	rand.Read(candidatePubKey2[:])

	state.Candidates.Create(address1, address1, candidatePubKey1, 10)
	state.Candidates.Create(address2, address2, candidatePubKey2, 30)
	state.Validators.Create(candidatePubKey1, helpers.BipToPip(big.NewInt(1)))
	state.FrozenFunds.AddFund(height, address1, candidatePubKey1, coinTest, helpers.BipToPip(big.NewInt(100)))
	state.FrozenFunds.AddFund(height+10, address1, candidatePubKey1, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(3)))
	state.FrozenFunds.AddFund(height+100, address2, candidatePubKey1, coinTest, helpers.BipToPip(big.NewInt(500)))
	state.FrozenFunds.AddFund(height+150, address2, candidatePubKey1, coinTest2, helpers.BipToPip(big.NewInt(1000)))

	newCheck := &check.Check{
		Nonce:    []byte("test nonce"),
		ChainID:  types.CurrentChainID,
		DueBlock: height + 1,
		Coin:     coinTest,
		Value:    helpers.BipToPip(big.NewInt(100)),
		GasCoin:  coinTest2,
	}

	err = newCheck.Sign(privateKey1)
	if err != nil {
		log.Panicf("Cannot sign check: %s", err)
	}

	state.Checks.UseCheck(newCheck)

	_, err = state.Commit()
	if err != nil {
		log.Panicf("Cannot commit state: %s", err)
	}

	newState := state.Export(height)

	if newState.StartHeight != height {
		t.Fatalf("Wrong new state start height. Expected %d, got %d", height, newState.StartHeight)
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
		newStateCoin.Volume != helpers.BipToPip(big.NewInt(1)).String() ||
		newStateCoin.Reserve != helpers.BipToPip(big.NewInt(100)).String() ||
		newStateCoin.MaxSupply != helpers.BipToPip(big.NewInt(100)).String() ||
		newStateCoin.Crr != 10 {
		t.Fatalf("Wrong new state coin data")
	}

	if newStateCoin1.Name != "TEST2" ||
		newStateCoin1.Symbol != coinTest2 ||
		newStateCoin1.Volume != helpers.BipToPip(big.NewInt(2)).String() ||
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

	if funds.Height != height ||
		funds.Address != address1 ||
		funds.Coin != coinTest ||
		*funds.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds.Value != helpers.BipToPip(big.NewInt(100)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if funds1.Height != height+10 ||
		funds1.Address != address1 ||
		funds1.Coin != types.GetBaseCoin() ||
		*funds1.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds1.Value != helpers.BipToPip(big.NewInt(3)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if funds2.Height != height+100 ||
		funds2.Address != address2 ||
		funds2.Coin != coinTest ||
		*funds2.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds2.Value != helpers.BipToPip(big.NewInt(500)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if funds3.Height != height+150 ||
		funds3.Address != address2 ||
		funds3.Coin != coinTest2 ||
		*funds3.CandidateKey != types.Pubkey(candidatePubKey1) ||
		funds3.Value != helpers.BipToPip(big.NewInt(1000)).String() {
		t.Fatalf("Wrong new state frozen fund data")
	}

	if len(newState.UsedChecks) != 1 {
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

	if account1.Balance[0].Coin != coinTest || account1.Balance[0].Value != helpers.BipToPip(big.NewInt(1)).String() {
		t.Fatal("Wrong new state account balance data")
	}

	if account1.Balance[1].Coin != types.GetBaseCoin() || account1.Balance[1].Value != helpers.BipToPip(big.NewInt(1)).String() {
		t.Fatal("Wrong new state account balance data")
	}

	if account2.Balance[0].Coin != coinTest2 || account2.Balance[0].Value != helpers.BipToPip(big.NewInt(2)).String() {
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
}
