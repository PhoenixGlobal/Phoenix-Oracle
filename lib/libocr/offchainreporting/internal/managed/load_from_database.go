package managed

import (
	"context"

	"PhoenixOracle/lib/libocr/offchainreporting/loghelper"
	"PhoenixOracle/lib/libocr/offchainreporting/types"
)

func loadConfigFromDatabase(ctx context.Context, database types.Database, logger loghelper.LoggerWithContext) *types.ContractConfig {
	cc, err := database.ReadConfig(ctx)
	if err != nil {
		logger.ErrorIfNotCanceled("loadConfigFromDatabase: Error during Database.ReadConfig", ctx, types.LogFields{
			"error": err,
		})
		return nil
	}

	if cc == nil {
		logger.Info("loadConfigFromDatabase: Database.ReadConfig returned nil, no configuration to restore", nil)
		return nil
	}

	return cc
}
