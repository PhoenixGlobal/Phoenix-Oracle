package gas

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"math/big"

	"PhoenixOracle/build/static"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shopspring/decimal"
)

var (
	promNumGasBumps = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tx_manager_num_gas_bumps",
		Help: "Number of gas bumps",
	})

	promGasBumpExceedsLimit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tx_manager_gas_bump_exceeds_limit",
		Help: "Number of times gas bumping failed from exceeding the configured limit. Any counts of this type indicate a serious problem.",
	})
)

func NewEstimator(ethClient ethereum.Client, config Config) Estimator {
	s := config.GasEstimatorMode()
	switch s {
	case "BlockHistory":
		return NewBlockHistoryEstimator(ethClient, config)
	case "FixedPrice":
		return NewFixedPriceEstimator(config)
	case "Optimism":
		return NewOptimismEstimator(config, ethClient)
	case "Optimism2":
		return NewOptimism2Estimator(config, ethClient)
	default:
		logger.Warnf("GasEstimator: unrecognised mode '%s', falling back to FixedPriceEstimator", s)
		return NewFixedPriceEstimator(config)
	}
}

type Estimator interface {
	OnNewLongestChain(context.Context, models.Head)
	Start() error
	Close() error
	EstimateGas(calldata []byte, gasLimit uint64, opts ...Opt) (gasPrice *big.Int, chainSpecificGasLimit uint64, err error)
	BumpGas(originalGasPrice *big.Int, gasLimit uint64) (bumpedGasPrice *big.Int, chainSpecificGasLimit uint64, err error)
}

type Opt int

const (
	OptForceRefetch Opt = iota
)

func applyMultiplier(gasLimit uint64, multiplier float32) uint64 {
	return uint64(decimal.NewFromBigInt(big.NewInt(0).SetUint64(gasLimit), 0).Mul(decimal.NewFromFloat32(multiplier)).IntPart())
}

type Config interface {
	BlockHistoryEstimatorBatchSize() uint32
	BlockHistoryEstimatorBlockDelay() uint16
	BlockHistoryEstimatorBlockHistorySize() uint16
	BlockHistoryEstimatorTransactionPercentile() uint16
	ChainID() *big.Int
	EvmFinalityDepth() uint
	EvmGasBumpPercent() uint16
	EvmGasBumpWei() *big.Int
	EvmGasLimitMultiplier() float32
	EvmGasPriceDefault() *big.Int
	EvmMaxGasPriceWei() *big.Int
	EvmMinGasPriceWei() *big.Int
	GasEstimatorMode() string
}

func Int64ToHex(n int64) string {
	return hexutil.EncodeBig(big.NewInt(n))
}

func HexToInt64(input interface{}) int64 {
	switch v := input.(type) {
	case string:
		big, err := hexutil.DecodeBig(v)
		if err != nil {
			return 0
		}
		return big.Int64()
	case []byte:
		big, err := hexutil.DecodeBig(string(v))
		if err != nil {
			return 0
		}
		return big.Int64()
	default:
		return 0
	}
}

type Block struct {
	Number       int64
	Hash         common.Hash
	ParentHash   common.Hash
	Transactions []Transaction
}

type blockInternal struct {
	Number       string
	Hash         common.Hash
	ParentHash   common.Hash
	Transactions []Transaction
}

