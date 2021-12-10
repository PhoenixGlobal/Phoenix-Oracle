package utils

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

const (
	FastN = 2
	FastP = 1
)

type (
	ScryptParams struct{ N, P int }
	ScryptConfigReader interface {
		InsecureFastScrypt() bool
	}
)

var DefaultScryptParams = ScryptParams{N: keystore.StandardScryptN, P: keystore.StandardScryptP}

var FastScryptParams = ScryptParams{N: FastN, P: FastP}

func GetScryptParams(config ScryptConfigReader) ScryptParams {
	if config.InsecureFastScrypt() {
		return FastScryptParams
	}
	return DefaultScryptParams
}
