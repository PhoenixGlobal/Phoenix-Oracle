package config

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/core/chain"
	ocr "PhoenixOracle/lib/libocr/offchainreporting"
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/logger"
	ethCore "github.com/ethereum/go-ethereum/core"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gorm.io/gorm"
)

type EVMOnlyConfig interface {
	BalanceMonitorEnabled() bool
	BlockEmissionIdleWarningThreshold() time.Duration
	BlockHistoryEstimatorBatchSize() (size uint32)
	BlockHistoryEstimatorBlockDelay() uint16
	BlockHistoryEstimatorBlockHistorySize() uint16
	BlockHistoryEstimatorTransactionPercentile() uint16
	EthTxReaperInterval() time.Duration
	EthTxReaperThreshold() time.Duration
	EthTxResendAfterThreshold() time.Duration
	EvmDefaultBatchSize() uint32
	EvmFinalityDepth() uint
	EvmGasBumpPercent() uint16
	EvmGasBumpThreshold() uint64
	EvmGasBumpTxDepth() uint16
	EvmGasBumpWei() *big.Int
	EvmGasLimitDefault() uint64
	EvmGasLimitMultiplier() float32
	EvmGasLimitTransfer() uint64
	EvmGasPriceDefault() *big.Int
	EvmHeadTrackerHistoryDepth() uint
	EvmHeadTrackerMaxBufferSize() uint
	EvmHeadTrackerSamplingInterval() time.Duration
	EvmLogBackfillBatchSize() uint32
	EvmMaxGasPriceWei() *big.Int
	EvmMaxInFlightTransactions() uint32
	EvmMaxQueuedTransactions() uint64
	EvmMinGasPriceWei() *big.Int
	EvmNonceAutoSync() bool
	EvmRPCDefaultBatchSize() uint32
	FlagsContractAddress() string
	GasEstimatorMode() string
	PhbContractAddress() string
	MinIncomingConfirmations() uint32
	MinRequiredOutgoingConfirmations() uint64
	MinimumContractPayment() *assets.Phb
	OCRContractConfirmations() uint16
	SetEvmGasPriceDefault(value *big.Int) error
	Validate() error
}

type EVMConfig interface {
	GeneralConfig
	EVMOnlyConfig
}

type evmConfig struct {
	GeneralConfig
	chainSpecificConfig chain.ChainSpecificConfig
}

func NewEVMConfig(cfg GeneralConfig) EVMConfig {
	css := cfg.Chain().Config()
	return &evmConfig{cfg, css}
}

func (c *evmConfig) Validate() error {
	return multierr.Combine(
		c.GeneralConfig.Validate(),
		c.validate(),
	)
}

func (c *evmConfig) validate() (err error) {
	ethGasBumpPercent := c.EvmGasBumpPercent()
	if uint64(ethGasBumpPercent) < ethCore.DefaultTxPoolConfig.PriceBump {
		err = multierr.Combine(err, errors.Errorf(
			"ETH_GAS_BUMP_PERCENT of %v may not be less than Geth's default of %v",
			c.EvmGasBumpPercent(),
			ethCore.DefaultTxPoolConfig.PriceBump,
		))
	}

	if uint32(c.EvmGasBumpTxDepth()) > c.EvmMaxInFlightTransactions() {
		err = multierr.Combine(err, errors.New("ETH_GAS_BUMP_TX_DEPTH must be less than or equal to ETH_MAX_IN_FLIGHT_TRANSACTIONS"))
	}
	if c.EvmMinGasPriceWei().Cmp(c.EvmGasPriceDefault()) > 0 {
		err = multierr.Combine(err, errors.New("ETH_MIN_GAS_PRICE_WEI must be less than or equal to ETH_GAS_PRICE_DEFAULT"))
	}
	if c.EvmMaxGasPriceWei().Cmp(c.EvmGasPriceDefault()) < 0 {
		err = multierr.Combine(err, errors.New("ETH_MAX_GAS_PRICE_WEI must be greater than or equal to ETH_GAS_PRICE_DEFAULT"))
	}
	if c.EvmHeadTrackerHistoryDepth() < c.EvmFinalityDepth() {
		err = multierr.Combine(err, errors.New("ETH_HEAD_TRACKER_HISTORY_DEPTH must be equal to or greater than ETH_FINALITY_DEPTH"))
	}
	if c.GasEstimatorMode() == "BlockHistory" && c.BlockHistoryEstimatorBlockHistorySize() <= 0 {
		err = multierr.Combine(err, errors.New("GAS_UPDATER_BLOCK_HISTORY_SIZE must be greater than or equal to 1 if block history estimator is enabled"))
	}
	if c.EvmFinalityDepth() < 1 {
		err = multierr.Combine(err, errors.New("ETH_FINALITY_DEPTH must be greater than or equal to 1"))
	}
	if c.MinIncomingConfirmations() < 1 {
		err = multierr.Combine(err, errors.New("MIN_INCOMING_CONFIRMATIONS must be greater than or equal to 1"))
	}
	lc := ocrtypes.LocalConfig{
		BlockchainTimeout:                      c.OCRBlockchainTimeout(),
		ContractConfigConfirmations:            c.OCRContractConfirmations(),
		ContractConfigTrackerPollInterval:      c.OCRContractPollInterval(),
		ContractConfigTrackerSubscribeInterval: c.OCRContractSubscribeInterval(),
		ContractTransmitterTransmitTimeout:     c.OCRContractTransmitterTransmitTimeout(),
		DatabaseTimeout:                        c.OCRDatabaseTimeout(),
		DataSourceTimeout:                      c.OCRObservationTimeout(),
		DataSourceGracePeriod:                  c.OCRObservationGracePeriod(),
	}
	if ocrerr := ocr.SanityCheckLocalConfig(lc); ocrerr != nil {
		err = multierr.Combine(err, ocrerr)
	}

	return err
}

