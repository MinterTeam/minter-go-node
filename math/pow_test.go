package math_test

import (
	"fmt"
	bigfloat "github.com/MinterTeam/minter-go-node/math"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func TestPow(t *testing.T) {
	for _, test := range []struct {
		z, w string
		want string
	}{
		{"1.5", "1.5", "1.8371173070873835736479630560294185439744606104925025963245194254382202830929862699048945748284801761139459509199606418436441490948783180062193379634279589146216845606457574284357225789531838276676109830092400181402243325144092030253566067045309391758849310432709781082027026621306513787250611923558785098172755465204952231278685708006003328040156619"},
		{"2", "1.5", "2.8284271247461900976033774484193961571393437507538961463533594759814649569242140777007750686552831454700276924618245940498496721117014744252882429941998716628264453318550111855115999010023055641211429402191199432119405490691937240294570348372817783972191046584609686174286429016795252072559905028159793745067930926636176592812412305167047901094915006"},

		{"1.5", "-1.5", "0.54433105395181735515495201660130919821465499570148225076282057050021341721273667256441320735658671884857657805035870869441308121329727940925017421138606190062864727722837257138836224561575817116077362459533037574525165407834346756306862420874990790396590549430251203206006004803871151962224035329063066957548905082088747351936846542240009860859723315"},
		{"2", "-1.5", "0.35355339059327376220042218105242451964241796884423701829416993449768311961552675971259688358191039318375346155772807425623120901396268430316103037427498395785330566648187639818894998762528819551514286752738999290149256863364921550368212935466022229965238808230762107717858036270994065090699881285199742181334913658295220741015515381458809876368643757"},
	} {
		for _, prec := range []uint{24, 53, 64, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000} {
			want := new(big.Float).SetPrec(prec)
			want.Parse(test.want, 10)

			z := new(big.Float).SetPrec(prec)
			z.Parse(test.z, 10)
			w := new(big.Float).SetPrec(prec)
			w.Parse(test.w, 10)

			x := bigfloat.Pow(z, w)

			if x.Cmp(want) != 0 {
				t.Errorf("prec = %d, Pow(%v, %v) =\ngot  %g;\nwant %g", prec, test.z, test.w, x, want)
			}
		}
	}
}

func TestPowIntegers(t *testing.T) {
	for _, test := range []struct {
		z, w string
		want string
	}{
		{"2", "5", "32"},
		{"2", "10", "1024"},
		{"2", "64", "18446744073709551616"},

		{"2", "-5", "0.03125"},
		{"2", "-10", "0.0009765625"},
		{"2", "-64", "5.42101086242752217003726400434970855712890625e-20"},

		{"1.5", "8", "25.62890625"},
	} {
		for _, prec := range []uint{24, 53, 64, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000} {
			want := new(big.Float).SetPrec(prec)
			want.Parse(test.want, 10)

			z := new(big.Float).SetPrec(prec)
			z.Parse(test.z, 10)
			w := new(big.Float).SetPrec(prec)
			w.Parse(test.w, 10)

			x := bigfloat.Pow(z, w)

			if x.Cmp(want) != 0 {
				t.Errorf("prec = %d, Pow(%v, %v) =\ngot  %g;\nwant %g", prec, test.z, test.w, x, want)
			}
		}
	}
}

func testPowFloat64(scale float64, nTests int, t *testing.T) {
	for i := 0; i < nTests; i++ {
		r1 := math.Abs(rand.Float64() * scale) // base always > 0
		r2 := rand.Float64() * scale

		z := big.NewFloat(r1).SetPrec(53)
		w := big.NewFloat(r2).SetPrec(53)

		x64, acc := bigfloat.Pow(z, w).Float64()

		want := math.Pow(r1, r2)

		// Unfortunately, the Go math.Pow function is not completely
		// accurate, so it doesn't make sense to require 100%
		// compatibility with it, since it happens that math.Pow
		// returns a result with the last bit off (same as math.Log).
		//
		// Just require a relative error smaller than 1e-14.
		if math.Abs(x64-want)/want > 1e-14 || acc != big.Exact {
			t.Errorf("Pow(%g, %g) =\n got %g (%s);\nwant %g (Exact)", z, w, x64, acc, want)
		}
	}
}

func TestPowFloat64Small(t *testing.T) {
	testPowFloat64(-100, 1e3, t)
	testPowFloat64(-10, 1e3, t)
	testPowFloat64(-1, 1e3, t)
}

func TestPowFloat64Medium(t *testing.T) {
	testPowFloat64(0.1, 4e3, t)
	testPowFloat64(1, 4e3, t)
}

func TestPowFloat64Big(t *testing.T) {
	testPowFloat64(10, 4e3, t)
	testPowFloat64(100, 4e3, t)
}

func TestPowSpecialValues(t *testing.T) {
	for _, f := range []struct {
		z, w float64
	}{
		{2, +0.0},
		{2, -0.0},
		{4.2, 1.0},
		{math.Inf(+1), 2.0},
	} {
		z := big.NewFloat(f.z).SetPrec(53)
		w := big.NewFloat(f.w).SetPrec(53)
		x64, acc := bigfloat.Pow(z, w).Float64()
		want := math.Pow(f.z, f.w)
		if x64 != want || acc != big.Exact {
			t.Errorf("Pow(%g, %g) =\n got %g (%s);\nwant %g (Exact)", f.z, f.w, x64, acc, want)
		}
	}
}

// ---------- Benchmarks ----------

func BenchmarkPowInt(b *testing.B) {
	z := big.NewFloat(2).SetPrec(1e5)
	w := big.NewFloat(50).SetPrec(1e5)
	_ = bigfloat.Pow(z, w) // fill pi cache before benchmarking

	for _, prec := range []uint{1e2, 1e3, 1e4, 1e5} {
		z = big.NewFloat(2).SetPrec(prec)
		w = big.NewFloat(50).SetPrec(prec)
		b.Run(fmt.Sprintf("%v", prec), func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				bigfloat.Pow(z, w)
			}
		})
	}
}

func BenchmarkPow(b *testing.B) {
	z := big.NewFloat(2).SetPrec(1e5)
	w := big.NewFloat(1.5).SetPrec(1e5)
	_ = bigfloat.Pow(z, w) // fill pi cache before benchmarking

	for _, prec := range []uint{1e2, 1e3, 1e4, 1e5} {
		z = big.NewFloat(2).SetPrec(prec)
		w = big.NewFloat(1.5).SetPrec(prec)
		b.Run(fmt.Sprintf("%v", prec), func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				bigfloat.Pow(z, w)
			}
		})
	}
}
