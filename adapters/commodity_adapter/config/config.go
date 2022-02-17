package config

import (
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config defines a configuration for the web API.
type Config struct {
	APIPort                int    `yaml:"api_port"`
	CommodityServiceTarget string `yaml:"commodity_service_target"`
	CurrencyServiceTarget  string `yaml:"currency_service_target"`
	CryptoServiceTarget    string `yaml:"crypto_service_target"`
}

// GetConfig tries to load and handle the configuration file.
func GetConfig(path string) (*Config, error) {

	// load configuration
	content, err := ioutil.ReadFile(filepath.Join(rootDir(), "..", path))
	if err != nil {
		return nil, errors.Wrap(err, "could not read config file")
	}

	// validate and apply the settings
	var cfg Config
	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config file")
	}

	return &cfg, nil
}

func rootDir() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Dir(b)
}
