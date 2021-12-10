package presenters

import (
	"time"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/db/models"
)

type BridgeResource struct {
	JAID
	Name          string `json:"name"`
	URL           string `json:"url"`
	Confirmations uint32 `json:"confirmations"`
	IncomingToken          string       `json:"incomingToken,omitempty"`
	OutgoingToken          string       `json:"outgoingToken"`
	MinimumContractPayment *assets.Phb `json:"minimumContractPayment"`
	CreatedAt              time.Time    `json:"createdAt"`
}

func (r BridgeResource) GetName() string {
	return "bridges"
}

func NewBridgeResource(b models.BridgeType) *BridgeResource {
	return &BridgeResource{
		JAID:                   NewJAID(b.Name.String()),
		Name:                   b.Name.String(),
		URL:                    b.URL.String(),
		Confirmations:          b.Confirmations,
		OutgoingToken:          b.OutgoingToken,
		MinimumContractPayment: b.MinimumContractPayment,
		CreatedAt:              b.CreatedAt,
	}
}
