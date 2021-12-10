package offchainreporting

import (
	"context"
	"math/big"
	"time"

	"PhoenixOracle/db/models"

	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/pipeline"
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"
	"github.com/pkg/errors"
)

type dataSource struct {
	pipelineRunner        pipeline.Runner
	jobSpec               job.Job
	spec                  pipeline.Spec
	ocrLogger             logger.Logger
	runResults            chan<- pipeline.Run
	currentBridgeMetadata models.BridgeMetaData
}

var _ ocrtypes.DataSource = (*dataSource)(nil)

func (ds *dataSource) Observe(ctx context.Context) (ocrtypes.Observation, error) {
	var observation ocrtypes.Observation
	md, err := models.MarshalBridgeMetaData(ds.currentBridgeMetadata.LatestAnswer, ds.currentBridgeMetadata.UpdatedAt)
	if err != nil {
		logger.Warnw("unable to attach metadata for run", "err", err)
	}

	vars := pipeline.NewVarsFrom(map[string]interface{}{
		"jobSpec": map[string]interface{}{
			"databaseID":    ds.jobSpec.ID,
			"externalJobID": ds.jobSpec.ExternalJobID,
			"name":          ds.jobSpec.Name.ValueOrZero(),
		},
		"jobRun": map[string]interface{}{
			"meta": md,
		},
	})

	run, trrs, err := ds.pipelineRunner.ExecuteRun(ctx, ds.spec, vars, ds.ocrLogger)
	if err != nil {
		return observation, errors.Wrapf(err, "error executing run for spec ID %v", ds.spec.ID)
	}
	finalResult := trrs.FinalResult()

	// Do the database write in a non-blocking fashion
	// so we can return the observation results immediately.
	// This is helpful in the case of a blocking API call, where
	// we reach the passed in context deadline and we want to
	// immediately return any result we have and do not want to have
	// a db write block that.
	select {
	case ds.runResults <- run:
	default:
		return nil, errors.Errorf("unable to enqueue run save for job ID %v, buffer full", ds.spec.JobID)
	}

	result, err := finalResult.SingularResult()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting singular result for job ID %v", ds.spec.JobID)
	}

	if result.Error != nil {
		return nil, result.Error
	}

	asDecimal, err := utils.ToDecimal(result.Value)
	if err != nil {
		return nil, errors.Wrap(err, "cannot convert observation to decimal")
	}
	ds.currentBridgeMetadata = models.BridgeMetaData{
		LatestAnswer: asDecimal.BigInt(),
		UpdatedAt:    big.NewInt(time.Now().Unix()),
	}
	return asDecimal.BigInt(), nil
}
