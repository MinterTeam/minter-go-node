package formula

import (
	"github.com/ALTree/floatutils"
	"math/big"
	"minter/core/types"
	"strings"
)

var (
	One            = big.NewInt(1)
	MaxWeight      = uint(100)
	MinPrecision   = 32
	MaxPrecision   = 127
	Fixed1         = hexToBig("0x080000000000000000000000000000000")
	Fixed2         = hexToBig("0x100000000000000000000000000000000")
	Ln2Numerator   = hexToBig("0x3f80fe03f80fe03f80fe03f80fe03f8")
	Ln2Denominator = hexToBig("0x5b9de1d10bf4103d647b0955897ba80")
	maxExpArray    = make([]*big.Int, 128)
)

func initMaxExpArray() {
	maxExpArray[32] = hexToBig("0x1c35fedd14ffffffffffffffffffffffff")
	maxExpArray[33] = hexToBig("0x1b0ce43b323fffffffffffffffffffffff")
	maxExpArray[34] = hexToBig("0x19f0028ec1ffffffffffffffffffffffff")
	maxExpArray[35] = hexToBig("0x18ded91f0e7fffffffffffffffffffffff")
	maxExpArray[36] = hexToBig("0x17d8ec7f0417ffffffffffffffffffffff")
	maxExpArray[37] = hexToBig("0x16ddc6556cdbffffffffffffffffffffff")
	maxExpArray[38] = hexToBig("0x15ecf52776a1ffffffffffffffffffffff")
	maxExpArray[39] = hexToBig("0x15060c256cb2ffffffffffffffffffffff")
	maxExpArray[40] = hexToBig("0x1428a2f98d72ffffffffffffffffffffff")
	maxExpArray[41] = hexToBig("0x13545598e5c23fffffffffffffffffffff")
	maxExpArray[42] = hexToBig("0x1288c4161ce1dfffffffffffffffffffff")
	maxExpArray[43] = hexToBig("0x11c592761c666fffffffffffffffffffff")
	maxExpArray[44] = hexToBig("0x110a688680a757ffffffffffffffffffff")
	maxExpArray[45] = hexToBig("0x1056f1b5bedf77ffffffffffffffffffff")
	maxExpArray[46] = hexToBig("0x0faadceceeff8bffffffffffffffffffff")
	maxExpArray[47] = hexToBig("0x0f05dc6b27edadffffffffffffffffffff")
	maxExpArray[48] = hexToBig("0x0e67a5a25da4107fffffffffffffffffff")
	maxExpArray[49] = hexToBig("0x0dcff115b14eedffffffffffffffffffff")
	maxExpArray[50] = hexToBig("0x0d3e7a392431239fffffffffffffffffff")
	maxExpArray[51] = hexToBig("0x0cb2ff529eb71e4fffffffffffffffffff")
	maxExpArray[52] = hexToBig("0x0c2d415c3db974afffffffffffffffffff")
	maxExpArray[53] = hexToBig("0x0bad03e7d883f69bffffffffffffffffff")
	maxExpArray[54] = hexToBig("0x0b320d03b2c343d5ffffffffffffffffff")
	maxExpArray[55] = hexToBig("0x0abc25204e02828dffffffffffffffffff")
	maxExpArray[56] = hexToBig("0x0a4b16f74ee4bb207fffffffffffffffff")
	maxExpArray[57] = hexToBig("0x09deaf736ac1f569ffffffffffffffffff")
	maxExpArray[58] = hexToBig("0x0976bd9952c7aa957fffffffffffffffff")
	maxExpArray[59] = hexToBig("0x09131271922eaa606fffffffffffffffff")
	maxExpArray[60] = hexToBig("0x08b380f3558668c46fffffffffffffffff")
	maxExpArray[61] = hexToBig("0x0857ddf0117efa215bffffffffffffffff")
	maxExpArray[62] = hexToBig("0x07ffffffffffffffffffffffffffffffff")
	maxExpArray[63] = hexToBig("0x07abbf6f6abb9d087fffffffffffffffff")
	maxExpArray[64] = hexToBig("0x075af62cbac95f7dfa7fffffffffffffff")
	maxExpArray[65] = hexToBig("0x070d7fb7452e187ac13fffffffffffffff")
	maxExpArray[66] = hexToBig("0x06c3390ecc8af379295fffffffffffffff")
	maxExpArray[67] = hexToBig("0x067c00a3b07ffc01fd6fffffffffffffff")
	maxExpArray[68] = hexToBig("0x0637b647c39cbb9d3d27ffffffffffffff")
	maxExpArray[69] = hexToBig("0x05f63b1fc104dbd39587ffffffffffffff")
	maxExpArray[70] = hexToBig("0x05b771955b36e12f7235ffffffffffffff")
	maxExpArray[71] = hexToBig("0x057b3d49dda84556d6f6ffffffffffffff")
	maxExpArray[72] = hexToBig("0x054183095b2c8ececf30ffffffffffffff")
	maxExpArray[73] = hexToBig("0x050a28be635ca2b888f77fffffffffffff")
	maxExpArray[74] = hexToBig("0x04d5156639708c9db33c3fffffffffffff")
	maxExpArray[75] = hexToBig("0x04a23105873875bd52dfdfffffffffffff")
	maxExpArray[76] = hexToBig("0x0471649d87199aa990756fffffffffffff")
	maxExpArray[77] = hexToBig("0x04429a21a029d4c1457cfbffffffffffff")
	maxExpArray[78] = hexToBig("0x0415bc6d6fb7dd71af2cb3ffffffffffff")
	maxExpArray[79] = hexToBig("0x03eab73b3bbfe282243ce1ffffffffffff")
	maxExpArray[80] = hexToBig("0x03c1771ac9fb6b4c18e229ffffffffffff")
	maxExpArray[81] = hexToBig("0x0399e96897690418f785257fffffffffff")
	maxExpArray[82] = hexToBig("0x0373fc456c53bb779bf0ea9fffffffffff")
	maxExpArray[83] = hexToBig("0x034f9e8e490c48e67e6ab8bfffffffffff")
	maxExpArray[84] = hexToBig("0x032cbfd4a7adc790560b3337ffffffffff")
	maxExpArray[85] = hexToBig("0x030b50570f6e5d2acca94613ffffffffff")
	maxExpArray[86] = hexToBig("0x02eb40f9f620fda6b56c2861ffffffffff")
	maxExpArray[87] = hexToBig("0x02cc8340ecb0d0f520a6af58ffffffffff")
	maxExpArray[88] = hexToBig("0x02af09481380a0a35cf1ba02ffffffffff")
	maxExpArray[89] = hexToBig("0x0292c5bdd3b92ec810287b1b3fffffffff")
	maxExpArray[90] = hexToBig("0x0277abdcdab07d5a77ac6d6b9fffffffff")
	maxExpArray[91] = hexToBig("0x025daf6654b1eaa55fd64df5efffffffff")
	maxExpArray[92] = hexToBig("0x0244c49c648baa98192dce88b7ffffffff")
	maxExpArray[93] = hexToBig("0x022ce03cd5619a311b2471268bffffffff")
	maxExpArray[94] = hexToBig("0x0215f77c045fbe885654a44a0fffffffff")
	maxExpArray[95] = hexToBig("0x01ffffffffffffffffffffffffffffffff")
	maxExpArray[96] = hexToBig("0x01eaefdbdaaee7421fc4d3ede5ffffffff")
	maxExpArray[97] = hexToBig("0x01d6bd8b2eb257df7e8ca57b09bfffffff")
	maxExpArray[98] = hexToBig("0x01c35fedd14b861eb0443f7f133fffffff")
	maxExpArray[99] = hexToBig("0x01b0ce43b322bcde4a56e8ada5afffffff")
	maxExpArray[100] = hexToBig("0x019f0028ec1fff007f5a195a39dfffffff")
	maxExpArray[101] = hexToBig("0x018ded91f0e72ee74f49b15ba527ffffff")
	maxExpArray[102] = hexToBig("0x017d8ec7f04136f4e5615fd41a63ffffff")
	maxExpArray[103] = hexToBig("0x016ddc6556cdb84bdc8d12d22e6fffffff")
	maxExpArray[104] = hexToBig("0x015ecf52776a1155b5bd8395814f7fffff")
	maxExpArray[105] = hexToBig("0x015060c256cb23b3b3cc3754cf40ffffff")
	maxExpArray[106] = hexToBig("0x01428a2f98d728ae223ddab715be3fffff")
	maxExpArray[107] = hexToBig("0x013545598e5c23276ccf0ede68034fffff")
	maxExpArray[108] = hexToBig("0x01288c4161ce1d6f54b7f61081194fffff")
	maxExpArray[109] = hexToBig("0x011c592761c666aa641d5a01a40f17ffff")
	maxExpArray[110] = hexToBig("0x0110a688680a7530515f3e6e6cfdcdffff")
	maxExpArray[111] = hexToBig("0x01056f1b5bedf75c6bcb2ce8aed428ffff")
	maxExpArray[112] = hexToBig("0x00faadceceeff8a0890f3875f008277fff")
	maxExpArray[113] = hexToBig("0x00f05dc6b27edad306388a600f6ba0bfff")
	maxExpArray[114] = hexToBig("0x00e67a5a25da41063de1495d5b18cdbfff")
	maxExpArray[115] = hexToBig("0x00dcff115b14eedde6fc3aa5353f2e4fff")
	maxExpArray[116] = hexToBig("0x00d3e7a3924312399f9aae2e0f868f8fff")
	maxExpArray[117] = hexToBig("0x00cb2ff529eb71e41582cccd5a1ee26fff")
	maxExpArray[118] = hexToBig("0x00c2d415c3db974ab32a51840c0b67edff")
	maxExpArray[119] = hexToBig("0x00bad03e7d883f69ad5b0a186184e06bff")
	maxExpArray[120] = hexToBig("0x00b320d03b2c343d4829abd6075f0cc5ff")
	maxExpArray[121] = hexToBig("0x00abc25204e02828d73c6e80bcdb1a95bf")
	maxExpArray[122] = hexToBig("0x00a4b16f74ee4bb2040a1ec6c15fbbf2df")
	maxExpArray[123] = hexToBig("0x009deaf736ac1f569deb1b5ae3f36c130f")
	maxExpArray[124] = hexToBig("0x00976bd9952c7aa957f5937d790ef65037")
	maxExpArray[125] = hexToBig("0x009131271922eaa6064b73a22d0bd4f2bf")
	maxExpArray[126] = hexToBig("0x008b380f3558668c46c91c49a2f8e967b9")
	maxExpArray[127] = hexToBig("0x00857ddf0117efa215952912839f6473e6")
}

