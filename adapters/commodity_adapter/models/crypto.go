package models

// Crypto model holds the data about the commodity.
type Crypto struct {

	// Name of the currency.
	Name string `json:"Name"`
	// Symbol of the currency.
	Symbol string `json:"Symbol"`

	// Market capitalization in USD.
	MarketCapUSD float64 `json:"MarketCapUSD"`
	// Cuurrent value of the currency.
	Price float64 `json:"price"`
	// The total value of the currently available amount of the currencies.
	CirculatingSupply float64 `json:"CirculatingSupply"`
	// Mineable indicates if the currency is mineable.
	Mineable bool `json:"Mineable"`
	// Volume is the total value of the currencies in USD which was traded in the last 24 hours.
	Volume float64 `json:"Volume"`

	// The percentage changes in the last hour/day/week.
	ChangeHour string `json:"ChangeHour"`
	ChangeDay  string `json:"ChangeDay"`
	ChangeWeek string `json:"ChangeWeek"`
}
