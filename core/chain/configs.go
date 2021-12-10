package chain

import (
	"math/big"
	"time"

	"PhoenixOracle/core/assets"
)

type (

	ChainSpecificConfig struct {
		BalanceMonitorEnabled                      bool
		BalanceMonitorBlockDelay                   uint16
		BlockEmissionIdleWarningThreshold          time.Duration
		BlockHistoryEstimatorBatchSize             uint32
		BlockHistoryEstimatorBlockDelay            uint16
		BlockHistoryEstimatorBlockHistorySize      uint16
		BlockHistoryEstimatorTransactionPercentile uint16
		EthTxReaperInterval                        time.Duration
		EthTxReaperThreshold                       time.Duration
		EthTxResendAfterThreshold                  time.Duration
		FinalityDepth                              uint
		FlagsContractAddress                       string
		GasBumpPercent                             uint16
		GasBumpThreshold                           uint64
		GasBumpTxDepth                             uint16
		GasBumpWei                                 big.Int
		GasEstimatorMode                           string
		GasLimitDefault                            uint64
		GasLimitMultiplier                         float32
		GasLimitTransfer                           uint64
		GasPriceDefault                            big.Int
		HeadTrackerHistoryDepth                    uint
		HeadTrackerMaxBufferSize                   uint
		HeadTrackerSamplingInterval                time.Duration
		Layer2Type                                 string
		PhbContractAddress                        string
		LogBackfillBatchSize                       uint32
		MaxGasPriceWei                             big.Int
		MaxInFlightTransactions                    uint32
		MaxQueuedTransactions                      uint64
		MinGasPriceWei                             big.Int
		MinIncomingConfirmations                   uint32
		MinRequiredOutgoingConfirmations           uint64
		MinimumContractPayment                     *assets.Phb
		NonceAutoSync                              bool
		OCRContractConfirmations                   uint16
		RPCDefaultBatchSize                        uint32
		set                                        bool
	}
)

// FallbackConfig represents the "base layer" of config defaults
// It can be overridden on a per-chain basis and may be used if the chain is unknown
var FallbackConfig ChainSpecificConfig

func setConfigs() {

	FallbackConfig = ChainSpecificConfig{
		BalanceMonitorEnabled:                      true,
		BalanceMonitorBlockDelay:                   1,
		BlockEmissionIdleWarningThreshold:          1 * time.Minute,
		BlockHistoryEstimatorBatchSize:             4,
		BlockHistoryEstimatorBlockDelay:            1,
		BlockHistoryEstimatorBlockHistorySize:      24,
		BlockHistoryEstimatorTransactionPercentile: 60,
		EthTxReaperInterval:                        1 * time.Hour,
		EthTxReaperThreshold:                       168 * time.Hour,
		EthTxResendAfterThreshold:                  1 * time.Minute,
		FinalityDepth:                              50,
		GasBumpPercent:                             20,
		GasBumpThreshold:                           3,
		GasBumpTxDepth:                             10,
		GasBumpWei:                                 *assets.GWei(5),
		GasEstimatorMode:                           "BlockHistory",
		GasLimitDefault:                            500000,
		GasLimitMultiplier:                         1.0,
		GasLimitTransfer:                           21000,
		GasPriceDefault:                            *assets.GWei(20),
		HeadTrackerHistoryDepth:                    100,
		HeadTrackerMaxBufferSize:                   3,
		HeadTrackerSamplingInterval:                1 * time.Second,
		PhbContractAddress:                        "",
		LogBackfillBatchSize:                       100,
		MaxGasPriceWei:                             *assets.GWei(5000),
		MaxInFlightTransactions:                    16,
		MaxQueuedTransactions:                      250,
		MinGasPriceWei:                             *assets.GWei(1),
		MinIncomingConfirmations:                   3,
		MinRequiredOutgoingConfirmations:           12,
		MinimumContractPayment:                     assets.NewPhb(100000000000000), // 0.0001 PHB
		NonceAutoSync:                              true,
		OCRContractConfirmations:                   4,
		RPCDefaultBatchSize:                        100,
		set:                                        true,
	}

	mainnet := FallbackConfig
	mainnet.PhbContractAddress = "0x0409633A72D846fc5BBe2f98D88564D35987904D"
	mainnet.MinimumContractPayment = assets.NewPhb(1000000000000000000) // 1 PHB

	kovan := mainnet
	kovan.PhbContractAddress = ""
	goerli := mainnet
	goerli.PhbContractAddress = ""
	rinkeby := mainnet
	rinkeby.PhbContractAddress = "0x66ef1A6a318e4Ce655e7183aceF44268C3995F16"

	// BSC uses Clique consensus with ~3s block times
	// Clique offers finality within (N/2)+1 blocks where N is number of signers
	// There are 21 BSC validators so theoretically finality should occur after 21/2+1 = 11 blocks
	bscMainnet := FallbackConfig
	bscMainnet.BalanceMonitorBlockDelay = 2
	bscMainnet.FinalityDepth = 50   // Keeping this >> 11 because it's not expensive and gives us a safety margin
	bscMainnet.GasBumpThreshold = 5 // 15s delay since feeds update every minute in volatile situations
	bscMainnet.GasBumpWei = *assets.GWei(5)
	bscMainnet.GasPriceDefault = *assets.GWei(5)
	bscMainnet.HeadTrackerHistoryDepth = 100
	bscMainnet.HeadTrackerSamplingInterval = 1 * time.Second
	bscMainnet.BlockEmissionIdleWarningThreshold = 15 * time.Second
	bscMainnet.MinGasPriceWei = *assets.GWei(1)
	bscMainnet.EthTxResendAfterThreshold = 1 * time.Minute
	bscMainnet.BlockHistoryEstimatorBlockDelay = 2
	bscMainnet.BlockHistoryEstimatorBlockHistorySize = 24
	bscMainnet.PhbContractAddress = "0x0409633A72D846fc5BBe2f98D88564D35987904D"
	bscMainnet.MinIncomingConfirmations = 3
	bscMainnet.MinRequiredOutgoingConfirmations = 12

	bscTestnet := bscMainnet

	hecoMainnet := bscMainnet

	EthMainnet.config = mainnet
	EthRinkeby.config = rinkeby
	EthGoerli.config = goerli
	EthKovan.config = kovan
	BSCMainnet.config = bscMainnet
	BSCTestnet.config=bscTestnet
	HecoMainnet.config = hecoMainnet
}