func (c *evmConfig) EvmBalanceMonitorBlockDelay() uint16 {
	val, ok := lookupEnv("ETH_BALANCE_MONITOR_BLOCK_DELAY", parseUint16)
	if ok {
		return val.(uint16)
	}
	return c.chainSpecificConfig.BalanceMonitorBlockDelay
}

func (c *evmConfig) EvmGasBumpThreshold() uint64 {
	val, ok := lookupEnv("ETH_GAS_BUMP_THRESHOLD", parseUint64)
	if ok {
		return val.(uint64)
	}
	return c.chainSpecificConfig.GasBumpThreshold
}

func (c *evmConfig) EvmGasBumpWei() *big.Int {
	val, ok := lookupEnv("ETH_GAS_BUMP_WEI", parseBigInt)
	if ok {
		return val.(*big.Int)
	}
	n := c.chainSpecificConfig.GasBumpWei
	return &n
}

func (c *evmConfig) EvmMaxInFlightTransactions() uint32 {
	val, ok := lookupEnv("ETH_MAX_IN_FLIGHT_TRANSACTIONS", parseUint32)
	if ok {
		return val.(uint32)
	}
	return c.chainSpecificConfig.MaxInFlightTransactions
}

func (c *evmConfig) EvmMaxGasPriceWei() *big.Int {
	val, ok := lookupEnv("ETH_MAX_GAS_PRICE_WEI", parseBigInt)
	if ok {
		return val.(*big.Int)
	}
	n := c.chainSpecificConfig.MaxGasPriceWei
	return &n
}

func (c *evmConfig) EvmMaxQueuedTransactions() uint64 {
	val, ok := lookupEnv("ETH_MAX_QUEUED_TRANSACTIONS", parseUint64)
	if ok {
		return val.(uint64)
	}
	return c.chainSpecificConfig.MaxQueuedTransactions
}

func (c *evmConfig) EvmMinGasPriceWei() *big.Int {
	val, ok := lookupEnv("ETH_MIN_GAS_PRICE_WEI", parseBigInt)
	if ok {
		return val.(*big.Int)
	}
	n := c.chainSpecificConfig.MinGasPriceWei
	return &n
}

func (c *evmConfig) EvmGasLimitDefault() uint64 {
	val, ok := lookupEnv("ETH_GAS_LIMIT_DEFAULT", parseUint64)
	if ok {
		return val.(uint64)
	}
	return c.chainSpecificConfig.GasLimitDefault
}

func (c *evmConfig) EvmGasLimitTransfer() uint64 {
	val, ok := lookupEnv("ETH_GAS_LIMIT_TRANSFER", parseUint64)
	if ok {
		return val.(uint64)
	}
	return c.chainSpecificConfig.GasLimitTransfer
}

