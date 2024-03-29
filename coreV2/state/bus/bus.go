package bus

import eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"

type Bus struct {
	coins       Coins
	app         App
	accounts    Accounts
	candidates  Candidates
	frozenfunds FrozenFunds
	halts       HaltBlocks
	waitlist    WaitList
	events      eventsdb.IEventsDB
	checker     Checker
	validators  Validators
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

func (b *Bus) SetValidators(validators Validators) {
	b.validators = validators
}

func (b *Bus) Accounts() Accounts {
	return b.accounts
}

func (b *Bus) Validators() Validators {
	return b.validators
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

func (b *Bus) SetHaltBlocks(halts HaltBlocks) {
	b.halts = halts
}

func (b *Bus) Halts() HaltBlocks {
	return b.halts
}

func (b *Bus) SetWaitList(waitList WaitList) {
	b.waitlist = waitList
}

func (b *Bus) WaitList() WaitList {
	return b.waitlist
}

func (b *Bus) SetEvents(events eventsdb.IEventsDB) {
	b.events = events
}

func (b *Bus) Events() eventsdb.IEventsDB {
	return b.events
}

func (b *Bus) SetChecker(checker Checker) {
	b.checker = checker
}

func (b *Bus) Checker() Checker {
	return b.checker
}
