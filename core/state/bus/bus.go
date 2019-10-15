package bus

import (
	"context"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
)

const App = "app"
const FrozenFunds = "frozenfunds"
const Coins = "coins"

type Event struct {
	From string
	To   string
	Data interface{}
}

type doneEvent struct {
	Data interface{}
}

type Bus struct {
	ps *pubsub.Server
}

func NewBus() *Bus {
	ps := pubsub.NewServer()
	if err := ps.Start(); err != nil {
		panic(err)
	}

	return &Bus{
		ps: ps,
	}
}

func (b *Bus) SendEvent(from, to string, data interface{}) interface{} {
	err := b.ps.PublishWithEvents(context.TODO(), Event{
		From: from,
		To:   to,
		Data: data,
	}, map[string][]string{
		"to":   {to},
		"from": {from},
	})
	if err != nil {
		panic(err)
	}

	return b.waitDoneEvent(to, from)
}

func (b *Bus) ListenEvents(to string, events chan Event) {
	s, err := b.ps.SubscribeUnbuffered(context.TODO(), to, query.MustParse("to='"+to+"'"))
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case msg := <-s.Out():
				events <- Event{
					From: msg.Events()["from"][0],
					To:   msg.Events()["to"][0],
					Data: msg.Data(),
				}
			}
		}
	}()
}

func (b *Bus) waitDoneEvent(from, to string) interface{} {
	s, err := b.ps.SubscribeUnbuffered(context.TODO(), "bus", query.MustParse("from='"+from+"' AND to='"+to+"'"))
	if err != nil {
		panic(err)
	}

	for {
		select {
		case msg := <-s.Out():
			switch msg.Data().(type) {
			case doneEvent:
				return msg.Data().(doneEvent).Data
			}
		}
	}
}

func (b *Bus) SendDone(from, to string, data interface{}) {
	b.SendEvent(from, to, doneEvent{
		Data: data,
	})
}
