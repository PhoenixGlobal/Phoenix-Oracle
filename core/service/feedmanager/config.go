package feedmanager

import (
	"math/big"
	"time"

	"PhoenixOracle/core/chain"
	"PhoenixOracle/db/models"
)

type Config interface {
	Chain() *chain.Chain
	ChainID() *big.Int
	Dev() bool
	FeatureOffchainReporting() bool
	DefaultHTTPTimeout() models.Duration
	OCRBlockchainTimeout() time.Duration
	OCRContractConfirmations() uint16
	OCRContractPollInterval() time.Duration
	OCRContractSubscribeInterval() time.Duration
	OCRContractTransmitterTransmitTimeout() time.Duration
	OCRDatabaseTimeout() time.Duration
	OCRObservationTimeout() time.Duration
	OCRObservationGracePeriod() time.Duration
}
