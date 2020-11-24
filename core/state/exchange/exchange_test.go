package exchange

import (
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestSwap_Add_createFirstLiquidity(t *testing.T) {
	swap := NewSwap(nil)
	tableTests := []struct {
		initialXVolume *big.Int
		initialYVolume *big.Int
		initialStake   *big.Float
	}{
		{
			initialXVolume: big.NewInt(2),
			initialYVolume: big.NewInt(200),
			initialStake:   big.NewFloat(20),
		},
		{
			initialXVolume: big.NewInt(2),
			initialYVolume: big.NewInt(800),
			initialStake:   big.NewFloat(40),
		},
	}
	for i, test := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := swap.Add(types.Address{1}, types.CoinID(i), test.initialXVolume, 999, test.initialYVolume)
			if err != nil {
				t.Fatal(err)
			}
			xVolume, yVolume, stake, err := swap.Balance(types.Address{1}, types.CoinID(i), 999)
			if err != nil {
				t.Fatal(err)
			}
			if xVolume.Cmp(test.initialXVolume) != 0 {
				t.Errorf("xVolume want %s, got %s", test.initialXVolume.String(), xVolume.String())
			}
			if yVolume.Cmp(test.initialYVolume) != 0 {
				t.Errorf("yVolume want %s, got %s", test.initialYVolume.String(), yVolume.String())
			}
			if stake.Cmp(test.initialStake) != 0 {
				t.Errorf("stake want %s, got %s", test.initialStake.String(), stake.String())
			}
		})
	}
}

func TestSwap_Add(t *testing.T) {
	swap := NewSwap(nil)
	tableTests := []struct {
		xCoin          types.CoinID
		yCoin          types.CoinID
		initialXVolume *big.Int
		initialYVolume *big.Int
		addXVolume     *big.Int
		maxAddYVolume  *big.Int
		supplyXVolume  *big.Int
		supplyYVolume  *big.Int
		supplyStakes   *big.Float
	}{
		{
			xCoin:          0,
			yCoin:          1,
			initialXVolume: big.NewInt(2),
			initialYVolume: big.NewInt(200),
			addXVolume:     big.NewInt(2),
			maxAddYVolume:  big.NewInt(200),
			supplyXVolume:  big.NewInt(4),
			supplyYVolume:  big.NewInt(400),
			supplyStakes:   big.NewFloat(40),
		},
		{
			xCoin:          1,
			yCoin:          2,
			initialXVolume: big.NewInt(2),
			initialYVolume: big.NewInt(200),
			addXVolume:     big.NewInt(2),
			maxAddYVolume:  big.NewInt(300),
			supplyXVolume:  big.NewInt(4),
			supplyYVolume:  big.NewInt(400),
			supplyStakes:   big.NewFloat(40),
		},
		{
			xCoin:          999,
			yCoin:          998,
			initialXVolume: big.NewInt(2),
			initialYVolume: big.NewInt(200),
			addXVolume:     big.NewInt(2),
			maxAddYVolume:  big.NewInt(200),
			supplyXVolume:  big.NewInt(4),
			supplyYVolume:  big.NewInt(400),
			supplyStakes:   big.NewFloat(40),
		},
		{
			xCoin:          998,
			yCoin:          997,
			initialXVolume: big.NewInt(2),
			initialYVolume: big.NewInt(200),
			addXVolume:     big.NewInt(2),
			maxAddYVolume:  big.NewInt(300),
			supplyXVolume:  big.NewInt(4),
			supplyYVolume:  big.NewInt(400),
			supplyStakes:   big.NewFloat(40),
		},
	}
	for i, test := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := swap.Add(types.Address{1}, test.xCoin, test.initialXVolume, test.yCoin, test.initialYVolume)
			if err != nil {
				t.Fatal(err)
			}
			err = swap.Add(types.Address{1}, test.xCoin, test.addXVolume, test.yCoin, test.maxAddYVolume)
			if err != nil {
				t.Fatal(err)
			}
			xVolume, yVolume, stake, err := swap.Balance(types.Address{1}, test.xCoin, test.yCoin)
			if err != nil {
				t.Fatal(err)
			}
			if xVolume.Cmp(test.supplyXVolume) != 0 {
				t.Errorf("xVolume want %s, got %s", test.supplyXVolume.String(), xVolume.String())
			}
			if yVolume.Cmp(test.supplyYVolume) != 0 {
				t.Errorf("yVolume want %s, got %s", test.supplyYVolume.String(), yVolume.String())
			}
			if stake.Cmp(test.supplyStakes) != 0 {
				t.Errorf("stake want %s, got %s", test.supplyStakes.String(), stake.String())
			}
		})
	}
}

