package types

type ChainID byte

const (
	ChainTestnet = 0x02
	ChainMainnet = 0x01

	CurrentChainID = ChainMainnet
)

var (
	coinTestnet = StrToCoinSymbol("MNT")
	coinMainnet = StrToCoinSymbol("BIP")
)

func GetBaseCoin() CoinSymbol {
	return getBaseCoin(CurrentChainID)
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
