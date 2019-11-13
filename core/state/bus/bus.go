package bus

import eventsdb "github.com/MinterTeam/events-db"

type Bus struct {
	coins       Coins
	app         App
	accounts    Accounts
	candidates  Candidates
	frozenfunds FrozenFunds
	events      eventsdb.IEventsDB
}

func NewBus() *Bus {
	return &Bus{}
}

func (b *Bus) SetCoins(coins Coins) {
	b.coins = coins
}

func (b *Bus) Coins() Coins {
	return b.coins
}

func (b *Bus) SetApp(app App) {
	b.app = app
}

func (b *Bus) App() App {
	return b.app
}

func (b *Bus) SetAccounts(accounts Accounts) {
	b.accounts = accounts
}

func (b *Bus) Accounts() Accounts {
	return b.accounts
}

func (b *Bus) SetCandidates(candidates Candidates) {
	b.candidates = candidates
}

func (b *Bus) Candidates() Candidates {
	return b.candidates
}

func (b *Bus) SetFrozenFunds(frozenfunds FrozenFunds) {
	b.frozenfunds = frozenfunds
}

func (b *Bus) FrozenFunds() FrozenFunds {
	return b.frozenfunds
}

func (b *Bus) SetEvents(events eventsdb.IEventsDB) {
	b.events = events
}

func (b *Bus) Events() eventsdb.IEventsDB {
	return b.events
}
