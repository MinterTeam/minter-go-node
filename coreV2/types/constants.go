package types

// ChainID is ID of the network (1 - mainnet, 2 - testnet)
type ChainID byte

const (
	// ChainMainnet is mainnet chain ID of the network
	ChainMainnet ChainID = 0x01
	// ChainTestnet is mainnet chain ID of the network
	ChainTestnet ChainID = 0x02
)

const unbondPeriod = 518400

func GetIncreasedRewardsPeriod() uint64 {
	return GetIncreasedRewardsPeriodWithChain(CurrentChainID)
}
func GetUnbondPeriod() uint64 {
	return GetUnbondPeriodWithChain(CurrentChainID)
}
func GetMovePeriod() uint64 {
	return GetMovePeriodWithChain(CurrentChainID)
}

func GetExpireOrdersPeriod() uint64 {
	return GetExpireOrdersPeriodWithChain(CurrentChainID)
}

const yearX3 = 21024000    // 3y = 94608000sec/4.5
const mounth = week * 4    // 1m
const week = day * 7       // 1w
const weekCorrect = 134400 // 1w
const day = 8640 * 2       // 1d
const m15 = 518400 / 2920  // 15m

func GetExpireOrdersPeriodWithChain(chain ChainID) uint64 {
	if chain == ChainTestnet {
		return day * 5
	}
	return mounth
}

func GetIncreasedRewardsPeriodWithChain(chain ChainID) uint64 {
	if chain == ChainTestnet {
		return day * 2
	}
	return yearX3
}
func GetUnbondPeriodWithChain(chain ChainID) uint64 {
	if chain == ChainTestnet {
		return m15 * 3
	}
	return 518400
}
func GetMovePeriodWithChain(chain ChainID) uint64 {
	if chain == ChainTestnet {
		return m15
	}
	return weekCorrect
}

const jailPeriod = day

func GetJailPeriod() uint64 {
	return GetJailPeriodWithChain(CurrentChainID)
}

func GetJailPeriodWithChain(chain ChainID) uint64 {
	if chain == ChainTestnet {
		return m15 * 2
	}
	return jailPeriod
}

// CurrentChainID is current ChainID of the network
var CurrentChainID = ChainMainnet

var (
	coinTestnet = StrToCoinSymbol("MNT")
	coinMainnet = StrToCoinSymbol("BIP")
)

// GetBaseCoin returns the coin symbol of the current ChainID
func GetBaseCoin() CoinSymbol {
	return getBaseCoin(CurrentChainID)
}

// GetBaseCoinID returns ID of base coin
func GetBaseCoinID() CoinID {
	return BasecoinID
}

func getBaseCoin(chainID ChainID) CoinSymbol {
	switch chainID {
	case ChainMainnet:
		return coinMainnet
	case ChainTestnet:
		return coinTestnet
	}

	panic("Unknown chain id")
}
