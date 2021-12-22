package presenters

import (
	"time"

	"PhoenixOracle/core/service/feedmanager"
	"PhoenixOracle/util/crypto"
	"gopkg.in/guregu/null.v4"
)

type FeedsManagerResource struct {
	JAID
	Name                   string           `json:"name"`
	URI                    string           `json:"uri"`
	PublicKey              crypto.PublicKey `json:"publicKey"`
	JobTypes               []string         `json:"jobTypes"`
	IsBootstrapPeer        bool             `json:"isBootstrapPeer"`
	BootstrapPeerMultiaddr null.String      `json:"bootstrapPeerMultiaddr"`
	IsConnectionActive     bool             `json:"isConnectionActive"`
	CreatedAt              time.Time        `json:"createdAt"`
}

func (r FeedsManagerResource) GetName() string {
	return "feeds_managers"
}

func NewFeedsManagerResource(ms feedmanager.FeedsManager) *FeedsManagerResource {
	return &FeedsManagerResource{
		JAID:                   NewJAIDInt64(ms.ID),
		Name:                   ms.Name,
		URI:                    ms.URI,
		PublicKey:              ms.PublicKey,
		JobTypes:               ms.JobTypes,
		IsBootstrapPeer:        ms.IsOCRBootstrapPeer,
		BootstrapPeerMultiaddr: ms.OCRBootstrapPeerMultiaddr,
		IsConnectionActive:     ms.IsConnectionActive,
		CreatedAt:              ms.CreatedAt,
	}
}

func NewFeedsManagerResources(mss []feedmanager.FeedsManager) []FeedsManagerResource {
	rs := []FeedsManagerResource{}

	for _, ms := range mss {
		rs = append(rs, *NewFeedsManagerResource(ms))
	}

	return rs
}
