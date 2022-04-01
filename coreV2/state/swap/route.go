package swap

import "github.com/MinterTeam/minter-go-node/coreV2/types"

type Route struct {
	Pairs  []EditableChecker
	Path   []types.CoinID
	Input  types.CoinID
	Output types.CoinID
}

func NewRoute(pairs []EditableChecker, input types.CoinID, output *types.CoinID) Route {
	path := []types.CoinID{input}
	for i, pair := range pairs {
		currentInput, currentOutput := path[i], pair.Coin0()
		if currentInput == pair.Coin0() {
			currentOutput = pair.Coin1()
		}

		path = append(path, currentOutput)
	}

	var outputCoinID types.CoinID
	if output == nil {
		outputCoinID = path[len(path)-1]
	}

	route := Route{
		Pairs:  pairs,
		Path:   path,
		Input:  input,
		Output: outputCoinID,
	}

	return route
}
