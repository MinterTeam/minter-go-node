package api

func MinGasPrice() (uint64, error) {
	return uint64(blockchain.MinGasPrice()), nil
}
