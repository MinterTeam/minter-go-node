package tests

import (
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
)

func TestOrder_sell_part(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           1,
		Name:         "Test 1",
		Symbol:       types.StrToCoinBaseSymbol("TEST1"),
		Volume:       "10010000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	}, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10015000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  true,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
		Coins:             []types.CoinID{1, 2},
		ValueToSell:       helpers.StringToBigInt("10000000000000000000000"),
		MinimumValueToBuy: helpers.StringToBigInt("1"),
	})

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 1)
		if balance.String() == "0" {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "> 0", balance)
		}
	}
}

func TestOrder_sell_part_plus_com(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           1,
		Name:         "Test 1",
		Symbol:       types.StrToCoinBaseSymbol("TEST1"),
		Volume:       "10010000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	}, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10025000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller
	state.NextOrderID = 3
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  true,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	}, types.Pool{
		Coin0:    0,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       2,
		Orders:   []types.Order{},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
		Coins:             []types.CoinID{1, 2},
		ValueToSell:       helpers.StringToBigInt("10000000000000000000000"),
		MinimumValueToBuy: helpers.StringToBigInt("1"),
	}, 2)

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 1)
		if balance.String() == "0" {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "> 0", balance)
		}
	}
}
func TestOrder_sell_part_plus_com_pool(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10015000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller

	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    0,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  true,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
		Coins:             []types.CoinID{0, 2},
		ValueToSell:       helpers.StringToBigInt("100000000000000000000000"),
		MinimumValueToBuy: helpers.StringToBigInt("1"),
	}, 2)

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}

	for _, event := range response.Events {
		for _, tag := range event.Attributes {
			t.Log(tag.String())
		}
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 0)
		if balance.Cmp(helpers.StringToBigInt("14985000000000000000000")) == -1 {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "more 14985000000000000000000", balance)
		}
	}
}

func TestOrder_sell_full(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           1,
		Name:         "Test 1",
		Symbol:       types.StrToCoinBaseSymbol("TEST1"),
		Volume:       "10010000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	}, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10015000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  true,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
		Coins:             []types.CoinID{1, 2},
		ValueToSell:       helpers.StringToBigInt("22330916069244653273088"),
		MinimumValueToBuy: helpers.StringToBigInt("1"),
	})

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 1)
		if balance.Cmp(helpers.StringToBigInt("14985000000000000000000")) == -1 {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "more 14985000000000000000000", balance)
		}
	}
}

func TestOrder_sell_more(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           1,
		Name:         "Test 1",
		Symbol:       types.StrToCoinBaseSymbol("TEST1"),
		Volume:       "10010000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	}, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10015000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  true,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
		Coins:             []types.CoinID{1, 2},
		ValueToSell:       helpers.StringToBigInt("22330916069244653273092"), // +4
		MinimumValueToBuy: helpers.StringToBigInt("1"),
	})

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 1)
		if balance.Cmp(helpers.StringToBigInt("14985000000000000000000")) == -1 {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "more 14985000000000000000000", balance)
		}
	}
}

func TestOrder_sell_more_a_lot(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           1,
		Name:         "Test 1",
		Symbol:       types.StrToCoinBaseSymbol("TEST1"),
		Volume:       "10010000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	}, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10015000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  true,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
		Coins:             []types.CoinID{1, 2},
		ValueToSell:       helpers.StringToBigInt("10000000000000000000000000"),
		MinimumValueToBuy: helpers.StringToBigInt("1"),
	})

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}
	for _, event := range response.Events {
		for _, tag := range event.Attributes {
			t.Log(tag.String())
		}
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 1)
		if balance.Cmp(helpers.StringToBigInt("14985000000000000000000")) == -1 {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "more 14985000000000000000000", balance)
		}
	}
}

func TestOrder_buy_more_a_more(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           1,
		Name:         "Test 1",
		Symbol:       types.StrToCoinBaseSymbol("TEST1"),
		Volume:       "10010000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	}, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10015000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  true,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeBuySwapPool, transaction.BuySwapPoolDataV230{
		Coins:              []types.CoinID{1, 2},
		MaximumValueToSell: helpers.StringToBigInt("22330916069244653273092"),
		ValueToBuy:         app.CurrentState().Swap().GetSwapper(1, 2).CalculateBuyForSellWithOrders(helpers.StringToBigInt("22330916069244653273092")), // +4
	})

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}
	for _, event := range response.Events {
		for _, tag := range event.Attributes {
			t.Log(tag.String())
		}
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 1)
		if balance.Cmp(helpers.StringToBigInt("14985000000000000000000")) == -1 {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "more 14985000000000000000000", balance)
		}
	}
}

func TestOrder_buy_10_more_a_lot(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	state.Coins = append(state.Coins, types.Coin{
		ID:           1,
		Name:         "Test 1",
		Symbol:       types.StrToCoinBaseSymbol("TEST1"),
		Volume:       "10015000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	}, types.Coin{
		ID:           2,
		Name:         "Test 2",
		Symbol:       types.StrToCoinBaseSymbol("TEST2"),
		Volume:       "10010000000000000000000000",
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    "90000000000000000000000000000",
		Version:      0,
		OwnerAddress: &address,
		Mintable:     false,
		Burnable:     false,
	})

	seller, _ := CreateAddress() // generate seller
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    2,
		Reserve0: "10000000000000000000000",
		Reserve1: "10000000000000000000000",
		ID:       1,
		Orders: []types.Order{
			{
				IsSale:  false,
				Volume0: "15000000000000000000000", // want to buy
				Volume1: "5000000000000000000000",  // want to sell
				ID:      1,
				Owner:   seller,
			},
		},
	})

	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
			{
				Coin:  2,
				Value: helpers.StringToBigInt("10000000000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	tx := CreateTx(app, address, transaction.TypeBuySwapPool, transaction.BuySwapPoolDataV230{
		Coins:              []types.CoinID{2, 1},
		MaximumValueToSell: helpers.StringToBigInt("10000000000000000000000000000"),
		ValueToBuy:         helpers.StringToBigInt("14989949147713228731141"),
	})

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}
	for _, event := range response.Events {
		for _, tag := range event.Attributes {
			t.Log(tag.String())
		}
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check seller's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(seller, 2)
		if balance.Cmp(helpers.StringToBigInt("14985000000000000000000")) == -1 {
			t.Fatalf("Saller balance is not correct. Expected %s, got %s", "more 14985000000000000000000", balance)
		}
	}
}
