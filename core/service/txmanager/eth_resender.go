package txmanager

import (
	"fmt"
	"time"

	"PhoenixOracle/build/static"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/null"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const defaultResenderPollInterval = 5 * time.Second

type EthResender struct {
	db        *gorm.DB
	ethClient ethereum.Client
	interval  time.Duration
	config    Config

	chStop chan struct{}
	chDone chan struct{}
}

func NewEthResender(db *gorm.DB, ethClient ethereum.Client, pollInterval time.Duration, config Config) *EthResender {
	if config.EthTxResendAfterThreshold() == 0 {
		panic("EthResender requires a non-zero threshold")
	}
	return &EthResender{
		db,
		ethClient,
		pollInterval,
		config,
		make(chan struct{}),
		make(chan struct{}),
	}
}

func (er *EthResender) Start() {
	logger.Infof("EthResender: Enabled with poll interval of %s and age threshold of %s", er.interval, er.config.EthTxResendAfterThreshold())
	go er.runLoop()
}

func (er *EthResender) Stop() {
	close(er.chStop)
	<-er.chDone
}

func (er *EthResender) runLoop() {
	defer close(er.chDone)

	if err := er.resendUnconfirmed(); err != nil {
		logger.Warnw("EthResender: failed to resend unconfirmed transactions", "err", err)
	}

	ticker := time.NewTicker(utils.WithJitter(er.interval))
	defer ticker.Stop()
	for {
		select {
		case <-er.chStop:
			return
		case <-ticker.C:
			if err := er.resendUnconfirmed(); err != nil {
				logger.Warnw("EthResender: failed to resend unconfirmed transactions", "err", err)
			}
		}
	}
}

func (er *EthResender) resendUnconfirmed() error {
	ageThreshold := er.config.EthTxResendAfterThreshold()
	maxInFlightTransactions := er.config.EvmMaxInFlightTransactions()

	olderThan := time.Now().Add(-ageThreshold)
	attempts, err := FindEthTxesRequiringResend(er.db, olderThan, maxInFlightTransactions)
	if err != nil {
		return errors.Wrap(err, "failed to findEthTxAttemptsRequiringReceiptFetch")
	}

	if len(attempts) == 0 {
		return nil
	}

	logger.Infow(fmt.Sprintf("EthResender: re-sending %d unconfirmed transactions that were last sent over %s ago. These transactions are taking longer than usual to be mined. %s", len(attempts), ageThreshold, static.EthNodeConnectivityProblemLabel), "n", len(attempts))

	reqs := make([]rpc.BatchElem, len(attempts))
	ethTxIDs := make([]int64, len(attempts))
	for i, attempt := range attempts {
		ethTxIDs[i] = attempt.EthTxID
		req := rpc.BatchElem{
			Method: "eth_sendRawTransaction",
			Args:   []interface{}{hexutil.Encode(attempt.SignedRawTx)},
			Result: &common.Hash{},
		}
		reqs[i] = req
	}

	now := time.Now()
	batchSize := int(er.config.EvmRPCDefaultBatchSize())
	if batchSize == 0 {
		batchSize = len(reqs)
	}
	for i := 0; i < len(reqs); i += batchSize {
		j := i + batchSize
		if j > len(reqs) {
			j = len(reqs)
		}

		logger.Debugw(fmt.Sprintf("EthResender: batch resending transactions %v thru %v", i, j))

		ctx, cancel := ethereum.DefaultQueryCtx()
		if err := er.ethClient.RoundRobinBatchCallContext(ctx, reqs[i:j]); err != nil {
			return errors.Wrap(err, "failed to re-send transactions")
		}
		cancel()

		if err := er.updateBroadcastAts(now, ethTxIDs[i:j]); err != nil {
			return errors.Wrap(err, "failed to update last succeeded on attempts")
		}
	}

	logResendResult(reqs)

	return nil
}

func FindEthTxesRequiringResend(db *gorm.DB, olderThan time.Time, maxInFlightTransactions uint32) (attempts []EthTxAttempt, err error) {
	var limit null.Uint32
	if maxInFlightTransactions > 0 {
		limit = null.Uint32From(maxInFlightTransactions)
	}
	err = db.Raw(`
SELECT DISTINCT ON (eth_tx_id) eth_tx_attempts.*
FROM eth_tx_attempts
JOIN eth_txes ON eth_txes.id = eth_tx_attempts.eth_tx_id AND eth_txes.state IN ('unconfirmed', 'confirmed_missing_receipt')
WHERE eth_tx_attempts.state <> 'in_progress' AND eth_txes.broadcast_at <= ?
ORDER BY eth_tx_attempts.eth_tx_id ASC, eth_txes.nonce ASC, eth_tx_attempts.gas_price DESC
LIMIT ?
`, olderThan, limit).
		Find(&attempts).Error

	return
}

func (er *EthResender) updateBroadcastAts(now time.Time, etxIDs []int64) error {
	return er.db.Exec(`UPDATE eth_txes SET broadcast_at = ? WHERE id = ANY(?) AND broadcast_at < ?`, now, pq.Array(etxIDs), now).Error
}

func logResendResult(reqs []rpc.BatchElem) {
	var nNew int
	var nFatal int
	for _, req := range reqs {
		serr := ethereum.NewSendError(req.Error)
		if serr == nil {
			nNew++
		} else if serr.Fatal() {
			nFatal++
		}
	}
	logger.Debugw("EthResender: completed", "n", len(reqs), "nNew", nNew, "nFatal", nFatal)
}
