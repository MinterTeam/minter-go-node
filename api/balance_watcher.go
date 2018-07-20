package api

import (
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/gorilla/websocket"
	"log"
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
		log.Fatal(err)
	}

	clients[ws] = true
}

func handleBalanceChanges() {
	for {
		msg := <-state.BalanceChangeChan

		var balanceInBasecoin *big.Int

		if msg.Coin == types.GetBaseCoin() {
			balanceInBasecoin = msg.Balance
		} else {
			sCoin := minter.GetBlockchain().CurrentState().GetStateCoin(msg.Coin).Data()
			balanceInBasecoin = formula.CalculateSaleReturn(sCoin.Volume, sCoin.ReserveBalance, sCoin.Crr, msg.Balance)
		}

		msg.BalanceInBasecoin = balanceInBasecoin

		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("ws error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
