package fluxmonitor

import (
	"time"

	"PhoenixOracle/core/assets"
)

type Config struct {
	DefaultHTTPTimeout             time.Duration
	FlagsContractAddress           string
	MinContractPayment             *assets.Phb
	EvmGasLimit                    uint64
	EvmMaxQueuedTransactions       uint64
	FMDefaultTransactionQueueDepth uint32
}

func (c *Config) MinimumPollingInterval() time.Duration {
	return c.DefaultHTTPTimeout
}
