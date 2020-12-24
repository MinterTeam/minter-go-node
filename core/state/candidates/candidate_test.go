package candidates

import (
	"encoding/json"
	"fmt"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/state/app"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/checker"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/state/waitlist"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/tendermint/tendermint/crypto/ed25519"
	db "github.com/tendermint/tm-db"
	"math/big"
	"strconv"
	"testing"
)

func TestCandidates_Create_oneCandidate(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}

	if candidates.PubKey(candidate.ID) != [32]byte{4} {
		t.Fatal("candidate error ID or PubKey")
	}
}

func TestCandidates_Commit_createThreeCandidatesWithInitialHeight(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 2)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.Create([20]byte{11}, [20]byte{21}, [20]byte{31}, [32]byte{41}, 10, 0)

	hash, version, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "D7A17D41EAE39D61D3F85BC3311DA1FE306E885FF03024D0173F23E3739E719B" {
		t.Fatalf("hash %X", hash)
	}
	candidates.Create([20]byte{1, 1}, [20]byte{2, 2}, [20]byte{3, 3}, [32]byte{4, 4}, 10, 0)

	hash, version, err = mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 3 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "51B9DC41F65A6BD3F76059E8CA1A9E3CB48750F87A2BD99376E5BA84F53AC12E" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_Commit_changePubKeyAndCheckBlockList(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.Create([20]byte{11}, [20]byte{21}, [20]byte{31}, [32]byte{41}, 10, 0)

	hash, version, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "D7A17D41EAE39D61D3F85BC3311DA1FE306E885FF03024D0173F23E3739E719B" {
		t.Fatalf("hash %X", hash)
	}

	candidates.ChangePubKey([32]byte{4}, [32]byte{5})
	candidates.ChangePubKey([32]byte{41}, [32]byte{6})

	hash, version, err = mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "BB335E1AA631D9540C2CB0AC9C959B556C366B79D39B828B07106CF2DACE5A2D" {
		t.Fatalf("hash %X", hash)
	}

	if !candidates.IsBlockedPubKey([32]byte{4}) {
		t.Fatal("pub_key is not blocked")
	}

	candidates = NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.LoadCandidates()
	candidate := candidates.GetCandidate([32]byte{5})
	if candidate == nil {
		t.Fatal("candidate not found")
	}
	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], types.Pubkey{5}.Bytes())
	var address types.TmAddress
	copy(address[:], pubkey.Address().Bytes())
	if *(candidate.tmAddress) != address {
		t.Fatal("tmAddress not change")
	}
	if candidates.PubKey(candidate.ID) != [32]byte{5} {
		t.Fatal("candidate map ids and pubKeys invalid")
	}

}
func TestCandidates_AddToBlockPubKey(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.AddToBlockPubKey([32]byte{4})

	if !candidates.IsBlockedPubKey([32]byte{4}) {
		t.Fatal("pub_key is not blocked")
	}
}

func TestCandidates_Commit_withStakeAndUpdate(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	hash, version, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	}, []types.Stake{
		{
			Owner:    [20]byte{2},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	})

	hash, version, err = mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "C1659B82F60F0883043A6948C567A31C5B172EB99E5F5F94C346679461A47CE1" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_Commit_edit(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	hash, version, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

	candidates.Edit([32]byte{4}, [20]byte{1, 1}, [20]byte{2, 2}, [20]byte{3, 3})

	hash, version, err = mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "482BE887F2E18DC1BB829BD6AFE8887CE4EC74D4DC485DB1355D78093EAB6B35" {
		t.Fatalf("hash %X", hash)
	}

	if candidates.GetCandidateControl([32]byte{4}) != [20]byte{3, 3} {
		t.Fatal("control address is not change")
	}

	if candidates.GetCandidateOwner([32]byte{4}) != [20]byte{2, 2} {
		t.Fatal("owner address is not change")
	}

}