func (c *evmConfig) EvmGasPriceDefault() *big.Int {
	concreteGCfg, ok := c.GeneralConfig.(*generalConfig)
	if ok && concreteGCfg.ORM != nil {
		var value big.Int
		if err := concreteGCfg.ORM.GetConfigValue("EvmGasPriceDefault", &value); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnw("Error while trying to fetch EvmGasPriceDefault.", "error", err)
		} else if err == nil {
			return &value
		}
	}
	val, ok := lookupEnv("ETH_GAS_PRICE_DEFAULT", parseBigInt)
	if ok {
		return val.(*big.Int)
	}
	n := c.chainSpecificConfig.GasPriceDefault
	return &n
}

func (c *evmConfig) SetEvmGasPriceDefault(value *big.Int) error {
	min := c.EvmMinGasPriceWei()
	max := c.EvmMaxGasPriceWei()
	if value.Cmp(min) < 0 {
		return errors.Errorf("cannot set default gas price to %s, it is below the minimum allowed value of %s", value.String(), min.String())
	}
	if value.Cmp(max) > 0 {
		return errors.Errorf("cannot set default gas price to %s, it is above the maximum allowed value of %s", value.String(), max.String())
	}
	concreteGCfg, ok := c.GeneralConfig.(*generalConfig)
	if !ok {
		return errors.Errorf("cannot get runtime store; %T is not *generalConfig", c.GeneralConfig)
	}
	if concreteGCfg.ORM == nil {
		return errors.New("SetEvmGasPriceDefault: No runtime store installed")
	}
	return concreteGCfg.ORM.SetConfigValue("EvmGasPriceDefault", value)
}

func (c *evmConfig) EvmFinalityDepth() uint {
	val, ok := lookupEnv("ETH_FINALITY_DEPTH", parseUint64)
	if ok {
		return val.(uint)
	}
	return c.chainSpecificConfig.FinalityDepth
}

func (c *evmConfig) EvmHeadTrackerHistoryDepth() uint {
	val, ok := lookupEnv("ETH_HEAD_TRACKER_HISTORY_DEPTH", parseUint64)
	if ok {
		return val.(uint)
	}
	return c.chainSpecificConfig.HeadTrackerHistoryDepth
}

func (c *evmConfig) EvmHeadTrackerSamplingInterval() time.Duration {
	val, ok := lookupEnv("ETH_HEAD_TRACKER_SAMPLING_INTERVAL", parseDuration)
	if ok {
		return val.(time.Duration)
	}
	return c.chainSpecificConfig.HeadTrackerSamplingInterval
}

func (c *evmConfig) BlockEmissionIdleWarningThreshold() time.Duration {
	return c.chainSpecificConfig.BlockEmissionIdleWarningThreshold
}

func (c *evmConfig) EthTxResendAfterThreshold() time.Duration {
	val, ok := lookupEnv("ETH_TX_RESEND_AFTER_THRESHOLD", parseDuration)
	if ok {
		return val.(time.Duration)
	}
	return c.chainSpecificConfig.EthTxResendAfterThreshold
}

func (c *evmConfig) BlockHistoryEstimatorBatchSize() (size uint32) {
	val, ok := lookupEnv("BLOCK_HISTORY_ESTIMATOR_BATCH_SIZE", parseUint32)
	if ok {
		size = val.(uint32)
	} else {
		val, ok = lookupEnv("GAS_UPDATER_BATCH_SIZE", parseUint32)
		if ok {
			logger.Warn("GAS_UPDATER_BATCH_SIZE is deprecated, please use BLOCK_HISTORY_ESTIMATOR_BATCH_SIZE instead")
			size = val.(uint32)
		} else {
			size = c.chainSpecificConfig.BlockHistoryEstimatorBatchSize
		}
	}
	if size > 0 {
		return size
	}
	return c.EvmDefaultBatchSize()
}

func (c *evmConfig) BlockHistoryEstimatorBlockDelay() uint16 {
	val, ok := lookupEnv("BLOCK_HISTORY_ESTIMATOR_BLOCK_DELAY", parseUint16)
	if ok {
		return val.(uint16)
	}
	val, ok = lookupEnv("GAS_UPDATER_BLOCK_DELAY", parseUint16)
	if ok {
		logger.Warn("GAS_UPDATER_BLOCK_DELAY is deprecated, please use BLOCK_HISTORY_ESTIMATOR_BLOCK_DELAY instead")
		return val.(uint16)
	}
	return c.chainSpecificConfig.BlockHistoryEstimatorBlockDelay
}

