package api

import (
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"github.com/tendermint/tendermint/rpc/core/types"
)

func SendTransaction(tx []byte) (*core_types.ResultBroadcastTx, error) {
	result, err := client.BroadcastTxSync(tx)
	if err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, rpctypes.TxError{
			Code: result.Code,
			Log:  result.Log,
		}
	}

	return result, nil
}
