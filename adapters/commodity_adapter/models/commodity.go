package models

// Commodity model holds the data about the commodity.
type Commodity struct {

	// Name of the commodity.
	Name string `json:"Name"`

	// Currency price of the commodity.
	Price float32 `json:"Price"`
	// The currency of the price.
	Currency string `json:"Currency"`
	// The weight for which the price of the commodity is  determined.
	WeightUnit string `json:"Weight_unit"`

	// Last change in percentages.
	ChangeP float32 `json:"ChangeP"`
	// Last change in a float.
	ChangeN float32 `json:"ChangeN"`
	// Last time updated (Unix time).
	LastUpdate int64 `json:"LastUpdate"`
}