func (c *evmConfig) BlockHistoryEstimatorBlockHistorySize() uint16 {
	val, ok := lookupEnv("BLOCK_HISTORY_ESTIMATOR_BLOCK_HISTORY_SIZE", parseUint16)
	if ok {
		return val.(uint16)
	}
	val, ok = lookupEnv("GAS_UPDATER_BLOCK_HISTORY_SIZE", parseUint16)
	if ok {
		logger.Warn("GAS_UPDATER_BLOCK_HISTORY_SIZE is deprecated, please use BLOCK_HISTORY_ESTIMATOR_BLOCK_HISTORY_SIZE instead")
		return val.(uint16)
	}
	return c.chainSpecificConfig.BlockHistoryEstimatorBlockHistorySize
}

func (c *evmConfig) BlockHistoryEstimatorTransactionPercentile() uint16 {
	val, ok := lookupEnv("BLOCK_HISTORY_ESTIMATOR_TRANSACTION_PERCENTILE", parseUint16)
	if ok {
		return val.(uint16)
	}
	val, ok = lookupEnv("GAS_UPDATER_TRANSACTION_PERCENTILE", parseUint16)
	if ok {
		logger.Warn("GAS_UPDATER_TRANSACTION_PERCENTILE is deprecated, please use BLOCK_HISTORY_ESTIMATORBLOCK_HISTORY_ESTIMATOR_PERCENTILE instead")
		return val.(uint16)
	}
	return c.chainSpecificConfig.BlockHistoryEstimatorTransactionPercentile
}

func (c *evmConfig) GasEstimatorMode() string {
	if c.EthereumDisabled() {
		return "FixedPrice"
	}
	val, ok := lookupEnv("GAS_ESTIMATOR_MODE", parseString)
	if ok {
		return val.(string)
	}
	enabled, ok := lookupEnv("GAS_UPDATER_ENABLED", parseBool)
	if ok {
		if enabled.(bool) {
			logger.Warn("GAS_UPDATER_ENABLED has been deprecated, to enable the block history estimator, please use GAS_ESTIMATOR_MODE=BlockHistory instead")
			return "BlockHistory"
		}
		logger.Warn("GAS_UPDATER_ENABLED has been deprecated, to disable the block history estimator, please use GAS_ESTIMATOR_MODE=FixedPrice instead")
		return "FixedPrice"
	}
	return c.chainSpecificConfig.GasEstimatorMode
}

// PhbContractAddress represents the address of the official PHB token
// contract on the current Chain
func (c *evmConfig) PhbContractAddress() string {
	val, ok := lookupEnv("PHB_CONTRACT_ADDRESS", parseString)
	if ok {
		return val.(string)
	}
	return c.chainSpecificConfig.PhbContractAddress
}

func (c *evmConfig) OCRContractConfirmations() uint16 {
	val, ok := lookupEnv("OCR_CONTRACT_CONFIRMATIONS", parseUint16)
	if ok {
		return val.(uint16)
	}
	return c.chainSpecificConfig.OCRContractConfirmations
}

func (c *evmConfig) MinIncomingConfirmations() uint32 {
	val, ok := lookupEnv("MIN_INCOMING_CONFIRMATIONS", parseUint32)
	if ok {
		return val.(uint32)
	}
	return c.chainSpecificConfig.MinIncomingConfirmations
}

func (c *evmConfig) MinRequiredOutgoingConfirmations() uint64 {
	val, ok := lookupEnv("MIN_REQUIRED_OUTGOING_CONFIRMATIONS", parseUint64)
	if ok {
		return val.(uint64)
	}
	return c.chainSpecificConfig.MinRequiredOutgoingConfirmations
}

func (c *evmConfig) MinimumContractPayment() *assets.Phb {
	val, ok := lookupEnv("MINIMUM_CONTRACT_PAYMENT_PHB_JUELS", parsePhb)
	if ok {
		return val.(*assets.Phb)
	}
	val, ok = lookupEnv("MINIMUM_CONTRACT_PAYMENT", parseString)
	if ok {
		logger.Warn("MINIMUM_CONTRACT_PAYMENT is deprecated, please use MINIMUM_CONTRACT_PAYMENT_PHB_JUELS instead")
		str := val.(string)
		value, ok := new(assets.Phb).SetString(str, 10)
		if ok {
			return value
		}
		logger.Errorw(
			"Invalid value provided for MINIMUM_CONTRACT_PAYMENT, falling back to default.",
			"value", str)
	}
	return c.chainSpecificConfig.MinimumContractPayment
}

