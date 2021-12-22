package store

import (
	"github.com/caarlos0/env"
	homedir "github.com/mitchellh/go-homedir"
	"log"
	"math/big"
	"os"
	"path"
)

type Config struct {
	RootDir           string     `env:"ROOT" envDefault:"~/.phoenix"`
	BasicAuthUsername string	 `env:"USERNAME" envDefault:"phoenix"`
	BasicAuthPassword string	 `env:"PASSWORD" envDefault:"p@ssword"`
	EthereumURL       string     `env:"ETHEREUM_URL" envDefault:"http://localhost:8545"`
	ChainID           int64      `env:"ETHEREUM_CHAIN_ID" envDefault:"0"`
	EthMinConfirmations        uint64     `env:"ETH_MIN_CONFIRMATIONS" envDefault:"12"`
	EthGasBumpWei      *big.Int `env:"ETH_GAS_BUMP_GWEI" envDefault:"5000000000"`
	EthGasPriceDefault  *big.Int `env:"ETH_GAS_PRICE_DEFAULT" envDefault:"20000000000"`
	EthGasBumpThreshold uint64   `env:"ETH_GAS_BUMP_THRESHOLD" envDefault:"12"`
	PollingSchedule   string     `env:"POLLING_SCHEDULE" envDefault:"* * * * * *"`
}

func NewConfig() Config {
	config := Config{}
	env.Parse(&config)
	dir, err := homedir.Expand(config.RootDir)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.MkdirAll(dir, os.FileMode(0700)); err != nil {
		log.Fatal(err)
	}
	config.RootDir = dir
	return config
}

func (self Config) KeysDir() string {
	return path.Join(self.RootDir, "keys")
}

func (self Config) EthereumSubscriptionURL() string {
	return self.EthereumURL
}