func TestCandidates_Commit_createOneCandidateWithID(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.CreateWithID([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 1)

	hash, version, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

	id := candidates.ID([32]byte{4})
	if id != 1 {
		t.Fatalf("ID %d", id)
	}
}

func TestCandidates_Commit_Delegate(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	hash, version, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}
	candidates.Delegate([20]byte{1, 1}, [32]byte{4}, 0, big.NewInt(10000000), big.NewInt(10000000))

	hash, version, err = mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "43FE25EB54D52C6516521FB0F951E87359040A9E8DAA23BDC27C6EC5DFBC10EF" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_SetOnlineAndBusSetOffline(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}
	candidates.SetOnline([32]byte{4})

	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}
	if candidate.Status != CandidateStatusOnline {
		t.Fatal("candidate not change status to online")
	}
	candidates.bus.Candidates().SetOffline([32]byte{4})
	if candidate.Status != CandidateStatusOffline {
		t.Fatal("candidate not change status to offline")
	}
}

func TestCandidates_Count(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.Create([20]byte{1, 1}, [20]byte{2, 2}, [20]byte{3, 3}, [32]byte{4, 4}, 20, 0)
	candidates.Create([20]byte{1, 1, 1}, [20]byte{2, 2, 2}, [20]byte{3, 3, 3}, [32]byte{4, 4, 4}, 30, 0)

	hash, version, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "25F7EF5A007B3D8A5FB4DCE32F9DBC28C2AE6848B893986E3055BC3045E8F00F" {
		t.Fatalf("hash %X", hash)
	}

	count := candidates.Count()
	if count != 3 {
		t.Fatalf("coun %d", count)
	}
}

func TestCandidates_GetTotalStake_fromModelAndFromDB(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	wl := waitlist.NewWaitList(b, mutableTree.GetLastImmutable())
	b.SetWaitList(waitlist.NewBus(wl))
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	accs := accounts.NewAccounts(b, mutableTree.GetLastImmutable())

	b.SetAccounts(accounts.NewBus(accs))
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	var stakes []types.Stake
	for i := 0; i < 1010; i++ {
		value := strconv.Itoa(i + 2000)
		stakes = append(stakes, types.Stake{
			Owner:    types.StringToAddress(strconv.Itoa(i)),
			Coin:     0,
			Value:    value,
			BipValue: value,
		})
	}
	candidates.SetStakes([32]byte{4}, stakes, []types.Stake{
		{
			Owner:    [20]byte{2},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
		{
			Owner:    types.StringToAddress("1"),
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	})

	candidates.RecalculateStakes(0)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	totalStake := candidates.GetTotalStake([32]byte{4})
	totalStakeString := totalStake.String()
	if totalStakeString != "2509591" {
		t.Fatalf("total stake %s", totalStakeString)
	}

	candidates = NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.LoadCandidates()
	candidates.GetCandidate([32]byte{4}).totalBipStake = nil
	totalStake = candidates.GetTotalStake([32]byte{4})
	totalStakeString = totalStake.String()
	if totalStakeString != "2509591" {
		t.Fatalf("total stake %s", totalStakeString)
	}
}

func TestCandidates_Export(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.AddToBlockPubKey([32]byte{10})
	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	}, []types.Stake{
		{
			Owner:    [20]byte{2},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	})
	candidates.recalculateStakes(0)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	state := new(types.AppState)
	candidates.Export(state)

	bytes, err := json.Marshal(state.Candidates)
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != "[{\"id\":1,\"reward_address\":\"Mx0200000000000000000000000000000000000000\",\"owner_address\":\"Mx0100000000000000000000000000000000000000\",\"control_address\":\"Mx0300000000000000000000000000000000000000\",\"total_bip_stake\":\"200\",\"public_key\":\"Mp0400000000000000000000000000000000000000000000000000000000000000\",\"commission\":10,\"stakes\":[{\"owner\":\"Mx0100000000000000000000000000000000000000\",\"coin\":0,\"value\":\"100\",\"bip_value\":\"100\"},{\"owner\":\"Mx0200000000000000000000000000000000000000\",\"coin\":0,\"value\":\"100\",\"bip_value\":\"100\"}],\"updates\":[],\"status\":1}]" {
		t.Fatal("not equal JSON", string(bytes))
	}

	bytes, err = json.Marshal(state.BlockListCandidates)
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != "[\"Mp0a00000000000000000000000000000000000000000000000000000000000000\"]" {
		t.Fatal("not equal JSON")
	}
}

func TestCandidates_busGetStakes(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	}, []types.Stake{
		{
			Owner:    [20]byte{2},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	})

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}
	stakes := candidates.bus.Candidates().GetStakes([32]byte{4})
	if len(stakes) != 1 {
		t.Fatalf("stakes count %d", len(stakes))
	}

	if stakes[0].Owner != [20]byte{1} {
		t.Fatal("owner is invalid")
	}
}