func (c *evmConfig) EvmGasBumpTxDepth() uint16 {
	val, ok := lookupEnv("ETH_GAS_BUMP_TX_DEPTH", parseUint16)
	if ok {
		return val.(uint16)
	}
	return c.chainSpecificConfig.GasBumpTxDepth
}

func (c *evmConfig) EvmDefaultBatchSize() uint32 {
	val, ok := lookupEnv("ETH_RPC_DEFAULT_BATCH_SIZE", parseUint32)
	if ok {
		return val.(uint32)
	}
	return c.chainSpecificConfig.RPCDefaultBatchSize
}

func (c *evmConfig) EvmGasBumpPercent() uint16 {
	val, ok := lookupEnv("ETH_GAS_BUMP_PERCENT", parseUint16)
	if ok {
		return val.(uint16)
	}
	return c.chainSpecificConfig.GasBumpPercent
}

func (c *evmConfig) EvmNonceAutoSync() bool {
	val, ok := lookupEnv("ETH_NONCE_AUTO_SYNC", parseBool)
	if ok {
		return val.(bool)
	}
	return c.chainSpecificConfig.NonceAutoSync
}

func (c *evmConfig) EvmGasLimitMultiplier() float32 {
	val, ok := lookupEnv("ETH_GAS_LIMIT_MULTIPLIER", parseF32)
	if ok {
		return val.(float32)
	}
	return c.chainSpecificConfig.GasLimitMultiplier
}

func (c *evmConfig) EvmHeadTrackerMaxBufferSize() uint {
	val, ok := lookupEnv("ETH_HEAD_TRACKER_MAX_BUFFER_SIZE", parseUint64)
	if ok {
		return uint(val.(uint64))
	}
	return c.chainSpecificConfig.HeadTrackerMaxBufferSize
}

func (c *evmConfig) EthTxReaperInterval() time.Duration {
	val, ok := lookupEnv("ETH_TX_REAPER_INTERVAL", parseDuration)
	if ok {
		return val.(time.Duration)
	}
	return c.chainSpecificConfig.EthTxReaperInterval
}

func (c *evmConfig) EthTxReaperThreshold() time.Duration {
	val, ok := lookupEnv("ETH_TX_REAPER_THRESHOLD", parseDuration)
	if ok {
		return val.(time.Duration)
	}
	return c.chainSpecificConfig.EthTxReaperThreshold
}

func (c *evmConfig) EvmLogBackfillBatchSize() uint32 {
	val, ok := lookupEnv("ETH_LOG_BACKFILL_BATCH_SIZE", parseUint32)
	if ok {
		return val.(uint32)
	}
	return c.chainSpecificConfig.LogBackfillBatchSize
}

func (c *evmConfig) EvmRPCDefaultBatchSize() uint32 {
	val, ok := lookupEnv("ETH_RPC_DEFAULT_BATCH_SIZE", parseUint32)
	if ok {
		return val.(uint32)
	}
	return c.chainSpecificConfig.RPCDefaultBatchSize
}

func (c *evmConfig) FlagsContractAddress() string {
	val, ok := lookupEnv("FLAGS_CONTRACT_ADDRESS", parseString)
	if ok {
		return val.(string)
	}
	return c.chainSpecificConfig.FlagsContractAddress
}

func (c *evmConfig) BalanceMonitorEnabled() bool {
	if c.EthereumDisabled() {
		return false
	}
	val, ok := lookupEnv("BALANCE_MONITOR_ENABLED", parseBool)
	if ok {
		return val.(bool)
	}
	return c.chainSpecificConfig.BalanceMonitorEnabled
}

func lookupEnv(k string, parse func(string) (interface{}, error)) (interface{}, bool) {
	s, ok := os.LookupEnv(k)
	if ok {
		val, err := parse(s)
		if err != nil {
			logger.Errorw(
				fmt.Sprintf("Invalid value provided for %s, falling back to default.", s),
				"value", s,
				"key", k,
				"error", err)
			return nil, false
		}
		return val, true
	}
	return nil, false
}
