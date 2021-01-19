package formula

import (
	"github.com/MinterTeam/minter-go-node/math"
	"math/big"
)

const (
	precision = 100
)

// CalculatePurchaseReturn calculates amount of coin that user will receive by depositing given amount of BIP
// Return = supply * ((1 + deposit / reserve) ^ (crr / 100) - 1)
func CalculatePurchaseReturn(supply *big.Int, reserve *big.Int, crr uint32, deposit *big.Int) *big.Int {
	if deposit.Sign() == 0 {
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

// CalculatePurchaseAmount is the reversed version of function CalculatePurchaseReturn
// Deposit = reserve * (((wantReceive + supply) / supply)^(100/c) - 1)
func CalculatePurchaseAmount(supply *big.Int, reserve *big.Int, crr uint32, wantReceive *big.Int) *big.Int {
	if wantReceive.Sign() == 0 {
		return big.NewInt(0)
	}

	if crr == 100 {
		result := big.NewInt(0).Mul(wantReceive, reserve)

		return result.Div(result, supply)
	}

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

// CalculateSaleReturn returns amount of BIP user will receive by depositing given amount of coins
// Return = reserve * (1 - (1 - sellAmount / supply) ^ (100 / crr))
func CalculateSaleReturn(supply *big.Int, reserve *big.Int, crr uint32, sellAmount *big.Int) *big.Int {
	// special case for 0 sell amount
	if sellAmount.Sign() == 0 {
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

	res := newFloat(0).Quo(tSellAmount, tSupply)      // sellAmount / supply
	res.Sub(newFloat(1), res)                         // (1 - sellAmount / supply)
	res = math.Pow(res, newFloat(100/(float64(crr)))) // (1 - sellAmount / supply) ^ (100 / crr)
	res.Sub(newFloat(1), res)                         // (1 - (1 - sellAmount / supply) ^ (1 / (crr / 100)))
	res.Mul(res, tReserve)                            // reserve * (1 - (1 - sellAmount / supply) ^ (1 / (crr / 100)))

	result, _ := res.Int(nil)

	return result
}

// CalculateSaleAmount is the reversed version of function CalculateSaleReturn
// Deposit = -(-1 + (-(wantReceive - reserve)/reserve)^(1/crr)) * supply
func CalculateSaleAmount(supply *big.Int, reserve *big.Int, crr uint32, wantReceive *big.Int) *big.Int {
	if wantReceive.Sign() == 0 {
		return big.NewInt(0)
	}

	if crr == 100 {
		ret := big.NewInt(0).Mul(wantReceive, supply)
		ret.Div(ret, reserve)

		return ret
	}

	tSupply := newFloat(0).SetInt(supply)
	tReserve := newFloat(0).SetInt(reserve)
	tWantReceive := newFloat(0).SetInt(wantReceive)

	res := newFloat(0).Sub(tWantReceive, tReserve)  // (wantReceive - reserve)
	res.Neg(res)                                    // -(wantReceive - reserve)
	res.Quo(res, tReserve)                          // -(wantReceive - reserve)/reserve
	res = math.Pow(res, newFloat(float64(crr)/100)) // (-(wantReceive - reserve)/reserve)^(1/crr)
	res.Add(res, newFloat(-1))                      // -1 + (-(wantReceive - reserve)/reserve)^(1/crr)
	res.Neg(res)                                    // -(-1 + (-(wantReceive - reserve)/reserve)^(1/crr))
	res.Mul(res, tSupply)                           // -(-1 + (-(wantReceive - reserve)/reserve)^(1/crr)) * supply

	result, _ := res.Int(nil)

	return result
}

func newFloat(x float64) *big.Float {
	return big.NewFloat(x).SetPrec(precision)
}
