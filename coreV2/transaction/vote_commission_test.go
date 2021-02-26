package transaction

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func TestPriceCommissionTx(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, addr := getAccount()
	coin1 := createNonReserveCoin(cState)
	cState.Accounts.SubBalance(types.Address{}, coin1, big.NewInt(1e18))

	_, _, liquidity, id := cState.Swap.PairCreate(types.GetBaseCoinID(), coin1, big.NewInt(1e18), big.NewInt(1e18))
	coins := liquidityCoinName(coin1, types.GetBaseCoinID())
	liquidityCoinID := cState.App.GetNextCoinID()
	cState.Coins.CreateToken(liquidityCoinID, LiquidityCoinSymbol(id), "Pool "+coins, true, true, big.NewInt(0).Set(liquidity), maxCoinSupply, nil)
	cState.Accounts.AddBalance(addr, liquidityCoinID, liquidity)
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), big.NewInt(1e18))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	{
		data := VoteCommissionData{
			PubKey:                  pubkey,
			Height:                  uint64(100500),
			Coin:                    coin1,
			PayloadByte:             big.NewInt(1e18),
			Send:                    big.NewInt(1e18),
			BuyBancor:               big.NewInt(1e18),
			SellBancor:              big.NewInt(1e18),
			SellAllBancor:           big.NewInt(1e18),
			BuyPoolBase:             big.NewInt(1e18),
			BuyPoolDelta:            big.NewInt(5e17),
			SellPoolBase:            big.NewInt(1e18),
			SellPoolDelta:           big.NewInt(5e17),
			SellAllPoolBase:         big.NewInt(1e18),
			SellAllPoolDelta:        big.NewInt(5e17),
			CreateTicker3:           big.NewInt(1e18),
			CreateTicker4:           big.NewInt(1e18),
			CreateTicker5:           big.NewInt(1e18),
			CreateTicker6:           big.NewInt(1e18),
			CreateTicker7to10:       big.NewInt(1e18),
			CreateCoin:              big.NewInt(1e18),
			CreateToken:             big.NewInt(1e18),
			RecreateCoin:            big.NewInt(1e18),
			RecreateToken:           big.NewInt(1e18),
			DeclareCandidacy:        big.NewInt(1e18),
			Delegate:                big.NewInt(1e18),
			Unbond:                  big.NewInt(1e18),
			RedeemCheck:             big.NewInt(1e18),
			SetCandidateOn:          big.NewInt(1e18),
			SetCandidateOff:         big.NewInt(1e18),
			CreateMultisig:          big.NewInt(1e18),
			MultisendBase:           big.NewInt(1e18),
			MultisendDelta:          big.NewInt(1e18),
			EditCandidate:           big.NewInt(1e18),
			SetHaltBlock:            big.NewInt(1e18),
			EditTickerOwner:         big.NewInt(1e18),
			EditMultisig:            big.NewInt(1e18),
			PriceVote:               big.NewInt(1e18),
			EditCandidatePublicKey:  big.NewInt(1e18),
			CreateSwapPool:          big.NewInt(5e17),
			AddLiquidity:            big.NewInt(1e18),
			RemoveLiquidity:         big.NewInt(1e18),
			EditCandidateCommission: big.NewInt(1e18),
			MoveStake:               big.NewInt(1e18),
			MintToken:               big.NewInt(1e18),
			BurnToken:               big.NewInt(1e18),
			VoteCommission:          big.NewInt(1e18),
			VoteUpdate:              big.NewInt(1e18),
			More:                    nil,
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
			Type:          TypeVoteCommission,
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

		response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != 0 {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	{
		data := VoteCommissionData{
			PayloadByte:             big.NewInt(1e18),
			Send:                    big.NewInt(1e18),
			BuyBancor:               big.NewInt(1e18),
			SellBancor:              big.NewInt(1e18),
			SellAllBancor:           big.NewInt(1e18),
			BuyPoolBase:             big.NewInt(1e18),
			BuyPoolDelta:            big.NewInt(5e17),
			SellPoolBase:            big.NewInt(1e18),
			SellPoolDelta:           big.NewInt(5e17),
			SellAllPoolBase:         big.NewInt(1e18),
			SellAllPoolDelta:        big.NewInt(5e17),
			CreateTicker3:           big.NewInt(1e18),
			CreateTicker4:           big.NewInt(1e18),
			CreateTicker5:           big.NewInt(1e18),
			CreateTicker6:           big.NewInt(1e18),
			CreateTicker7to10:       big.NewInt(1e18),
			CreateCoin:              big.NewInt(1e18),
			CreateToken:             big.NewInt(1e18),
			RecreateCoin:            big.NewInt(1e18),
			RecreateToken:           big.NewInt(1e18),
			DeclareCandidacy:        big.NewInt(1e18),
			Delegate:                big.NewInt(1e18),
			Unbond:                  big.NewInt(1e18),
			RedeemCheck:             big.NewInt(1e18),
			SetCandidateOn:          big.NewInt(1e18),
			SetCandidateOff:         big.NewInt(1e18),
			CreateMultisig:          big.NewInt(1e18),
			MultisendBase:           big.NewInt(1e18),
			MultisendDelta:          big.NewInt(1e18),
			EditCandidate:           big.NewInt(1e18),
			SetHaltBlock:            big.NewInt(1e18),
			EditTickerOwner:         big.NewInt(1e18),
			EditMultisig:            big.NewInt(1e18),
			PriceVote:               big.NewInt(1e18),
			EditCandidatePublicKey:  big.NewInt(1e18),
			AddLiquidity:            big.NewInt(1e18),
			RemoveLiquidity:         big.NewInt(1e18),
			EditCandidateCommission: big.NewInt(1e18),
			MoveStake:               big.NewInt(1e18),
			MintToken:               big.NewInt(1e18),
			BurnToken:               big.NewInt(1e18),
			VoteCommission:          big.NewInt(1e18),
			VoteUpdate:              big.NewInt(1e18),
			Coin:                    coin1,
			PubKey:                  pubkey,
			Height:                  uint64(100500),
		}
		encodedData, err := rlp.EncodeToBytes(data)
		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeVoteCommission,
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

		response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != code.VoiceAlreadyExists {
			t.Fatalf("Response code is not %d. Error: %s", code.VoiceAlreadyExists, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestPriceCommissionDeleteTx(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, addr := getAccount()
	coin1 := createNonReserveCoin(cState)
	cState.Accounts.SubBalance(types.Address{}, coin1, big.NewInt(1e18))

	_, _, liquidity, id := cState.Swap.PairCreate(types.GetBaseCoinID(), coin1, big.NewInt(1e18), big.NewInt(1e18))
	coins := liquidityCoinName(coin1, types.GetBaseCoinID())
	liquidityCoinID := cState.App.GetNextCoinID()
	cState.Coins.CreateToken(liquidityCoinID, LiquidityCoinSymbol(id), "Pool "+coins, true, true, big.NewInt(0).Set(liquidity), maxCoinSupply, nil)
	cState.Accounts.AddBalance(addr, liquidityCoinID, liquidity)
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), big.NewInt(2e18))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	{
		data := VoteCommissionData{
			PayloadByte:             big.NewInt(1e18),
			Send:                    big.NewInt(1e18),
			BuyBancor:               big.NewInt(1e18),
			SellBancor:              big.NewInt(1e18),
			SellAllBancor:           big.NewInt(1e18),
			BuyPoolBase:             big.NewInt(1e18),
			BuyPoolDelta:            big.NewInt(5e17),
			SellPoolBase:            big.NewInt(1e18),
			SellPoolDelta:           big.NewInt(5e17),
			SellAllPoolBase:         big.NewInt(1e18),
			SellAllPoolDelta:        big.NewInt(5e17),
			CreateTicker3:           big.NewInt(1e18),
			CreateTicker4:           big.NewInt(1e18),
			CreateTicker5:           big.NewInt(1e18),
			CreateTicker6:           big.NewInt(1e18),
			CreateTicker7to10:       big.NewInt(1e18),
			CreateCoin:              big.NewInt(1e18),
			CreateToken:             big.NewInt(1e18),
			RecreateCoin:            big.NewInt(1e18),
			RecreateToken:           big.NewInt(1e18),
			DeclareCandidacy:        big.NewInt(1e18),
			Delegate:                big.NewInt(1e18),
			Unbond:                  big.NewInt(1e18),
			RedeemCheck:             big.NewInt(1e18),
			SetCandidateOn:          big.NewInt(1e18),
			SetCandidateOff:         big.NewInt(1e18),
			CreateMultisig:          big.NewInt(1e18),
			MultisendBase:           big.NewInt(1e18),
			MultisendDelta:          big.NewInt(1e18),
			EditCandidate:           big.NewInt(1e18),
			SetHaltBlock:            big.NewInt(1e18),
			EditTickerOwner:         big.NewInt(1e18),
			EditMultisig:            big.NewInt(1e18),
			PriceVote:               big.NewInt(1e18),
			EditCandidatePublicKey:  big.NewInt(1e18),
			AddLiquidity:            big.NewInt(1e18),
			RemoveLiquidity:         big.NewInt(1e18),
			EditCandidateCommission: big.NewInt(1e18),
			MoveStake:               big.NewInt(1e18),
			MintToken:               big.NewInt(1e18),
			BurnToken:               big.NewInt(1e18),
			VoteCommission:          big.NewInt(1e18),
			VoteUpdate:              big.NewInt(1e18),
			Coin:                    coin1,
			PubKey:                  pubkey,
			Height:                  uint64(100500),
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
			Type:          TypeVoteCommission,
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

		response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != 0 {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	cState.Commission.Delete(100500)
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
	{
		data := VoteCommissionData{
			PayloadByte:             big.NewInt(1e18),
			Send:                    big.NewInt(1e18),
			BuyBancor:               big.NewInt(1e18),
			SellBancor:              big.NewInt(1e18),
			SellAllBancor:           big.NewInt(1e18),
			BuyPoolBase:             big.NewInt(1e18),
			BuyPoolDelta:            big.NewInt(5e17),
			SellPoolBase:            big.NewInt(1e18),
			SellPoolDelta:           big.NewInt(5e17),
			SellAllPoolBase:         big.NewInt(1e18),
			SellAllPoolDelta:        big.NewInt(5e17),
			CreateTicker3:           big.NewInt(1e18),
			CreateTicker4:           big.NewInt(1e18),
			CreateTicker5:           big.NewInt(1e18),
			CreateTicker6:           big.NewInt(1e18),
			CreateTicker7to10:       big.NewInt(1e18),
			CreateCoin:              big.NewInt(1e18),
			CreateToken:             big.NewInt(1e18),
			RecreateCoin:            big.NewInt(1e18),
			RecreateToken:           big.NewInt(1e18),
			DeclareCandidacy:        big.NewInt(1e18),
			Delegate:                big.NewInt(1e18),
			Unbond:                  big.NewInt(1e18),
			RedeemCheck:             big.NewInt(1e18),
			SetCandidateOn:          big.NewInt(1e18),
			SetCandidateOff:         big.NewInt(1e18),
			CreateMultisig:          big.NewInt(1e18),
			MultisendBase:           big.NewInt(1e18),
			MultisendDelta:          big.NewInt(1e18),
			EditCandidate:           big.NewInt(1e18),
			SetHaltBlock:            big.NewInt(1e18),
			EditTickerOwner:         big.NewInt(1e18),
			EditMultisig:            big.NewInt(1e18),
			PriceVote:               big.NewInt(1e18),
			EditCandidatePublicKey:  big.NewInt(1e18),
			AddLiquidity:            big.NewInt(1e18),
			RemoveLiquidity:         big.NewInt(1e18),
			EditCandidateCommission: big.NewInt(1e18),
			MoveStake:               big.NewInt(1e18),
			MintToken:               big.NewInt(1e18),
			BurnToken:               big.NewInt(1e18),
			VoteCommission:          big.NewInt(1e18),
			VoteUpdate:              big.NewInt(1e18),
			Coin:                    coin1,
			PubKey:                  pubkey,
			Height:                  uint64(100500),
		}
		encodedData, err := rlp.EncodeToBytes(data)
		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeVoteCommission,
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

		response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != code.OK {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestPriceCommissionAnyTx(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin1 := createNonReserveCoin(cState)

	{
		privateKey, addr := getAccount()
		cState.Accounts.SubBalance(types.Address{}, coin1, big.NewInt(1e18))

		_, _, liquidity, id := cState.Swap.PairCreate(types.GetBaseCoinID(), coin1, big.NewInt(1e18), big.NewInt(1e18))
		coins := liquidityCoinName(coin1, types.GetBaseCoinID())
		liquidityCoinID := cState.App.GetNextCoinID()
		cState.Coins.CreateToken(liquidityCoinID, LiquidityCoinSymbol(id), "Pool "+coins, true, true, big.NewInt(0).Set(liquidity), maxCoinSupply, nil)
		cState.Accounts.AddBalance(addr, liquidityCoinID, liquidity)
		cState.App.SetCoinsCount(liquidityCoinID.Uint32())
		cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), big.NewInt(2e18))

		pubkey := [32]byte{}
		rand.Read(pubkey[:])

		cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
		cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

		data := VoteCommissionData{
			PayloadByte:             big.NewInt(1e18),
			Send:                    big.NewInt(1e18),
			BuyBancor:               big.NewInt(1e18),
			SellBancor:              big.NewInt(1e18),
			SellAllBancor:           big.NewInt(1e18),
			BuyPoolBase:             big.NewInt(1e18),
			BuyPoolDelta:            big.NewInt(5e17),
			SellPoolBase:            big.NewInt(1e18),
			SellPoolDelta:           big.NewInt(5e17),
			SellAllPoolBase:         big.NewInt(1e18),
			SellAllPoolDelta:        big.NewInt(5e17),
			CreateTicker3:           big.NewInt(1e18),
			CreateTicker4:           big.NewInt(1e18),
			CreateTicker5:           big.NewInt(1e18),
			CreateTicker6:           big.NewInt(1e18),
			CreateTicker7to10:       big.NewInt(1e18),
			CreateCoin:              big.NewInt(1e18),
			CreateToken:             big.NewInt(1e18),
			RecreateCoin:            big.NewInt(1e18),
			RecreateToken:           big.NewInt(1e18),
			DeclareCandidacy:        big.NewInt(1e18),
			Delegate:                big.NewInt(1e18),
			Unbond:                  big.NewInt(1e18),
			RedeemCheck:             big.NewInt(1e18),
			SetCandidateOn:          big.NewInt(1e18),
			SetCandidateOff:         big.NewInt(1e18),
			CreateMultisig:          big.NewInt(1e18),
			MultisendBase:           big.NewInt(1e18),
			MultisendDelta:          big.NewInt(1e18),
			EditCandidate:           big.NewInt(1e18),
			SetHaltBlock:            big.NewInt(1e18),
			EditTickerOwner:         big.NewInt(1e18),
			EditMultisig:            big.NewInt(1e18),
			PriceVote:               big.NewInt(1e18),
			EditCandidatePublicKey:  big.NewInt(1e18),
			AddLiquidity:            big.NewInt(1e18),
			RemoveLiquidity:         big.NewInt(1e18),
			EditCandidateCommission: big.NewInt(1e18),
			MoveStake:               big.NewInt(1e18),
			MintToken:               big.NewInt(1e18),
			BurnToken:               big.NewInt(1e18),
			VoteCommission:          big.NewInt(1e18),
			VoteUpdate:              big.NewInt(1e18),
			Coin:                    coin1,
			PubKey:                  pubkey,
			Height:                  uint64(100500),
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
			Type:          TypeVoteCommission,
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

		response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != 0 {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
	{
		privateKey, addr := getAccount()
		cState.Accounts.SubBalance(types.Address{}, coin1, big.NewInt(1e18))

		_, _, liquidity, id := cState.Swap.PairCreate(types.GetBaseCoinID(), coin1, big.NewInt(1e18), big.NewInt(1e18))
		coins := liquidityCoinName(coin1, types.GetBaseCoinID())
		liquidityCoinID := cState.App.GetNextCoinID()
		cState.Coins.CreateToken(liquidityCoinID, LiquidityCoinSymbol(id), "Pool "+coins, true, true, big.NewInt(0).Set(liquidity), maxCoinSupply, nil)
		cState.Accounts.AddBalance(addr, liquidityCoinID, liquidity)
		cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), big.NewInt(2e18))
		cState.App.SetCoinsCount(liquidityCoinID.Uint32())

		pubkey := [32]byte{}
		rand.Read(pubkey[:])

		cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
		cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

		data := VoteCommissionData{
			PayloadByte:             big.NewInt(1e18),
			Send:                    big.NewInt(1e18),
			BuyBancor:               big.NewInt(1e18),
			SellBancor:              big.NewInt(1e18),
			SellAllBancor:           big.NewInt(1e18),
			BuyPoolBase:             big.NewInt(1e18),
			BuyPoolDelta:            big.NewInt(5e17),
			SellPoolBase:            big.NewInt(1e18),
			SellPoolDelta:           big.NewInt(5e17),
			SellAllPoolBase:         big.NewInt(1e18),
			SellAllPoolDelta:        big.NewInt(5e17),
			CreateTicker3:           big.NewInt(1e18),
			CreateTicker4:           big.NewInt(1e18),
			CreateTicker5:           big.NewInt(1e18),
			CreateTicker6:           big.NewInt(1e18),
			CreateTicker7to10:       big.NewInt(1e18),
			CreateCoin:              big.NewInt(1e18),
			CreateToken:             big.NewInt(1e18),
			RecreateCoin:            big.NewInt(1e18),
			RecreateToken:           big.NewInt(1e18),
			DeclareCandidacy:        big.NewInt(1e18),
			Delegate:                big.NewInt(1e18),
			Unbond:                  big.NewInt(1e18),
			RedeemCheck:             big.NewInt(1e18),
			SetCandidateOn:          big.NewInt(1e18),
			SetCandidateOff:         big.NewInt(1e18),
			CreateMultisig:          big.NewInt(1e18),
			MultisendBase:           big.NewInt(1e18),
			MultisendDelta:          big.NewInt(1e18),
			EditCandidate:           big.NewInt(1e18),
			SetHaltBlock:            big.NewInt(1e18),
			EditTickerOwner:         big.NewInt(1e18),
			EditMultisig:            big.NewInt(1e18),
			PriceVote:               big.NewInt(1e18),
			EditCandidatePublicKey:  big.NewInt(1e18),
			AddLiquidity:            big.NewInt(1e18),
			RemoveLiquidity:         big.NewInt(1e18),
			EditCandidateCommission: big.NewInt(1e18),
			MoveStake:               big.NewInt(1e18),
			MintToken:               big.NewInt(1e18),
			BurnToken:               big.NewInt(1e18),
			VoteCommission:          big.NewInt(1e18),
			VoteUpdate:              big.NewInt(1e18),
			Coin:                    coin1,
			PubKey:                  pubkey,
			Height:                  uint64(100500),
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
			Type:          TypeVoteCommission,
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

		response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != code.OK {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestCustomCommissionPriceCoin_sendTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	usdCoinID := types.CoinID(666999)
	cState.Coins.CreateToken(usdCoinID, types.StrToCoinSymbol("USD"), "USD Stable", true, true, helpers.BipToPip(big.NewInt(1e18)), maxCoinSupply, nil)
	usdPool := helpers.BipToPip(big.NewInt(1e18))
	bipPool := big.NewInt(0).Sub(helpers.BipToPip(big.NewInt(1e18)), big.NewInt(0).Div(big.NewInt(0).Mul(helpers.BipToPip(big.NewInt(1e18)), big.NewInt(2)), big.NewInt(1000)))
	_, _, liquidity, id := cState.Swap.PairCreate(usdCoinID, types.GetBaseCoinID(), usdPool, bipPool)
	coins := liquidityCoinName(usdCoinID, types.GetBaseCoinID())
	coinID := cState.App.GetNextCoinID()
	cState.Coins.CreateToken(coinID, LiquidityCoinSymbol(id), "Pool "+coins, true, true, big.NewInt(0).Set(liquidity), maxCoinSupply, nil)
	cState.Accounts.AddBalance(types.Address{}, coinID, liquidity)

	price := cState.Commission.GetCommissions()
	price.Coin = usdCoinID
	cState.Commission.SetNewCommissions(price.Encode())

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := SendData{
		Coin:  coin,
		To:    to,
		Value: value,
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
		Type:          TypeSend,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error: %s", response.Log)
	}

	for _, tag := range response.Tags {
		t.Log(string(tag.Key), string(tag.Value))
	}

	targetBalance, _ := big.NewInt(0).SetString("999989989999999999999999", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), targetBalance, balance)
	}

	targetTestBalance, _ := big.NewInt(0).SetString("10000000000000000000", 10)
	testBalance := cState.Accounts.GetBalance(to, coin)
	if testBalance.Cmp(targetTestBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", to.String(), targetTestBalance, testBalance)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCustomCommissionPriceCoinAndGasCastomCoin_sendTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	usdCoinID := types.CoinID(666999)
	usdPool := helpers.BipToPip(big.NewInt(1e18))
	bipPool := big.NewInt(0).Sub(helpers.BipToPip(big.NewInt(1e18)), big.NewInt(0).Div(big.NewInt(0).Mul(helpers.BipToPip(big.NewInt(1e18)), big.NewInt(2)), big.NewInt(1000)))
	cState.Coins.CreateToken(usdCoinID, types.StrToCoinSymbol("USD"), "USD Stable", true, true, big.NewInt(0).Add(big.NewInt(1e18), usdPool), maxCoinSupply, nil)
	_, _, liquidity, id := cState.Swap.PairCreate(usdCoinID, types.GetBaseCoinID(), usdPool, bipPool)
	coins := liquidityCoinName(usdCoinID, types.GetBaseCoinID())
	coinID := cState.App.GetNextCoinID()
	cState.Coins.CreateToken(coinID, LiquidityCoinSymbol(id), "Pool "+coins, true, true, big.NewInt(0).Set(liquidity), maxCoinSupply, nil)
	cState.Accounts.AddBalance(types.Address{}, coinID, liquidity)
	cState.App.SetCoinsCount(coinID.Uint32())

	price := cState.Commission.GetCommissions()
	price.Coin = usdCoinID
	cState.Commission.SetNewCommissions(price.Encode())

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, usdCoinID, big.NewInt(1e18))

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := SendData{
		Coin:  coin,
		To:    to,
		Value: value,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       usdCoinID,
		Type:          TypeSend,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error: %s", response.Log)
	}

	// for _, tag := range response.Tags {
	// 	t.Log(string(tag.Key), string(tag.Value))
	// }

	targetBalance, _ := big.NewInt(0).SetString("999990000000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), targetBalance, balance)
	}

	targetGasBalance, _ := big.NewInt(0).SetString("989959879679198074", 10)
	balanceGas := cState.Accounts.GetBalance(addr, usdCoinID)
	if balanceGas.Cmp(targetGasBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), targetGasBalance, balanceGas)
	}

	targetTestBalance, _ := big.NewInt(0).SetString("10000000000000000000", 10)
	testBalance := cState.Accounts.GetBalance(to, coin)
	if testBalance.Cmp(targetTestBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", to.String(), targetTestBalance, testBalance)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
