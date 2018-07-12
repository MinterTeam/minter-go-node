package math

import (
	"math"
	"math/big"
)

// https://github.com/ALTree/bigfloat

// Exp returns a big.Float representation of exp(z). Precision is
// the same as the one of the argument. The function returns +Inf
// when z = +Inf, and 0 when z = -Inf.
func ExpFloat(z *big.Float) *big.Float {

	// exp(0) == 1
	if z.Sign() == 0 {
		return big.NewFloat(1).SetPrec(z.Prec())
	}

	// Exp(+Inf) = +Inf
	if z.IsInf() && z.Sign() > 0 {
		return big.NewFloat(math.Inf(+1)).SetPrec(z.Prec())
	}

	// Exp(-Inf) = 0
	if z.IsInf() && z.Sign() < 0 {
		return big.NewFloat(0).SetPrec(z.Prec())
	}

	guess := new(big.Float)

	// try to get initial estimate using IEEE-754 math
	zf, _ := z.Float64()
	if zfs := math.Exp(zf); zfs == math.Inf(+1) || zfs == 0 {
		// too big or too small for IEEE-754 math,
		// perform argument reduction using
		//     e^{2z} = (e^z)²
		halfZ := new(big.Float).Mul(z, big.NewFloat(0.5))
		halfExp := ExpFloat(halfZ.SetPrec(z.Prec() + 64))
		return new(big.Float).Mul(halfExp, halfExp).SetPrec(z.Prec())
	} else {
		// we got a nice IEEE-754 estimate
		guess.SetFloat64(zfs)
	}

	// f(t)/f'(t) = t*(log(t) - z)
	f := func(t *big.Float) *big.Float {
		x := new(big.Float)
		x.Sub(Log(t), z)
		return x.Mul(x, t)
	}

	x := newton(f, guess, z.Prec())

	return x
}