func TestCandidates_GetCandidateByTendermintAddress(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}

	candidateByTmAddr := candidates.GetCandidateByTendermintAddress(candidate.GetTmAddress())
	if candidate.ID != candidateByTmAddr.ID {
		t.Fatal("candidate ID != candidateByTmAddr.ID")
	}
}
func TestCandidates_busGetCandidateByTendermintAddress(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	candidates := NewCandidates(bus.NewBus(), mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}

	candidateByTmAddr := candidates.bus.Candidates().GetCandidateByTendermintAddress(candidate.GetTmAddress())
	if candidate.ID != candidateByTmAddr.ID {
		t.Fatal("candidate ID != candidateByTmAddr.ID")
	}
}

func TestCandidates_Punish(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	wl := waitlist.NewWaitList(b, mutableTree.GetLastImmutable())
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	b.SetWaitList(waitlist.NewBus(wl))
	accs := accounts.NewAccounts(b, mutableTree.GetLastImmutable())

	b.SetAccounts(accounts.NewBus(accs))
	appBus := app.NewApp(b, mutableTree.GetLastImmutable())

	b.SetApp(appBus)
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	coinsState := coins.NewCoins(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	coinsState.Create(1,
		types.StrToCoinSymbol("AAA"),
		"AAACOIN",
		helpers.BipToPip(big.NewInt(10)),
		10,
		helpers.BipToPip(big.NewInt(10000)),
		big.NewInt(0).Exp(big.NewInt(10), big.NewInt(10+18), nil),
		nil)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	symbol := coinsState.GetCoinBySymbol(types.StrToCoinSymbol("AAA"), 0)
	if symbol == nil {
		t.Fatal("coin not found")
	}

	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
		{
			Owner:    [20]byte{1},
			Coin:     uint64(symbol.ID()),
			Value:    "100",
			BipValue: "0",
		},
	}, nil)

	candidates.RecalculateStakes(1)
	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}
	candidates.bus.Candidates().Punish(0, candidate.GetTmAddress())

	if candidate.stakesCount != 2 {
		t.Fatalf("stakes count %d", candidate.stakesCount)
	}

	if candidate.stakes[0].Value.String() != "99" {
		t.Fatalf("stakes[0] == %s", candidate.stakes[0].Value.String())
	}
}

type fr struct {
	unbounds []*big.Int
}

