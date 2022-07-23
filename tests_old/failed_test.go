package tests_old

//
//func TestFailedTxPayCommission(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "100",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "100",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "50",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "50",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(1))
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "49" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "49", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if reward.Sign() != 1 || r1.String() != "51" || big.NewInt(0).Add(reward, r0).Cmp(reserve0) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "51", r0, r1, reward)
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoin(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(1))
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d", response.Log, response.Code)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "6000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "49", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if reward.Sign() != 1 || r1.String() != "6345678901234567890" || big.NewInt(0).Add(reward, r0).Cmp(reserve0) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "51", r0, r1, reward)
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndPayBaseCoin_lessAvailable(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  2,
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(0))
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 102 {
//		t.Fatalf("Response code is not error: %s, %d", response.Log, response.Code)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(0)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if reward.String() != "10000000000000000000" || r1.String() != "2345678901234567890" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "5000000000000000000000000000000000000", r0, r1, reward)
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndPayBaseCoin_lessAvailable_withGasPrice(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  2,
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(0), 50)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 102 {
//		t.Fatalf("Response code is not error: %s, %d", response.Log, response.Code)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(0)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if reward.String() != "10000000000000000000" || r1.String() != "2345678901234567890" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "5000000000000000000000000000000000000", r0, r1, reward)
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndPayBaseCoin_withGasPrice(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "10000000000000000000000000000000000000",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  2,
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(0), 50)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 102 {
//		t.Fatalf("Response code is not error: %s, %d", response.Log, response.Code)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(helpers.StringToBigInt("5058076976788935588754051349077150287")) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if reward.String() != "4941923023211064411245948650922849713" || r1.String() != "2345678901234567890" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "5000000000000000000000000000000000000", r0, r1, reward)
//		}
//	}
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndPayBaseCoin(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "10000000000000000000000000000000000000",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  2,
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(0), 1)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 102 {
//		t.Fatalf("Response code is not error: %s, %d", response.Log, response.Code)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(helpers.StringToBigInt("6850582001541316700832722827411023323")) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if reward.String() != "3149417998458683299167277172588976677" || r1.String() != "2345678901234567890" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "5000000000000000000000000000000000000", r0, r1, reward)
//		}
//	}
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndCustomCommissionToken(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	}, types.Coin{
//		ID:           2,
//		Name:         "bbb",
//		Symbol:       types.StrToCoinBaseSymbol("BBB"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	}, types.Pool{
//		Coin0:    0,
//		Coin1:    2,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       2,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  2,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(2))
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 2)
//		if balance.String() != "6000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "6000000000000000000", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if r0.String() != "5000000000000000000000000000000000000" || r1.String() != "2345678901234567890" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s", "5000000000000000000000000000000000000", r0, r1)
//		}
//	}
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 2).Reserves()
//		if reward.Sign() != 1 || r1.String() != "6345678901234567890" || big.NewInt(0).Add(reward, r0).Cmp(reserve0) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "51", r0, r1, reward)
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndCustomCommissionToken_withGasPrice(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	}, types.Coin{
//		ID:           2,
//		Name:         "bbb",
//		Symbol:       types.StrToCoinBaseSymbol("BBB"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	}, types.Pool{
//		Coin0:    0,
//		Coin1:    2,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       2,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  2,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(2), 2)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 2)
//		if balance.String() != "2000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "2000000000000000000", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if r0.String() != "5000000000000000000000000000000000000" || r1.String() != "2345678901234567890" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s", "5000000000000000000000000000000000000", r0, r1)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 2).Reserves()
//		if reward.Sign() != 1 || r1.String() != "10345678901234567890" || big.NewInt(0).Add(reward, r0).Cmp(reserve0) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "1135407462159458011337752529693750189", r0, r1, reward)
//		}
//	}
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndCustomCommissionToken_withGasPrice_lessAvailable(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	}, types.Coin{
//		ID:           2,
//		Name:         "bbb",
//		Symbol:       types.StrToCoinBaseSymbol("BBB"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	}, types.Pool{
//		Coin0:    0,
//		Coin1:    2,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       2,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  2,
//				Value: "10000000000000000000",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(2), 50)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check commission balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 2)
//		if balance.String() != "0" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "0", balance)
//		}
//	}
//
//	// check pool reserves
//	{
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 1).Reserves()
//		if r0.String() != "5000000000000000000000000000000000000" || r1.String() != "2345678901234567890" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s", "5000000000000000000000000000000000000", r0, r1)
//		}
//	}
//
//	// check pool reserves
//	{
//		reward := app.GetCurrentRewards()
//		r0, r1 := app.CurrentState().Swap().GetSwapper(0, 2).Reserves()
//		if reward.Sign() != 1 || r1.String() != "12345678901234567890" || big.NewInt(0).Add(reward, r0).Cmp(reserve0) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s, %s and reward %s", "951541460730256157863947745387590710", r0, r1, reward)
//		}
//	}
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndCustomCommissionBancorReserveCoin(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	}, types.Coin{
//		ID:           2,
//		Name:         "bbb",
//		Symbol:       types.StrToCoinBaseSymbol("BBB"),
//		Volume:       "695664472679",
//		Crr:          50,
//		Reserve:      "3149417998458693299167277172588976677",
//		MaxSupply:    "695664472679",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  2,
//				Value: "695664472679",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(2))
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check commission balance
//	{
//		coin := app.CurrentState().Coins().GetCoin(2)
//		balance := app.CurrentState().Accounts().GetBalance(address, 2)
//		if balance.String() != "39200" && coin.Volume().String() != "39200" {
//			t.Fatalf("volume or balance is not correct. Expected %s, got %s and coin volume %s", "49", balance, coin.Volume())
//		}
//
//		if coin.Reserve().Cmp(helpers.StringToBigInt("10000000000000000000000")) == -1 {
//			t.Fatalf("reserve is not correct. Got %s", coin.Reserve())
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndCustomCommissionBancorReserveCoin_withGasPrice(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	}, types.Coin{
//		ID:           2,
//		Name:         "bbb",
//		Symbol:       types.StrToCoinBaseSymbol("BBB"),
//		Volume:       "695664472679",
//		Crr:          50,
//		Reserve:      "4941923023211074411245948650922849713",
//		MaxSupply:    "695664472679",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  2,
//				Value: "695664472679",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	})
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(2), 50)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check commission balance
//	{
//		coin := app.CurrentState().Coins().GetCoin(2)
//		balance := app.CurrentState().Accounts().GetBalance(address, 2)
//		if balance.String() != "31294" && coin.Volume().String() != "31294" {
//			t.Fatalf("volume or balance is not correct. Expected %s, got %s and coin volume %s", "31294", balance, coin.Volume())
//		}
//
//		if coin.Reserve().Cmp(helpers.StringToBigInt("10000000000000000000000")) == -1 {
//			t.Fatalf("reserve is not correct. Got %s", coin.Reserve())
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndCustomCommissionBancorReserveCoin_withGasPrice_lessAvailable(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	}, types.Coin{
//		ID:           2,
//		Name:         "bbb",
//		Symbol:       types.StrToCoinBaseSymbol("BBB"),
//		Volume:       "695664472679",
//		Crr:          50,
//		Reserve:      "4941923023211074411245948650922849713",
//		MaxSupply:    "695664472679",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(4e18).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  2,
//				Value: "95664472679",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	},
//		types.Account{
//			Address: types.Address{},
//			Balance: []types.Balance{
//				{
//					Coin:  2,
//					Value: "600000000000",
//				},
//			},
//		},
//	)
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(2), 50)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check commission balance
//	{
//		coin := app.CurrentState().Coins().GetCoin(2)
//		balance := app.CurrentState().Accounts().GetBalance(address, 2)
//		if balance.String() != "0" && coin.Volume().String() != "600000000000" {
//			t.Fatalf("volume or balance is not correct. Expected %s, got %s and coin volume %s", "0", balance, coin.Volume())
//		}
//
//		if coin.Reserve().Cmp(helpers.StringToBigInt("10000000000000000000000")) == -1 {
//			t.Fatalf("reserve is not correct. Got %s", coin.Reserve())
//		}
//	}
//
//}
//
//func TestFailedTxPayCommission_customPriceCoinAndCustomCommissionBancorReserveCoin_withGasPrice_lessReserve(t *testing.T) {
//	address, pk := CreateAddress() // create account for test
//
//	state := DefaultAppState() // generate default state
//
//	state.Version = "v300"
//
//	state.Coins = append(state.Coins, types.Coin{
//		ID:           1,
//		Name:         "aaa",
//		Symbol:       types.StrToCoinBaseSymbol("AAA"),
//		Volume:       "12345678901234567890",
//		Crr:          0,
//		Reserve:      "0",
//		MaxSupply:    "12345678901234567890",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	}, types.Coin{
//		ID:           2,
//		Name:         "bbb",
//		Symbol:       types.StrToCoinBaseSymbol("BBB"),
//		Volume:       "695664472679",
//		Crr:          50,
//		Reserve:      "4941923023211074411245948650922849710",
//		MaxSupply:    "695664472679",
//		Version:      0,
//		OwnerAddress: nil,
//		Mintable:     false,
//		Burnable:     false,
//	})
//	state.Commission.Coin = 1
//	state.Commission.FailedTx = big.NewInt(1e15).String()
//
//	reserve0 := helpers.BipToPip(big.NewInt(5e18))
//	state.Pools = append(state.Pools, types.Pool{
//		Coin0:    0,
//		Coin1:    1,
//		Reserve0: reserve0.String(),
//		Reserve1: "2345678901234567890",
//		ID:       1,
//	})
//	state.Accounts = append(state.Accounts, types.Account{
//		Address: address,
//		Balance: []types.Balance{
//			{
//				Coin:  uint64(types.GetBaseCoinID()),
//				Value: "1",
//			},
//			{
//				Coin:  1,
//				Value: "10000000000000000000",
//			},
//			{
//				Coin:  2,
//				Value: "95664472679",
//			},
//		},
//		Nonce:        0,
//		MultisigData: nil,
//	},
//		types.Account{
//			Address: types.Address{},
//			Balance: []types.Balance{
//				{
//					Coin:  2,
//					Value: "600000000000",
//				},
//			},
//		},
//	)
//
//	app := CreateApp(state) // create application
//	SendBeginBlock(app, 1)  // send BeginBlock
//
//	recipient, _ := CreateAddress() // generate recipient
//	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
//		Coin:  types.GetBaseCoinID(),
//		To:    recipient,
//		Value: big.NewInt(100),
//	}, types.CoinID(2), 50)
//
//	response := SendTx(app, SignTx(pk, tx)) // compose and send tx
//
//	// check that result is error
//	if response.Code != 107 {
//		t.Fatalf("Response code is not error: %s, %d, %s", response.Log, response.Code, response.Info)
//	}
//
//	SendEndBlock(app, 1) // send EndBlock
//	SendCommit(app)      // send Commit
//
//	// check recipient's balance
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
//		if balance.Cmp(big.NewInt(1)) != 0 {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", big.NewInt(1), balance)
//		}
//	}
//
//	{
//		balance := app.CurrentState().Accounts().GetBalance(address, 1)
//		if balance.String() != "10000000000000000000" {
//			t.Fatalf("address balance is not correct. Expected %s, got %s", "10000000000000000000", balance)
//		}
//	}
//
//	// check commission balance
//	{
//		coin := app.CurrentState().Coins().GetCoin(2)
//		balance := app.CurrentState().Accounts().GetBalance(address, 2)
//		if balance.String() != "88294932483" && coin.Volume().String() != "688294932483" {
//			t.Fatalf("volume or balance is not correct. Expected %s, got %s and coin volume %s", "88294932483", balance, coin.Volume())
//		}
//
//		if coin.Reserve().Cmp(helpers.StringToBigInt("10000000000000000000000")) == -1 {
//			t.Fatalf("reserve is not correct. Got %s", coin.Reserve())
//		}
//	}
//
//}
