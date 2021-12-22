package fluxmonitor

import (
	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/keystore/keys/ethkey"
	"github.com/ethereum/go-ethereum/common"
)

type KeyStoreInterface interface {
	SendingKeys() ([]ethkey.KeyV2, error)
	GetRoundRobinAddress(...common.Address) (common.Address, error)
}

type KeyStore struct {
	keystore.Eth
}

func NewKeyStore(ks keystore.Eth) *KeyStore {
	return &KeyStore{ks}
}