func (fr *fr) AddFrozenFund(_ uint64, _ types.Address, _ types.Pubkey, _ uint32, _ types.CoinID, value *big.Int) {
	fr.unbounds = append(fr.unbounds, value)
}
func TestCandidates_PunishByzantineCandidate(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	frozenfunds := &fr{}
	b.SetFrozenFunds(frozenfunds)
	wl := waitlist.NewWaitList(b, mutableTree.GetLastImmutable())

	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	b.SetWaitList(waitlist.NewBus(wl))
	accs := accounts.NewAccounts(b, mutableTree.GetLastImmutable())

	b.SetAccounts(accounts.NewBus(accs))
	appBus := app.NewApp(b, mutableTree.GetLastImmutable())

	b.SetApp(appBus)
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	coinsState := coins.NewCoins(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	coinsState.Create(1,
		types.StrToCoinSymbol("AAA"),
		"AAACOIN",
		helpers.BipToPip(big.NewInt(10)),
		10,
		helpers.BipToPip(big.NewInt(10000)),
		big.NewInt(0).Exp(big.NewInt(10), big.NewInt(10+18), nil),
		nil)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	symbol := coinsState.GetCoinBySymbol(types.StrToCoinSymbol("AAA"), 0)
	if symbol == nil {
		t.Fatal("coin not found")
	}

	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
		{
			Owner:    [20]byte{1},
			Coin:     uint64(symbol.ID()),
			Value:    "100",
			BipValue: "0",
		},
	}, nil)

	candidates.RecalculateStakes(1)

	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}
	candidates.PunishByzantineCandidate(0, candidate.GetTmAddress())

	if candidates.GetStakeValueOfAddress([32]byte{4}, [20]byte{1}, symbol.ID()).String() != "0" {
		t.Error("stake[0] not unbound")
	}
	if candidates.GetStakeValueOfAddress([32]byte{4}, [20]byte{1}, 0).String() != "0" {
		t.Error("stake[1] not unbound")
	}

	if len(frozenfunds.unbounds) != 2 {
		t.Fatalf("count unbounds == %d", len(frozenfunds.unbounds))
	}

	if frozenfunds.unbounds[0].String() != "95" {
		t.Fatalf("frozenfunds.unbounds[0] == %s", frozenfunds.unbounds[0].String())
	}
	if frozenfunds.unbounds[1].String() != "95" {
		t.Fatalf("frozenfunds.unbounds[1] == %s", frozenfunds.unbounds[1].String())
	}
}

func TestCandidates_SubStake(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	}, nil)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	candidates.SubStake([20]byte{1}, [32]byte{4}, 0, big.NewInt(10))
	stake := candidates.GetStakeOfAddress([32]byte{4}, [20]byte{1}, 0)
	if stake == nil {
		t.Fatal("stake not found")
	}

	if stake.Value.String() != "90" {
		t.Fatal("sub stake error")
	}
}

func TestCandidates_IsNewCandidateStakeSufficient(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	}, nil)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !candidates.IsNewCandidateStakeSufficient(0, big.NewInt(1000), 1) {
		t.Log("is not new candidate stake sufficient")
	}
}

func TestCandidates_IsDelegatorStakeSufficient(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	wl := waitlist.NewWaitList(b, mutableTree.GetLastImmutable())

	b.SetWaitList(waitlist.NewBus(wl))
	b.SetChecker(checker.NewChecker(b))
	accs := accounts.NewAccounts(b, mutableTree.GetLastImmutable())

	b.SetAccounts(accounts.NewBus(accs))
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)

	var stakes []types.Stake
	for i := 0; i < 1010; i++ {
		value := strconv.Itoa(i + 2000)
		stakes = append(stakes, types.Stake{
			Owner:    types.StringToAddress(strconv.Itoa(i)),
			Coin:     0,
			Value:    value,
			BipValue: value,
		})
	}
	candidates.SetStakes([32]byte{4}, stakes, []types.Stake{
		{
			Owner:    [20]byte{2},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	})

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    types.StringToAddress("10000"),
			Coin:     0,
			Value:    "10000",
			BipValue: "10000",
		},
	}, nil)

	candidates.recalculateStakes(0)
	_, _, err = mutableTree.Commit(candidates)

	if candidates.IsDelegatorStakeSufficient([20]byte{1}, [32]byte{4}, 0, big.NewInt(10)) {
		t.Fatal("is not delegator stake sufficient")
	}
}
func TestCandidates_IsDelegatorStakeSufficient_false(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	}, nil)

	candidates.recalculateStakes(0)
	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}

	if !candidates.IsDelegatorStakeSufficient([20]byte{1}, [32]byte{4}, 0, big.NewInt(10)) {
		t.Fatal("is delegator stake sufficient")
	}
}

