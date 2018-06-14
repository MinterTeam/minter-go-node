package formula

import (
	"math/big"
	"testing"
)

type PurchaseReturnData struct {
	Supply  *big.Int
	Reserve *big.Int
	Crr     uint
	Deposit *big.Int
	Result  *big.Int
}

func TestCalculatePurchaseReturn(t *testing.T) {

	data := []PurchaseReturnData{
		{
			Supply:  big.NewInt(1000000),
			Reserve: big.NewInt(100),
			Crr:     40,
			Deposit: big.NewInt(100),
			Result:  big.NewInt(319507),
		},
		{
			Supply:  big.NewInt(1000000),
			Reserve: big.NewInt(100),
			Crr:     100,
			Deposit: big.NewInt(100),
			Result:  big.NewInt(1000000),
		},
	}

	for _, item := range data {
		result := CalculatePurchaseReturn(item.Supply, item.Reserve, item.Crr, item.Deposit)

		if result.Cmp(item.Result) != 0 {
			t.Errorf("CalculatePurchaseReturn result is not correct. Expected %s, got %s", item.Result, result)
		}
	}
}

type CalculateSaleReturnData struct {
	Supply     *big.Int
	Reserve    *big.Int
	Crr        uint
	SellAmount *big.Int
	Result     *big.Int
}

func TestCalculateSaleReturn(t *testing.T) {

	data := []CalculateSaleReturnData{
		{
			Supply:     big.NewInt(1000000),
			Reserve:    big.NewInt(100),
			Crr:        40,
			SellAmount: big.NewInt(1000000),
			Result:     big.NewInt(100),
		},
		{
			Supply:     big.NewInt(1000000),
			Reserve:    big.NewInt(100),
			Crr:        10,
			SellAmount: big.NewInt(100000),
			Result:     big.NewInt(65),
		},
	}

	for _, item := range data {
		result := CalculateSaleReturn(item.Supply, item.Reserve, item.Crr, item.SellAmount)

		if result.Cmp(item.Result) != 0 {
			t.Errorf("CalculateSaleReturn result is not correct. Expected %s, got %s", item.Result, result)
		}
	}
}

type CalculateBuyDepositData struct {
	Supply      *big.Int
	Reserve     *big.Int
	Crr         uint
	WantReceive *big.Int
	Result      *big.Int
}

func TestCalculateBuyDeposit(t *testing.T) {

	data := []CalculateBuyDepositData{
		{
			Supply:      big.NewInt(1000000),
			Reserve:     big.NewInt(100),
			Crr:         40,
			WantReceive: big.NewInt(10),
			Result:      big.NewInt(41268),
		},
		{
			Supply:      big.NewInt(1000000),
			Reserve:     big.NewInt(100),
			Crr:         10,
			WantReceive: big.NewInt(100),
			Result:      big.NewInt(1000000),
		},
	}

	for _, item := range data {
		result := CalculateSaleAmount(item.Supply, item.Reserve, item.Crr, item.WantReceive)

		if result.Cmp(item.Result) != 0 {
			t.Errorf("CalculateSaleAmount result is not correct. Expected %s, got %s", item.Result, result)
		}
	}
}