func (b Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(blockInternal{
		Int64ToHex(b.Number),
		b.Hash,
		b.ParentHash,
		b.Transactions,
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {
	bi := blockInternal{}
	if err := json.Unmarshal(data, &bi); err != nil {
		return errors.Wrapf(err, "failed to unmarshal to blockInternal, got: '%s'", data)
	}
	n, err := hexutil.DecodeBig(bi.Number)
	if err != nil {
		return errors.Wrapf(err, "failed to decode block number while unmarshalling block, got: '%s'", data)
	}
	*b = Block{
		n.Int64(),
		bi.Hash,
		bi.ParentHash,
		bi.Transactions,
	}
	return nil
}

type TxType uint8

func (txt *TxType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte(`"0x00"`)) {
		data = []byte(`"0x0"`)
	}
	var hx hexutil.Uint64
	if err := (&hx).UnmarshalJSON(data); err != nil {
		return err
	}
	if hx > math.MaxUint8 {
		return errors.Errorf("expected 'type' to fit into a single byte, got: '%s'", data)
	}
	*txt = TxType(hx)
	return nil
}

type transactionInternal struct {
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Type                 *TxType         `json:"type"`
	Hash                 common.Hash     `json:"hash"`
}

type Transaction struct {
	GasPrice             *big.Int
	GasLimit             uint64
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	Type                 TxType
	Hash                 common.Hash
}

const LegacyTxType = TxType(0x0)

func (t *Transaction) UnmarshalJSON(data []byte) error {
	ti := transactionInternal{}
	if err := json.Unmarshal(data, &ti); err != nil {
		return errors.Wrapf(err, "failed to unmarshal to transactionInternal, got: '%s'", data)
	}
	if ti.Gas == nil {
		return errors.Errorf("expected 'gas' to not be null, got: '%s'", data)
	}
	if ti.Type == nil {
		tpe := LegacyTxType
		ti.Type = &tpe
	}
	*t = Transaction{
		(*big.Int)(ti.GasPrice),
		uint64(*ti.Gas),
		(*big.Int)(ti.MaxFeePerGas),
		(*big.Int)(ti.MaxPriorityFeePerGas),
		*ti.Type,
		ti.Hash,
	}
	return nil
}

func BumpGasPriceOnly(config Config, currentGasPrice, originalGasPrice *big.Int, originalGasLimit uint64) (gasPrice *big.Int, chainSpecificGasLimit uint64, err error) {
	gasPrice, err = bumpGasPrice(config, currentGasPrice, originalGasPrice)
	if err != nil {
		return nil, 0, err
	}
	chainSpecificGasLimit = applyMultiplier(originalGasLimit, config.EvmGasLimitMultiplier())
	return
}

func bumpGasPrice(config Config, currentGasPrice, originalGasPrice *big.Int) (*big.Int, error) {
	maxGasPrice := config.EvmMaxGasPriceWei()

	var priceByPercentage = new(big.Int)
	priceByPercentage.Mul(originalGasPrice, big.NewInt(int64(100+config.EvmGasBumpPercent())))
	priceByPercentage.Div(priceByPercentage, big.NewInt(100))

	var priceByIncrement = new(big.Int)
	priceByIncrement.Add(originalGasPrice, config.EvmGasBumpWei())

	bumpedGasPrice := max(priceByPercentage, priceByIncrement)
	if currentGasPrice != nil {
		if currentGasPrice.Cmp(maxGasPrice) > 0 {
			logger.Errorf("invariant violation: ignoring current gas price of %s that would exceed max gas price of %s", currentGasPrice.String(), maxGasPrice.String())
		} else if bumpedGasPrice.Cmp(currentGasPrice) < 0 {
			bumpedGasPrice = currentGasPrice
		}
	}
	if bumpedGasPrice.Cmp(maxGasPrice) > 0 {
		promGasBumpExceedsLimit.Inc()
		return config.EvmMaxGasPriceWei(), errors.Errorf("bumped gas price of %s would exceed configured max gas price of %s (original price was %s). %s",
			bumpedGasPrice.String(), config.EvmMaxGasPriceWei(), originalGasPrice.String(), static.EthNodeConnectivityProblemLabel)
	} else if bumpedGasPrice.Cmp(originalGasPrice) == 0 {
		return bumpedGasPrice, errors.Errorf("bumped gas price of %s is equal to original gas price of %s."+
			" ACTION REQUIRED: This is a configuration error, you must increase either "+
			"ETH_GAS_BUMP_PERCENT or ETH_GAS_BUMP_WEI", bumpedGasPrice.String(), originalGasPrice.String())
	}
	promNumGasBumps.Inc()
	return bumpedGasPrice, nil
}

func max(a, b *big.Int) *big.Int {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}
