package formula

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/math"
	"math/big"
)

const (
	precision = 5000
)

func newFloat(x float64) *big.Float {
	return big.NewFloat(x).SetPrec(precision)
}

// Return = supply * ((1 + deposit / reserve) ^ (crr / 100) - 1)
func CalculatePurchaseReturn(supply *big.Int, reserve *big.Int, crr uint, deposit *big.Int) *big.Int {
	if deposit.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	if crr == 100 {
		result := big.NewInt(0).Mul(supply, deposit)

		return result.Div(result, reserve)
	}

	tSupply := newFloat(0).SetInt(supply)
	tReserve := newFloat(0).SetInt(reserve)
	tDeposit := newFloat(0).SetInt(deposit)

	res := newFloat(0).Quo(tDeposit, tReserve)      // deposit / reserve
	res.Add(res, newFloat(1))                       // 1 + (deposit / reserve)
	res = math.Pow(res, newFloat(float64(crr)/100)) // (1 + deposit / reserve) ^ (crr / 100)
	res.Sub(res, newFloat(1))                       // ((1 + deposit / reserve) ^ (crr / 100) - 1)
	res.Mul(res, tSupply)                           // supply * ((1 + deposit / reserve) ^ (crr / 100) - 1)

	result, _ := res.Int(nil)

	return result
}

// reversed function CalculatePurchaseReturn
// deposit = reserve * (((wantReceive + supply) / supply)^(100/c) - 1)
func CalculatePurchaseAmount(supply *big.Int, reserve *big.Int, crr uint, wantReceive *big.Int) *big.Int {

	tSupply := newFloat(0).SetInt(supply)
	tReserve := newFloat(0).SetInt(reserve)
	tWantReceive := newFloat(0).SetInt(wantReceive)

	res := newFloat(0).Add(tWantReceive, tSupply)   // reserve + supply
	res.Quo(res, tSupply)                           // (reserve + supply) / supply
	res = math.Pow(res, newFloat(100/float64(crr))) // ((reserve + supply) / supply)^(100/c)
	res.Sub(res, newFloat(1))                       // (((reserve + supply) / supply)^(100/c) - 1)
	res.Mul(res, tReserve)                          // reserve * (((reserve + supply) / supply)^(100/c) - 1)

	result, _ := res.Int(nil)

	return result
}

// Return = reserve * (1 - (1 - sellAmount / supply) ^ (1 / (crr / 100)))
func CalculateSaleReturn(supply *big.Int, reserve *big.Int, crr uint, sellAmount *big.Int) *big.Int {

	// special case for 0 sell amount
	if sellAmount.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	// special case for selling the entire supply
	if sellAmount.Cmp(supply) == 0 {
		return big.NewInt(0).Set(reserve)
	}

	if crr == 100 {
		ret := big.NewInt(0).Mul(reserve, sellAmount)
		ret.Div(ret, supply)

		return ret
	}

	tSupply := newFloat(0).SetInt(supply)
	tReserve := newFloat(0).SetInt(reserve)
	tSellAmount := newFloat(0).SetInt(sellAmount)

	res := newFloat(0).Quo(tSellAmount, tSupply)        // sellAmount / supply
	res.Sub(newFloat(1), res)                           // (1 - sellAmount / supply)
	res = math.Pow(res, newFloat(1/(float64(crr)/100))) // (1 - sellAmount / supply) ^ (1 / (crr / 100))
	res.Sub(newFloat(1), res)                           // (1 - (1 - sellAmount / supply) ^ (1 / (crr / 100)))
	res.Mul(res, tReserve)                              // reserve * (1 - (1 - sellAmount / supply) ^ (1 / (crr / 100)))

	result, _ := res.Int(nil)

	return result
}

// reversed function CalculateSaleReturn
func CalculateSaleAmount(supply *big.Int, reserve *big.Int, crr uint, wantReceive *big.Int) *big.Int {

	tSupply := newFloat(0).SetInt(supply)
	tReserve := newFloat(0).SetInt(reserve)
	tWantReceive := newFloat(0).SetInt(wantReceive)

	res := newFloat(0).Sub(tWantReceive, tReserve)
	res.Mul(res, newFloat(-1))
	res.Quo(res, tReserve)
	res = math.Pow(res, newFloat(float64(crr)/100))
	res.Add(res, newFloat(-1))
	res.Mul(res, newFloat(-1))
	res.Mul(res, tSupply)

	result, _ := res.Int(nil)

	return result
}
