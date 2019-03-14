package main

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/appdb"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
)

func main() {
	err := common.EnsureDir(utils.GetMinterHome()+"/config", 0777)
	if err != nil {
		panic(err)
	}

	ldb, err := db.NewGoLevelDB("state", utils.GetMinterHome()+"/data")
	if err != nil {
		panic(err)
	}

	applicationDB := appdb.NewAppDB()
	currentState, err := state.New(applicationDB.GetLastHeight(), ldb)
	if err != nil {
		panic(err)
	}

	jsonBytes, err := json.MarshalIndent(currentState.Export(), "", "	")
	if err != nil {
		panic(err)
	}

	println(string(jsonBytes))
}