func init() {
	initMaxExpArray()
}

func hexToBig(hex string) *big.Int {
	ret, _ := big.NewInt(0).SetString(strings.TrimLeft(strings.TrimLeft(hex, "0x"), "0"), 16)

	return ret
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

	result := big.NewInt(0)
	var precision uint

	baseN := big.NewInt(0).Add(deposit, reserve)

	result, precision = power(baseN, reserve, crr, MaxWeight)

	temp := big.NewInt(0).Mul(supply, result)
	temp.Rsh(temp, precision)

	return temp.Sub(temp, supply)

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

	result := big.NewInt(0)
	var precision uint

	baseD := big.NewInt(0).Sub(supply, sellAmount)

	result, precision = power(supply, baseD, MaxWeight, crr)

	temp1 := big.NewInt(0).Mul(reserve, result)
	temp2 := big.NewInt(0).Lsh(reserve, precision)

	res := big.NewInt(0).Sub(temp1, temp2)
	res.Div(res, result)

	return res
}

func CalculateBuyDeposit(supply *big.Int, reserve *big.Int, crr uint, wantBuy *big.Int) *big.Int {

	tSupply := big.NewFloat(0).SetInt(supply)
	tReserve := big.NewFloat(0).SetInt(reserve)
	tWantBuy := big.NewFloat(0).SetInt(wantBuy)

	res := big.NewFloat(0).Sub(tWantBuy, tReserve)
	res.Mul(res, big.NewFloat(-1))
	res.Quo(res, tReserve)
	res = bigfloat.Pow(res, big.NewFloat(float64(crr)/100))
	res.Add(res, big.NewFloat(-1))
	res.Mul(res, big.NewFloat(-1))
	res.Mul(res, tSupply)

	result, _ := res.Int(nil)

	return result
}

