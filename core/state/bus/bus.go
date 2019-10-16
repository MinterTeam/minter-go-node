package bus

type Bus struct {
	coins Coins
	app   App
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