func TestCandidates_GetNewCandidates(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "1000000000000000000000",
			BipValue: "1000000000000000000000",
		},
	}, nil)
	candidates.SetOnline([32]byte{4})

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{5}, 10, 0)
	candidates.SetStakes([32]byte{5}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "1000000000000000000000",
			BipValue: "1000000000000000000000",
		},
	}, nil)
	candidates.SetOnline([32]byte{5})

	candidates.RecalculateStakes(1)

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}
	newCandidates := candidates.GetNewCandidates(2)
	if len(newCandidates) != 2 {
		t.Fatal("error count of new candidates")
	}
}

func TestCandidate_GetFilteredUpdates(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidates.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "100",
			BipValue: "100",
		},
	})

	_, _, err := mutableTree.Commit(candidates)
	if err != nil {
		t.Fatal(err)
	}
	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}

	candidate.filterUpdates()

	if len(candidate.updates) != 1 {
		t.Fatal("updates not merged")
	}

	if candidate.updates[0].Value.String() != "200" {
		t.Fatal("error merge updates")
	}
}

func TestCandidates_CalculateBipValue_RecalculateStakes_GetTotalStake(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	busCoins := coins.NewCoins(b, mutableTree.GetLastImmutable())

	b.SetCoins(coins.NewBus(busCoins))
	candidates := NewCandidates(b, mutableTree.GetLastImmutable())

	coinsState := coins.NewCoins(b, mutableTree.GetLastImmutable())

	candidates.Create([20]byte{1}, [20]byte{1}, [20]byte{1}, [32]byte{1}, 1, 0)
	candidates.SetStakes([32]byte{1}, []types.Stake{
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "27331500301898443574821601",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "26788352158593847436109305",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "23056159980819190092008573",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "11588709101209768903338862",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "10699458018244407488345007",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "10178615801247206484340203",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "9695040709408605598614475",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "9311613733840163086812673",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "8035237015568850680085714",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "7751636678470495902806639",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "7729118857616059555215844",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "7246351659896715230790480",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "5634000000000000000000000",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "5111293424492290525817483",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "4636302767358508700208179",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "4375153667350433703873779",
			BipValue: "0",
		},
		{
			Owner:    types.Address{1},
			Coin:     52,
			Value:    "6468592759016388938414535",
			BipValue: "0",
		},
	}, nil)
	volume, _ := big.NewInt(0).SetString("235304453408778922901904166", 10)
	reserve, _ := big.NewInt(0).SetString("3417127836274022127064945", 10)
	maxSupply, _ := big.NewInt(0).SetString("1000000000000000000000000000000000", 10)
	coinsState.Create(52,
		types.StrToCoinSymbol("ONLY1"),
		"ONLY1",
		volume,
		70,
		reserve,
		maxSupply,
		nil)

	amount, _ := big.NewInt(0).SetString("407000000000000000000000", 10)
	cache := newCoinsCache()

	bipValue := candidates.calculateBipValue(52, amount, false, true, cache)
	if bipValue.Sign() < 0 {
		t.Fatalf("%s", bipValue.String())
	}
	bipValue = candidates.calculateBipValue(52, amount, false, true, cache)
	if bipValue.Sign() < 0 {
		t.Fatalf("%s", bipValue.String())
	}

	candidates.RecalculateStakes(0)
	totalStake := candidates.GetTotalStake([32]byte{1})
	if totalStake.String() != "2435386873327199834002556" {
		t.Fatalf("total stake %s", totalStake.String())
	}
}
