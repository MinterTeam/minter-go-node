package api

import (
	"net/http"
	"github.com/gorilla/websocket"
	"log"
	"minter/core/state"
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