func TestSwap_Remove(t *testing.T) {
	swap := NewSwap(nil)
	tableTests := []struct {
		xCoin             types.CoinID
		yCoin             types.CoinID
		initialXVolume    *big.Int
		initialYVolume    *big.Int
		wantRemoveXVolume *big.Int
		wantRemoveYVolume *big.Int
		removeStake       *big.Float
		supplyXVolume     *big.Int
		supplyYVolume     *big.Int
		supplyStakes      *big.Float
		err               error
	}{
		{
			xCoin:             0,
			yCoin:             1,
			initialXVolume:    big.NewInt(2),
			initialYVolume:    big.NewInt(200),
			wantRemoveXVolume: big.NewInt(2),
			wantRemoveYVolume: big.NewInt(200),
			removeStake:       big.NewFloat(20),
			supplyXVolume:     nil,
			supplyYVolume:     nil,
			supplyStakes:      nil,
			err:               errors.New("provider's stake not found"),
		},
		{
			xCoin:             1,
			yCoin:             2,
			initialXVolume:    big.NewInt(2),
			initialYVolume:    big.NewInt(200),
			wantRemoveXVolume: big.NewInt(1),
			wantRemoveYVolume: big.NewInt(100),
			removeStake:       big.NewFloat(10),
			supplyXVolume:     big.NewInt(1),
			supplyYVolume:     big.NewInt(100),
			supplyStakes:      big.NewFloat(10),
		},
		{
			xCoin:             2,
			yCoin:             3,
			initialXVolume:    big.NewInt(2),
			initialYVolume:    big.NewInt(200),
			wantRemoveXVolume: nil,
			wantRemoveYVolume: nil,
			removeStake:       big.NewFloat(30),
			supplyXVolume:     big.NewInt(1),
			supplyYVolume:     big.NewInt(100),
			supplyStakes:      big.NewFloat(10),
			err:               errors.New("provider's stake less"),
		},
	}
	for i, test := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := swap.Add(types.Address{1}, test.xCoin, test.initialXVolume, test.yCoin, test.initialYVolume)
			if err != nil {
				t.Fatal(err)
			}
			removedXVolume, removedYVolume, err := swap.Remove(types.Address{1}, test.xCoin, test.yCoin, test.removeStake)
			if test.err != nil {
				if err != nil {
					if err.Error() == test.err.Error() {
						return
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if removedXVolume.Cmp(test.wantRemoveXVolume) != 0 {
				t.Errorf("xVolume remove want %s, got %s", test.wantRemoveXVolume.String(), removedXVolume.String())
			}
			if removedYVolume.Cmp(test.wantRemoveYVolume) != 0 {
				t.Errorf("yVolume remove want %s, got %s", test.wantRemoveYVolume.String(), removedYVolume.String())
			}
			xVolume, yVolume, stake, err := swap.Balance(types.Address{1}, test.xCoin, test.yCoin)
			if test.err != nil {
				if err != nil {
					if err.Error() == test.err.Error() {
						return
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if xVolume.Cmp(test.supplyXVolume) != 0 {
				t.Errorf("xVolume want %s, got %s", test.supplyXVolume.String(), xVolume.String())
			}
			if yVolume.Cmp(test.supplyYVolume) != 0 {
				t.Errorf("yVolume want %s, got %s", test.supplyYVolume.String(), yVolume.String())
			}
			if stake.Cmp(test.supplyStakes) != 0 {
				t.Errorf("stake want %s, got %s", test.supplyStakes.String(), stake.String())
			}
		})
	}
}

func TestSwap_Commit(t *testing.T) {
	memDB := db.NewMemDB()
	mutableTree1, err := tree.NewMutableTree(0, memDB, 1024)
	if err != nil {
		t.Fatal(err)
	}
	swap := NewSwap(mutableTree1.GetImmutable())

	initialXVolume := big.NewInt(2)
	initialYVolume := big.NewInt(200)
	initialStake := big.NewFloat(20)

	err = swap.Add(types.Address{1}, 0, initialXVolume, 1, initialYVolume)
	if err != nil {
		t.Fatal(err)
	}
	err = swap.Commit(mutableTree1.MutableTree())
	if err != nil {
		t.Fatal(err)
	}

	_, version, err := mutableTree1.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	mutableTree2, err := tree.NewMutableTree(uint64(version), memDB, 1024)
	if err != nil {
		t.Fatal(err)
	}
	swap = NewSwap(mutableTree2.GetImmutable())
	xVolume, yVolume, stake, err := swap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if xVolume.Cmp(initialXVolume) != 0 {
		t.Errorf("xVolume want %s, got %s", initialXVolume.String(), xVolume.String())
	}
	if yVolume.Cmp(initialYVolume) != 0 {
		t.Errorf("yVolume want %s, got %s", initialYVolume.String(), yVolume.String())
	}
	if stake.Cmp(initialStake) != 0 {
		t.Errorf("stake want %s, got %s", initialStake.String(), stake.String())
	}
}
