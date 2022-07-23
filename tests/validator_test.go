package tests

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"testing"
)

func TestEqualValidatorCmpBalances(t *testing.T) {
	var addresses []types.Address
	var users []*User
	for i := 0; i < 10; i++ {
		owner := CreateAddress()
		users = append(users, owner)
		addresses = append(addresses, owner.address)
	}

	helper := NewHelper(DefaultAppState(addresses...))

	var initialBalances = make(map[types.Address]map[types.CoinID]*big.Int)
	for i, address := range addresses {
		for _, addr := range []types.Address{address, getCustomAddress(i), getRewardAddress(i)} {
			initialBalances[addr] = make(map[types.CoinID]*big.Int)
			for _, balance := range helper.app.CurrentState().Accounts().GetBalances(addr) {
				initialBalances[addr][balance.Coin.ID] = balance.Value
			}
		}
	}

	t.Run("rewards", func(t *testing.T) {
		testBlocks(t, helper, 50)

		for i, address := range addresses {
			for _, addr := range []types.Address{address, getCustomAddress(i), getRewardAddress(i)} {
				for coin, balance := range initialBalances[addr] {
					if balance.String() != helper.app.CurrentState().Accounts().GetBalance(address, coin).String() {
						t.Error("account"+address.String()+"balance is diff", balance)
					}
				}
			}
		}
	})

	t.Run("delegate", func(t *testing.T) {
		delegatedCoin := types.CoinID(1)
		{
			var txs []transaction.Transaction
			for i, delegator := range users {
				txs = append(txs, helper.CreateTx(delegator.privateKey, transaction.DelegateDataV260{
					PubKey: getValidatorAddress(i),
					Coin:   delegatedCoin,
					Value:  initialBalances[delegator.address][delegatedCoin],
				}, types.USDTID))
			}

			_, results := helper.NextBlock(txs...)
			for _, resp := range results {
				if resp.Code != code.OK {
					t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
				}
			}
		}
		testBlocks(t, helper, 50)

		var equalRewards = make(map[int]*big.Int)
		for i, address := range addresses {
			candidate := helper.app.CurrentState().Candidates().GetCandidate(getValidatorAddress(i))
			if candidate.Status != 2 {
				t.Error("status is", candidate.Status)
			}
			delegateCoin1Balance := helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(i), address, delegatedCoin)
			if delegateCoin1Balance.String() != initialBalances[address][delegatedCoin].String() {
				t.Errorf("validator "+getValidatorAddress(i).String()+" address "+address.String()+"delegate Coin1 Volume wanted %s, got %s", initialBalances[address][delegatedCoin].String(), delegateCoin1Balance)
			}
			for j, addr := range []types.Address{address, getCustomAddress(i), getRewardAddress(i)} {
				rewardAddressBalance := helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(i), addr, types.GetBaseCoinID())
				if equalRewards[j] == nil {
					equalRewards[j] = rewardAddressBalance
				}
				if rewardAddressBalance.String() != equalRewards[j].String() {
					t.Error("validator "+getValidatorAddress(i).String()+" address "+address.String()+" diff reward", i, rewardAddressBalance.String())
				}
			}
			balance := helper.app.CurrentState().Accounts().GetBalance(address, delegatedCoin).String()
			if balance != "0" {
				t.Error(helper.app.CurrentState().Coins().GetCoin(delegatedCoin).Symbol().String(), address.String(), "balance is", balance)
			}
		}
	})

}

