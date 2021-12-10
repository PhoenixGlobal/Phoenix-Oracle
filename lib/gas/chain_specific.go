package gas

import "math/big"

func chainSpecificIsUsableTx(tx Transaction, minGasPriceWei, chainID *big.Int) bool {
	if isXDai(chainID) {
		if tx.GasPrice.Cmp(minGasPriceWei) < 0 {
			return false
		}
	}
	return true
}

func isXDai(chainID *big.Int) bool {
	return chainID.Cmp(big.NewInt(100)) == 0
}
