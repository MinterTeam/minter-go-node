package swap

type Route struct {
	Pairs    []*PairTrade
	Path     []Token
	Input    Token
	Output   Token
	MidPrice Price
}

func NewRoute(pairs []*PairTrade, input Token, output *Token) Route {
	path := []Token{input}
	for i, pair := range pairs {
		currentInput, currentOutput := path[i], pair.Token0
		if currentInput.IsEqual(pair.Token0.Token) {
			currentOutput = pair.Token1
		}

		path = append(path, currentOutput.Token)
	}

	if output == nil {
		output = new(Token)
		*output = path[len(path)-1]
	}

	route := Route{
		Pairs:  pairs,
		Path:   path,
		Input:  input,
		Output: *output,
	}

	route.MidPrice = NewPriceFromRoute(route)

	return route
}
