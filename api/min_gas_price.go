package api

func MinGasPrice() (uint32, error) {
	return blockchain.MinGasPrice(), nil
}
