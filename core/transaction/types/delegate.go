package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"math/big"
)

type DelegateData struct {
	PubKey []byte
	Coin   types.CoinSymbol
	Stake  *big.Int
}

func (s DelegateData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string
		Coin   types.CoinSymbol
		Stake  string
	}{
		PubKey: fmt.Sprintf("Mp%x", s.PubKey),
		Coin:   s.Coin,
		Stake:  s.Stake.String(),
	})
}

func (s DelegateData) String() string {
	return fmt.Sprintf("DELEGATE ubkey:%s ",
		hexutil.Encode(s.PubKey[:]))
}

func (s DelegateData) Gas() int64 {
	return commissions.DelegateTx
}
