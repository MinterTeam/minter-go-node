package formula

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/math"
	"math/big"
)

// Return = supply * ((1 + deposit / reserve) ^ (crr / 100) - 1)
func CalculatePurchaseReturn(supply *big.Int, reserve *big.Int, crr uint, deposit *big.Int) *big.Int {
	if deposit.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	if crr == 100 {
		result := big.NewInt(0).Mul(supply, deposit)

		return result.Div(result, reserve)
	}

	tSupply := big.NewFloat(0).SetInt(supply)
	tReserve := big.NewFloat(0).SetInt(reserve)
	tDeposit := big.NewFloat(0).SetInt(deposit)

	res := big.NewFloat(0).Quo(tDeposit, tReserve)      // deposit / reserve
	res.Add(res, big.NewFloat(1))                       // 1 + (deposit / reserve)
	res = math.Pow(res, big.NewFloat(float64(crr)/100)) // (1 + deposit / reserve) ^ (crr / 100)
	res.Sub(res, big.NewFloat(1))                       // ((1 + deposit / reserve) ^ (crr / 100) - 1)
	res.Mul(res, tSupply)                               // supply * ((1 + deposit / reserve) ^ (crr / 100) - 1)

	result, _ := res.Int(nil)

	return result
}

// reversed function CalculatePurchaseReturn
func CalculatePurchaseAmount(supply *big.Int, reserve *big.Int, crr uint, wantReceive *big.Int) *big.Int {

	panic("Implement")

	return big.NewInt(0)
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

	tSupply := big.NewFloat(0).SetInt(supply)
	tReserve := big.NewFloat(0).SetInt(reserve)
	tSellAmount := big.NewFloat(0).SetInt(sellAmount)

	res := big.NewFloat(0).Quo(tSellAmount, tSupply)        // sellAmount / supply
	res.Sub(big.NewFloat(1), res)                           // (1 - sellAmount / supply)
	res = math.Pow(res, big.NewFloat(1/(float64(crr)/100))) // (1 - sellAmount / supply) ^ (1 / (crr / 100))
	res.Sub(big.NewFloat(1), res)                           // (1 - (1 - sellAmount / supply) ^ (1 / (crr / 100)))
	res.Mul(res, tReserve)                                  // reserve * (1 - (1 - sellAmount / supply) ^ (1 / (crr / 100)))

	result, _ := res.Int(nil)

	return result
}

// reversed function CalculateSaleReturn
func CalculateSaleAmount(supply *big.Int, reserve *big.Int, crr uint, wantReceive *big.Int) *big.Int {

	tSupply := big.NewFloat(0).SetInt(supply)
	tReserve := big.NewFloat(0).SetInt(reserve)
	tWantReceive := big.NewFloat(0).SetInt(wantReceive)

	res := big.NewFloat(0).Sub(tWantReceive, tReserve)
	res.Mul(res, big.NewFloat(-1))
	res.Quo(res, tReserve)
	res = math.Pow(res, big.NewFloat(float64(crr)/100))
	res.Add(res, big.NewFloat(-1))
	res.Mul(res, big.NewFloat(-1))
	res.Mul(res, tSupply)

	result, _ := res.Int(nil)

	return result
}
