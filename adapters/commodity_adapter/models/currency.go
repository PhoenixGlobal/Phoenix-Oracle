package models

// Currency model holds the data about the currency.
type Currency struct {

	// Name is the currency code of the currency.
	Name string `json:"Name"`
	// The name of the country where the currency came from.
	Country string `json:"Country"`
	// Fullname of the currency.
	Description string `json:"Description"`

	// Latest currency change in percentages.
	Change float32 `json:"Change"`
	// Exchange rate to USD.
	RateUSD float32 `json:"RateUSD"`
	// Last time updated.
	UpdatedAt string `json:"UpdatedAt"`
}

// ExchangeRate holds the exchange rate.
type ExchangeRate struct {
	// Exchange rate.
	Rate float32 `json:"Rate"`
}
