package types

type ChainID byte

const (
	ChainTestnet = 0x02
	ChainMainnet = 0x01

	CurrentChainID = ChainTestnet
)

func GetBaseCoin() CoinSymbol {
	return getBaseCoin(CurrentChainID)
}

func getBaseCoin(chainID ChainID) CoinSymbol {
	var coin CoinSymbol

	switch chainID {
	case ChainMainnet:
		copy(coin[:], "BIP")
	case ChainTestnet:
		copy(coin[:], "MNT")
	}

	coin[4] = byte(0)

	return coin
}
