package fluxmonitor

import (
	"PhoenixOracle/core/service/txmanager"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type transmitter interface {
	CreateEthTransaction(db *gorm.DB, newTx txmanager.NewTx) (etx txmanager.EthTx, err error)
}

type ORM interface {
	MostRecentFluxMonitorRoundID(aggregator common.Address) (uint32, error)
	DeleteFluxMonitorRoundsBackThrough(aggregator common.Address, roundID uint32) error
	FindOrCreateFluxMonitorRoundStats(aggregator common.Address, roundID uint32, newRoundLogs uint) (FluxMonitorRoundStatsV2, error)
	UpdateFluxMonitorRoundStats(db *gorm.DB, aggregator common.Address, roundID uint32, runID int64, newRoundLogsAddition uint) error
	CreateEthTransaction(db *gorm.DB, fromAddress, toAddress common.Address, payload []byte, gasLimit uint64) error
}

type orm struct {
	db       *gorm.DB
	txm      transmitter
	strategy txmanager.TxStrategy
}

func NewORM(db *gorm.DB, txm transmitter, strategy txmanager.TxStrategy) *orm {
	return &orm{db, txm, strategy}
}

func (o *orm) MostRecentFluxMonitorRoundID(aggregator common.Address) (uint32, error) {
	var stats FluxMonitorRoundStatsV2
	err := o.db.
		Order("round_id DESC").
		First(&stats, "aggregator = ?", aggregator).
		Error
	if err != nil {
		return 0, err
	}

	return stats.RoundID, nil
}

func (o *orm) DeleteFluxMonitorRoundsBackThrough(aggregator common.Address, roundID uint32) error {
	return o.db.Exec(`
        DELETE FROM flux_monitor_round_stats_v2
        WHERE aggregator = ?
          AND round_id >= ?
    `, aggregator, roundID).Error
}

func (o *orm) FindOrCreateFluxMonitorRoundStats(aggregator common.Address, roundID uint32, newRoundLogs uint) (FluxMonitorRoundStatsV2, error) {

	var stats FluxMonitorRoundStatsV2
	stats.Aggregator = aggregator
	stats.RoundID = roundID
	stats.NumNewRoundLogs = uint64(newRoundLogs)

	err := o.db.FirstOrCreate(&stats,
		FluxMonitorRoundStatsV2{Aggregator: aggregator, RoundID: roundID},
	).Error

	return stats, err
}

func (o *orm) UpdateFluxMonitorRoundStats(db *gorm.DB, aggregator common.Address, roundID uint32, runID int64, newRoundLogsAddition uint) error {
	err := db.Exec(`
        INSERT INTO flux_monitor_round_stats_v2 (
            aggregator, round_id, pipeline_run_id, num_new_round_logs, num_submissions
        ) VALUES (
            ?, ?, ?, ?, 1
        ) ON CONFLICT (aggregator, round_id)
        DO UPDATE SET
          num_new_round_logs = flux_monitor_round_stats_v2.num_new_round_logs + ?,
					num_submissions    = flux_monitor_round_stats_v2.num_submissions + 1,
					pipeline_run_id    = EXCLUDED.pipeline_run_id
    `, aggregator, roundID, runID, newRoundLogsAddition, newRoundLogsAddition).Error
	return errors.Wrapf(err, "Failed to insert round stats for roundID=%v, runID=%v, newRoundLogsAddition=%v", roundID, runID, newRoundLogsAddition)
}

func (o *orm) CountFluxMonitorRoundStats() (int, error) {
	var count int64
	err := o.db.Table("flux_monitor_round_stats_v2").Count(&count).Error

	return int(count), err
}

func (o *orm) CreateEthTransaction(
	db *gorm.DB,
	fromAddress common.Address,
	toAddress common.Address,
	payload []byte,
	gasLimit uint64,
) (err error) {
	_, err = o.txm.CreateEthTransaction(db, txmanager.NewTx{
		FromAddress:    fromAddress,
		ToAddress:      toAddress,
		EncodedPayload: payload,
		GasLimit:       gasLimit,
		Meta:           nil,
		Strategy:       o.strategy,
	})
	return errors.Wrap(err, "Skipped Flux Monitor submission")
}
