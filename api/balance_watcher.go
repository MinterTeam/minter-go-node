package api

import (
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/gorilla/websocket"
	"math/big"
	"net/http"
)

var clients = make(map[*websocket.Conn]bool)
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func init() {
	go handleBalanceChanges()
}

func GetBalanceWatcher(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err.Error())
	}

	clients[ws] = true
}

func handleBalanceChanges() {
	for {
		handleBalanceChange(<-state.BalanceChangeChan)
	}
}

func handleBalanceChange(msg state.BalanceChangeStruct) {

	defer func() {
		if r := recover(); r != nil {
			log.Error("Error in balance change handler")
		}
	}()

	balanceInBasecoin := big.NewInt(0)

	if msg.Coin == types.GetBaseCoin() {
		balanceInBasecoin = msg.Balance
	} else {
		sCoin := minter.GetBlockchain().CurrentState().GetStateCoin(msg.Coin)

		if sCoin != nil {
			balanceInBasecoin = formula.CalculateSaleReturn(sCoin.Data().Volume, sCoin.Data().ReserveBalance, sCoin.Data().Crr, msg.Balance)
		}
	}

	msg.BalanceInBasecoin = balanceInBasecoin

	for client := range clients {
		err := client.WriteJSON(msg)
		if err != nil {
			log.Info("ws error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}
