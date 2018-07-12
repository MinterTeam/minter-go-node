package math

import (
	"math"
	"math/big"
)

// https://github.com/ALTree/bigfloat

// Log returns a big.Float representation of the natural logarithm of
// z. Precision is the same as the one of the argument. The function
// panics if z is negative, returns -Inf when z = 0, and +Inf when z =
// +Inf
func Log(z *big.Float) *big.Float {

	// panic on negative z
	if z.Sign() == -1 {
		panic("Log: argument is negative")
	}

	// Log(0) = -Inf
	if z.Sign() == 0 {
		return big.NewFloat(math.Inf(-1)).SetPrec(z.Prec())
	}

	prec := z.Prec() + 64 // guard digits

	one := big.NewFloat(1).SetPrec(prec)
	two := big.NewFloat(2).SetPrec(prec)
	four := big.NewFloat(4).SetPrec(prec)

	// Log(1) = 0
	if z.Cmp(one) == 0 {
		return big.NewFloat(0).SetPrec(z.Prec())
	}

	// Log(+Inf) = +Inf
	if z.IsInf() {
		return big.NewFloat(math.Inf(+1)).SetPrec(z.Prec())
	}

	x := new(big.Float).SetPrec(prec)

	// if 0 < z < 1 we compute log(z) as -log(1/z)
	var neg bool
	if z.Cmp(one) < 0 {
		x.Quo(one, z)
		neg = true
	} else {
		x.Set(z)
	}

	// We scale up x until x >= 2**(prec/2), and then we'll be allowed
	// to use the AGM formula for Log(x).
	//
	// Double x until the condition is met, and keep track of the
	// number of doubling we did (needed to scale back later).

	lim := new(big.Float)
	lim.SetMantExp(two, int(prec/2))

	k := 0
	for x.Cmp(lim) < 0 {
		x.Mul(x, x)
		k++
	}

	// Compute the natural log of x using the fact that
	//     log(x) = π / (2 * AGM(1, 4/x))
	// if
	//     x >= 2**(prec/2),
	// where prec is the desired precision (in bits)
	pi := pi(prec)
	agm := agm(one, x.Quo(four, x)) // agm = AGM(1, 4/x)

	x.Quo(pi, x.Mul(two, agm)) // reuse x, we don't need it

	if neg {
		x.Neg(x)
	}

	// scale the result back multiplying by 2**-k
	// reuse lim to reduce allocations.
	x.Mul(x, lim.SetMantExp(one, -k))

	return x.SetPrec(z.Prec())
}