func power(aBaseN *big.Int, aBaseD *big.Int, aExpN uint, aExpD uint) (*big.Int, uint) {
	lnBaseTimesExp := ln(aBaseN, aBaseD)
	lnBaseTimesExp.Mul(lnBaseTimesExp, big.NewInt(int64(aExpN)))
	lnBaseTimesExp.Div(lnBaseTimesExp, big.NewInt(int64(aExpD)))

	precision := findPositionInMaxExpArray(lnBaseTimesExp)

	return fixedExp(lnBaseTimesExp.Rsh(lnBaseTimesExp, uint(MaxPrecision)-precision), precision), precision
}

func ln(aNumerator *big.Int, aDenominator *big.Int) *big.Int {

	res := big.NewInt(0)
	x := big.NewInt(0).Mul(aNumerator, Fixed1)
	x.Div(x, aDenominator)

	if x.Cmp(Fixed2) != -1 {
		count := floorLog2(big.NewInt(0).Div(x, Fixed1))
		x.Rsh(x, count)
		res.Mul(big.NewInt(int64(count)), Fixed1)
	}

	if x.Cmp(Fixed1) == 1 {
		for i := MaxPrecision; i > 0; i -= 1 {
			x.Mul(x, x)
			x.Div(x, Fixed1)

			if x.Cmp(Fixed2) != -1 {
				x.Rsh(x, 1)
				res.Add(res, big.NewInt(1).Lsh(big.NewInt(1), uint(i-1)))
			}
		}
	}

	res.Mul(res, Ln2Numerator)
	res.Div(res, Ln2Denominator)

	return res
}

