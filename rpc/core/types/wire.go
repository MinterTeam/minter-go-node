package core_types

import (
	"github.com/tendermint/go-amino"
	"github.com/tendermint/go-crypto"
	types "minter/tmtypes"
)

func RegisterAmino(cdc *amino.Codec) {
	types.RegisterEventDatas(cdc)
	types.RegisterEvidences(cdc)
	crypto.RegisterAmino(cdc)
}
