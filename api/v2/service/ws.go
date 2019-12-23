package service

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"log"
	"net/http"
	"time"
)

const (
	SubscribeTimeout = 5 * time.Second
)

var (
	upgrader = websocket.Upgrader{}
)

// Subscribe for events via WebSocket.
func (s *Service) Subscribe(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {

	query := r.URL.Query().Get("query")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	addr := conn.RemoteAddr().String()

	if s.client.NumClients() >= s.minterCfg.RPC.MaxSubscriptionClients {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("max_subscription_clients %d reached", s.minterCfg.RPC.MaxSubscriptionClients)))
		return
	} else if s.client.NumClientSubscriptions(addr) >= s.minterCfg.RPC.MaxSubscriptionsPerClient {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("max_subscriptions_per_client %d reached", s.minterCfg.RPC.MaxSubscriptionsPerClient)))
		return
	}

	s.client.Logger.Info("Subscribe to query", "remote", addr, "query", query)

	subCtx, cancel := context.WithTimeout(r.Context(), SubscribeTimeout)
	defer cancel()

	sub, err := s.client.Subscribe(subCtx, addr, query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	done := make(chan struct{})

	closeErr := make(chan error)
	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				closeErr <- err
				return
			}
		}
	}()

	go func() {
		defer close(done)
		for {
			select {
			case msg, ok := <-sub:
				if !ok {
					if err := conn.WriteMessage(websocket.CloseMessage, []byte("subscription was cancelled")); err != nil {
						s.client.Logger.Error(err.Error())
					}
					return
				}
				resultEvent := &ctypes.ResultEvent{Query: query, Data: msg.Data, Events: msg.Events}
				if err := conn.WriteJSON(resultEvent); err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						s.client.Logger.Error(err.Error())
					}
					return
				}
			case <-closeErr:
				return
			}
		}
	}()

	<-done
	if err := conn.Close(); err != nil {
		s.client.Logger.Error(err.Error())
	}
	if err := s.client.UnsubscribeAll(r.Context(), addr); err != nil {
		s.client.Logger.Error(err.Error())
	}
}
