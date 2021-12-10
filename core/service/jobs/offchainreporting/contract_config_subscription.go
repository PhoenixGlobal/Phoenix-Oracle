package offchainreporting

import (
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
)

var _ ocrtypes.ContractConfigSubscription = &OCRContractConfigSubscription{}

type OCRContractConfigSubscription OCRContractTracker

func (sub *OCRContractConfigSubscription) Configs() <-chan ocrtypes.ContractConfig {
	return sub.chConfigs
}

func (sub *OCRContractConfigSubscription) Close() {}