func TestX3Rewards(t *testing.T) {
	var addresses []types.Address
	var users []*User
	for i := 0; i < 20; i++ {
		owner := CreateAddress()
		users = append(users, owner)
		addresses = append(addresses, owner.address)
	}

	helper := NewHelper(DefaultAppState(addresses...))
	if helper.app.GetVersionName(helper.app.Height()) != minter.V340 {
		t.Fatalf("version want %s, got %s", minter.V340, helper.app.GetVersionName(helper.app.Height()))
	}
	if r1, r3 := helper.app.CurrentState().App().Reward(); r1.Cmp(r3) != 0 {
		t.Fatal("rewards diff", r1, r3)
	}

	{
		delegatedCoin := types.CoinID(1)
		var initialBalances = make(map[types.Address]*big.Int)
		for _, address := range addresses {
			initialBalances[address] = helper.app.CurrentState().Accounts().GetBalance(address, delegatedCoin)
		}
		{
			//var txs []transaction.Transaction
			for i, delegator := range users {
				_, results := helper.NextBlock(helper.CreateTx(delegator.privateKey, transaction.DelegateDataV260{
					PubKey: getValidatorAddress(i),
					Coin:   delegatedCoin,
					Value:  initialBalances[delegator.address],
				}, types.USDTID))
				for _, resp := range results {
					if resp.Code != code.OK {
						t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
					}
				}
			}
		}
	}

	for h := uint64(1); h%helper.app.UpdateStakesAndPayRewardsPeriod() != 0; h, _ = helper.NextBlock() {
	}

	for h := uint64(1); h%helper.app.UpdateStakesAndPayRewardsPeriod() != 0; h, _ = helper.NextBlock() {
	}

	{
		h := helper.app.Height()
		if len(helper.app.GetEventsDB().LoadEvents(uint32(h))) < 5*len(addresses) {
			t.Fatalf("reward events want more than %d, got %d", 5*len(addresses), len(helper.app.GetEventsDB().LoadEvents(uint32(h))))
		}
	}

	{
		lastJ := 0
		for j, user := range users {
			var txs []transaction.Transaction
			for i, delegator := range users {
				if lastJ <= i {
					delegatedBalance := big.NewInt(0).Sub(helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(i), delegator.address, types.GetBaseCoinID()), initialBIPStake)
					txs = append(txs, helper.CreateTx(delegator.privateKey, transaction.UnbondDataV3{
						PubKey: getValidatorAddress(i),
						Coin:   types.GetBaseCoinID(),
						Value:  delegatedBalance,
					}, types.USDTID))
					lastJ = j + 1
				}
			}
			_, results := helper.NextBlock(txs...)
			for _, resp := range results {
				if resp.Code != code.OK {
					t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
				}
			}

			{
				h, results := helper.NextBlock(helper.CreateTx(user.privateKey, transaction.LockStakeData{}, types.USDTID))
				for _, resp := range results {
					if resp.Code != code.OK {
						t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
					}
				}

				for ; h%helper.app.UpdateStakesAndPayRewardsPeriod() != 0; h, _ = helper.NextBlock() {
				}

				if lastJ > 0 && lastJ < len(users) {
					quo := big.NewFloat(0).Quo(
						big.NewFloat(0).SetInt(big.NewInt(0).Sub(
							helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(lastJ-1), users[lastJ-1].address, types.GetBaseCoinID()),
							initialBIPStake)),
						big.NewFloat(0).SetInt(big.NewInt(0).Sub(
							helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(lastJ), users[lastJ].address, types.GetBaseCoinID()),
							initialBIPStake))).Text('f', 1)
					if quo != "3.0" {
						t.Errorf("rewards want x3, got %s", quo)
					}
				}
				if lastJ > 1 && lastJ <= len(users) {
					quo := big.NewFloat(0).Quo(
						big.NewFloat(0).SetInt(big.NewInt(0).Sub(
							helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(lastJ-2), users[lastJ-2].address, types.GetBaseCoinID()),
							initialBIPStake)),
						big.NewFloat(0).SetInt(big.NewInt(0).Sub(
							helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(lastJ-1), users[lastJ-1].address, types.GetBaseCoinID()),
							initialBIPStake))).Text('f', 1)
					if quo != "2.0" {
						t.Errorf("diff stakes want x2, got %s", quo)
					}
				}

			}
		}
	}

}

func getCommissionFromTags(event abciTypes.Event) *big.Int {
	for _, attr := range event.Attributes {
		if string(attr.Key) == "tx.commission_in_base_coin" {
			return helpers.StringToBigInt(string(attr.Value))
		}
	}

	return big.NewInt(0)
}
