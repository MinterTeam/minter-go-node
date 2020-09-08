package candidates

import (
	"encoding/json"
	"fmt"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/checker"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/state/waitlist"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"strconv"
	"testing"
)

func TestCandidates_Commit_createOneCandidate(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

	if !candidates.Exists([32]byte{4}) {
		t.Fatal("candidate not found by pub_key")
	}
}

func TestCandidates_Commit_createThreeCandidates(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)
	candidates.Create([20]byte{11}, [20]byte{21}, [20]byte{31}, [32]byte{41}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "D7A17D41EAE39D61D3F85BC3311DA1FE306E885FF03024D0173F23E3739E719B" {
		t.Fatalf("hash %X", hash)
	}

	candidates.Create([20]byte{1, 1}, [20]byte{2, 2}, [20]byte{3, 3}, [32]byte{4, 4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "01E34A08A0CF18403B8C3708FA773A4D0B152635F321085CE7B68F04FD520A9A" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_Commit_changePubKeyAndCheckBlockList(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)
	candidates.Create([20]byte{11}, [20]byte{21}, [20]byte{31}, [32]byte{41}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
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

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "C0DC396CF17399CF3E05EAFFD29D94A99698633C9160D46C469D5F6575DC66E0" {
		t.Fatalf("hash %X", hash)
	}

	if !candidates.IsBlockedPubKey([32]byte{4}) {
		t.Fatal("pub_key is not blocked")
	}
}
func TestCandidates_Commit_addBlockList(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.AddToBlockPubKey([32]byte{4})

	if !candidates.IsBlockedPubKey([32]byte{4}) {
		t.Fatal("pub_key is not blocked")
	}
}

func TestCandidates_Commit_withStakeAndUpdate(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
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
	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "2D206158AA79C3BDAA019C61FEAD47BB9B6170C445EE7B36E935AC954765E99F" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_Commit_withStakesMoreMaxDelegatorsPerCandidate(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

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
	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "51022ED08FD976D0305B3BD8CB90C0139CDC4970CD9548237DF358ECD54BA6D1" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_Commit_edit(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
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

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
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
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.CreateWithID([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 1)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
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
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates, err := NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
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

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
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

func TestCandidates_Commit_setOnline(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates, err := NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

	candidates.SetOnline([32]byte{4})

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "B7CDFAE03E151CA2CD105295E101D4BD00F64CD55D2D8E1AD853853C623BEC23" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_Commit_setOffline(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates, err := NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

	candidates.SetOffline([32]byte{4})

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "17D06535CC123FDF2DA9B97D272E683EC143CEEC73C143D151D0311388E82CBC" {
		t.Fatalf("hash %X", hash)
	}
}

func TestCandidates_Count(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	candidates, err := NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)
	candidates.Create([20]byte{1, 1}, [20]byte{2, 2}, [20]byte{3, 3}, [32]byte{4, 4}, 20)
	candidates.Create([20]byte{1, 1, 1}, [20]byte{2, 2, 2}, [20]byte{3, 3, 3}, [32]byte{4, 4, 4}, 30)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
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
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	b := bus.NewBus()
	wl, err := waitlist.NewWaitList(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	b.SetWaitList(waitlist.NewBus(wl))
	accs, err := accounts.NewAccounts(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}
	b.SetAccounts(accounts.NewBus(accs))
	b.SetChecker(checker.NewChecker(b))
	candidates, err := NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FCF3853839873D3EC344016C04A5E75166F51063745670DF5D561C060E7F45A1" {
		t.Fatalf("hash %X", hash)
	}

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

	candidates.RecalculateStakes(1)

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	hash, version, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "5DDA783086104DCC15B5825F0C2BD559EA813A3024AEB8E8A3D336A24676887B" {
		t.Fatalf("hash %X", hash)
	}

	totalStake := candidates.GetTotalStake([32]byte{4})
	totalStakeString := totalStake.String()
	if totalStakeString != "2509500" {
		t.Fatalf("total stake %s", totalStakeString)
	}

	candidates, err = NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}
	candidates.LoadCandidates()
	totalStake = candidates.GetTotalStake([32]byte{4})
	totalStakeString = totalStake.String()
	if totalStakeString != "2509500" {
		t.Fatalf("total stake %s", totalStakeString)
	}
}

func TestCandidates_GetTotalStake_forCustomCoins(t *testing.T) {

	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	b := bus.NewBus()
	wl, err := waitlist.NewWaitList(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	b.SetWaitList(waitlist.NewBus(wl))
	accs, err := accounts.NewAccounts(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}
	b.SetAccounts(accounts.NewBus(accs))
	b.SetChecker(checker.NewChecker(b))
	candidates, err := NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	coinsState, err := coins.NewCoins(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)
	coinsState.Create(1,
		types.StrToCoinSymbol("AAA"),
		"AAACOIN",
		helpers.BipToPip(big.NewInt(10)),
		10,
		helpers.BipToPip(big.NewInt(10000)),
		big.NewInt(0).Exp(big.NewInt(10), big.NewInt(10+18), nil),
		nil)

	err = coinsState.Commit()
	if err != nil {
		t.Fatal(err)
	}

	symbol := coinsState.GetCoinBySymbol(types.StrToCoinSymbol("AAA"), 0)
	if symbol == nil {
		t.Fatal("coin not found")
	}

	var stakes []types.Stake
	for i := 0; i < 50; i++ {
		value := strconv.Itoa(i + 2000)
		stakes = append(stakes, types.Stake{
			Owner:    types.StringToAddress(strconv.Itoa(i)),
			Coin:     symbol.ID(),
			Value:    value,
			BipValue: "0",
		})
	}
	candidates.SetStakes([32]byte{4}, stakes, nil)

	candidates.RecalculateStakes(1)

	candidates.LoadCandidates()
	totalStake := candidates.GetTotalStake([32]byte{4})
	totalStakeString := totalStake.String()
	if totalStakeString != "9802420350703877401368" {
		t.Fatalf("total stake %s", totalStakeString)
	}
}

func TestCandidates_Export(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)
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

	err = candidates.Commit()
	if err != nil {
		t.Fatal(err)
	}

	state := new(types.AppState)
	candidates.Export(state)

	bytes, err := json.Marshal(state.Candidates)
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != "[{\"id\":1,\"reward_address\":\"Mx0200000000000000000000000000000000000000\",\"owner_address\":\"Mx0100000000000000000000000000000000000000\",\"control_address\":\"Mx0300000000000000000000000000000000000000\",\"total_bip_stake\":\"0\",\"public_key\":\"Mp0400000000000000000000000000000000000000000000000000000000000000\",\"commission\":10,\"stakes\":[{\"owner\":\"Mx0100000000000000000000000000000000000000\",\"coin\":0,\"value\":\"100\",\"bip_value\":\"100\"}],\"updates\":[{\"owner\":\"Mx0200000000000000000000000000000000000000\",\"coin\":0,\"value\":\"100\",\"bip_value\":\"100\"}],\"status\":1}]" {
		t.Fatal("not equal JSON")
	}

	bytes, err = json.Marshal(state.BlockListCandidates)
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != "[\"Mp0a00000000000000000000000000000000000000000000000000000000000000\"]" {
		t.Fatal("not equal JSON")
	}
}

func TestCandidates_bus(t *testing.T) {
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)
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

	err = candidates.Commit()
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
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	candidates, err := NewCandidates(bus.NewBus(), mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidates.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10)

	candidate := candidates.GetCandidate([32]byte{4})
	if candidate == nil {
		t.Fatal("candidate not found")
	}

	candidateByTmAddr := candidates.GetCandidateByTendermintAddress(*candidate.tmAddress)
	if candidate.ID != candidateByTmAddr.ID {
		t.Fatal("candidate ID != candidateByTmAddr.ID")
	}
}