func floorLog2(aN *big.Int) uint {
	n := big.NewInt(0).Set(aN)
	res := 0

	if n.Cmp(big.NewInt(256)) == -1 {
		for n.Cmp(big.NewInt(1)) == 1 {
			n.Rsh(n, 1)
			res += 1
		}
	} else {
		for s := 128; s > 0; s >>= 1 {
			if n.Cmp(big.NewInt(1).Lsh(One, uint(s))) != -1 {
				n.Rsh(n, uint(s))
				res |= s
			}
		}
	}

	return uint(res)
}

func findPositionInMaxExpArray(x *big.Int) uint {
	lo := MinPrecision
	hi := MaxPrecision

	for lo+1 < hi {
		mid := (lo + hi) / 2

		if maxExpArray[mid].Cmp(x) != -1 {
			lo = mid
		} else {
			hi = mid
		}
	}

	if maxExpArray[hi].Cmp(x) != -1 {
		return uint(hi)
	}

	if maxExpArray[lo].Cmp(x) != -1 {
		return uint(lo)
	}

	return 0
}

func fixedExp(x *big.Int, precision uint) *big.Int {
	xi := big.NewInt(0).Set(x)
	res := big.NewInt(0)

	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x03442c4e6074a82f1797f72ac0000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0116b96f757c380fb287fd0e40000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0045ae5bdd5f0e03eca1ff4390000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000defabf91302cd95b9ffda50000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0002529ca9832b22439efff9b8000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000054f1cf12bd04e516b6da88000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000a9e39e257a09ca2d6db51000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0000012e066e7b839fa050c309000000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0000001e33d7d926c329a1ad1a800000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000002bee513bdb4a6b19b5f800000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000003a9316fa79b88eccf2a00000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000048177ebe1fa812375200000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000005263fe90242dcbacf00000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0000000000057e22099c030d94100000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000000057e22099c030d9410000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000000000052b6b54569976310000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000000000004985f67696bf748000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0000000000000003dea12ea99e498000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000000000000031880f2214b6e000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0000000000000000025bcff56eb36000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0000000000000000001b722e10ab1000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000000000000001317c70077000")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000000000000000000cba84aafa00")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000000000000000000082573a0a00")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000000000000000000005035ad900")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x0000000000000000000000002f881b00")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000000000000000000001b29340")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x000000000000000000000000000efc40")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000000000000000000000007fe0")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000000000000000000000000420")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000000000000000000000000021")))
	xi.Mul(xi, x)
	xi.Rsh(xi, precision)
	res.Add(res, big.NewInt(0).Mul(xi, hexToBig("0x00000000000000000000000000000001")))

	ret := big.NewInt(0).Div(res, hexToBig("0x688589cc0e9505e2f2fee5580000000"))
	ret.Add(ret, x)
	ret.Add(ret, big.NewInt(0).Lsh(One, precision))

	return ret
}
