package chain

import (
	"fmt"
	"math/big"
	"sync"

	"PhoenixOracle/lib/logger"
)

// To add a new chain to this file:
// 1. Add the global var in the vars list
// 2. Add the chain ID in the map in the init() function
// 3. Add a config set in configs.go

// Chain represents a blockchain with a unique Chain ID
type Chain struct {
	id      *big.Int
	config  ChainSpecificConfig
	logOnce sync.Once
}

func (c *Chain) setChainID(id int64) {
	c.id = big.NewInt(id)
}

func (c *Chain) ID() *big.Int {
	return c.id
}

func (c *Chain) Config() ChainSpecificConfig {
	if !c.config.set {
		c.logOnce.Do(func() {
			logger.Warnf("chain with ID %s does not have a chain-specific config, using fallback config instead", c.ID())
		})
		return FallbackConfig
	}
	return c.config
}

// IsArbitrum returns true if the chain is arbitrum
func (c *Chain) IsArbitrum() bool {
	return c.Config().Layer2Type == "Arbitrum"
}

// IsOptimism returns true if the chain is optimism
func (c *Chain) IsOptimism() bool {
	return c.Config().Layer2Type == "Optimism"
}

// IsL2 returns true if this chain is an L2 chain. Notably the block numbers
// used for log searching are different from calling block.number
func (c *Chain) IsL2() bool {
	return c.Config().Layer2Type != ""
}

var chains map[int64]*Chain
var (
	EthMainnet       = new(Chain)
	EthRinkeby       = new(Chain)
	EthGoerli        = new(Chain)
	EthKovan         = new(Chain)
	BSCMainnet       = new(Chain)
	BSCTestnet       = new(Chain)
	HecoMainnet      = new(Chain)
)

func init() {
	chains = make(map[int64]*Chain)

	chains[1] = EthMainnet
	chains[4] = EthRinkeby
	chains[5] = EthGoerli
	chains[42] = EthKovan
	chains[97] = BSCTestnet
	chains[56] = BSCMainnet
	chains[128] = HecoMainnet

	for id, chain := range chains {
		chain.setChainID(id)
	}

	setConfigs()
}

var chainsMu sync.Mutex

func ChainFromID(id *big.Int) *Chain {
	if !id.IsInt64() {
		panic(fmt.Sprintf("chain IDs larger than the max 64 bit integer are not currently supported, got: %s", id.String()))
	}
	chainsMu.Lock()
	defer chainsMu.Unlock()
	chain, exists := chains[id.Int64()]
	if exists {
		return chain
	}
	logger.Warnf("Chain ID %s is not known, falling back to generic chain", id)
	chain = new(Chain)
	chain.id = id
	chains[id.Int64()] = chain
	return chain
}
