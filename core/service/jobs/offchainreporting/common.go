package offchainreporting

import (
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/lib/libocr/offchainreporting/types"
)

func NewLocalConfig(cfg ValidationConfig, spec job.OffchainReportingOracleSpec) types.LocalConfig {
	spec = *job.LoadDynamicConfigVars(cfg, spec)
	lc := types.LocalConfig{
		BlockchainTimeout:                      spec.BlockchainTimeout.Duration(),
		ContractConfigConfirmations:            spec.ContractConfigConfirmations,
		SkipContractConfigConfirmations:        cfg.Chain().IsL2(),
		ContractConfigTrackerPollInterval:      spec.ContractConfigTrackerPollInterval.Duration(),
		ContractConfigTrackerSubscribeInterval: spec.ContractConfigTrackerSubscribeInterval.Duration(),
		ContractTransmitterTransmitTimeout:     cfg.OCRContractTransmitterTransmitTimeout(),
		DatabaseTimeout:                        cfg.OCRDatabaseTimeout(),
		DataSourceTimeout:                      spec.ObservationTimeout.Duration(),
		DataSourceGracePeriod:                  cfg.OCRObservationGracePeriod(),
	}
	if cfg.Dev() {
		lc.DevelopmentMode = types.EnableDangerousDevelopmentMode
	}
	return lc
}
