package offchainreporting

import (
	"context"
	"math/big"
	"time"

	"PhoenixOracle/core/log"
	"PhoenixOracle/lib/libocr/gethwrappers/offchainaggregator"
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var (
	_ ocrtypes.ContractTransmitter = &OCRContractTransmitter{}
)

type (
	OCRContractTransmitter struct {
		contractAddress gethCommon.Address
		contractABI     abi.ABI
		transmitter     Transmitter
		contractCaller  *offchainaggregator.OffchainAggregatorCaller
		tracker         *OCRContractTracker
		chainID         *big.Int
	}

	Transmitter interface {
		CreateEthTransaction(ctx context.Context, toAddress gethCommon.Address, payload []byte) error
		FromAddress() gethCommon.Address
	}
)

func NewOCRContractTransmitter(
	address gethCommon.Address,
	contractCaller *offchainaggregator.OffchainAggregatorCaller,
	contractABI abi.ABI,
	transmitter Transmitter,
	logBroadcaster log.Broadcaster,
	tracker *OCRContractTracker,
	chainID *big.Int,
) *OCRContractTransmitter {
	return &OCRContractTransmitter{
		contractAddress: address,
		contractABI:     contractABI,
		transmitter:     transmitter,
		contractCaller:  contractCaller,
		tracker:         tracker,
		chainID:         chainID,
	}
}

func (oc *OCRContractTransmitter) Transmit(ctx context.Context, report []byte, rs, ss [][32]byte, vs [32]byte) error {
	payload, err := oc.contractABI.Pack("transmit", report, rs, ss, vs)
	if err != nil {
		return errors.Wrap(err, "abi.Pack failed")
	}

	return errors.Wrap(oc.transmitter.CreateEthTransaction(ctx, oc.contractAddress, payload), "failed to send Eth transaction")
}

func (oc *OCRContractTransmitter) LatestTransmissionDetails(ctx context.Context) (configDigest ocrtypes.ConfigDigest, epoch uint32, round uint8, latestAnswer ocrtypes.Observation, latestTimestamp time.Time, err error) {
	opts := bind.CallOpts{Context: ctx, Pending: false}
	result, err := oc.contractCaller.LatestTransmissionDetails(&opts)
	if err != nil {
		return configDigest, 0, 0, ocrtypes.Observation(nil), time.Time{}, errors.Wrap(err, "error getting LatestTransmissionDetails")
	}
	return result.ConfigDigest, result.Epoch, result.Round, ocrtypes.Observation(result.LatestAnswer), time.Unix(int64(result.LatestTimestamp), 0), nil
}

func (oc *OCRContractTransmitter) FromAddress() gethCommon.Address {
	return oc.transmitter.FromAddress()
}

func (oc *OCRContractTransmitter) ChainID() *big.Int {
	return oc.chainID
}

func (oc *OCRContractTransmitter) LatestRoundRequested(ctx context.Context, lookback time.Duration) (configDigest ocrtypes.ConfigDigest, epoch uint32, round uint8, err error) {
	return oc.tracker.LatestRoundRequested(ctx, lookback)
}

func (oc *OCRContractTransmitter) LatestNewIndexes(ctx context.Context, lookback time.Duration) (newIndexes []int,err error) {
	newIndexes,err=oc.tracker.LatestNewIndexes(ctx, lookback)
	if err!=nil{
		return
	}
	if len(newIndexes)==0{
		newIndexes,err=oc.GetLatestNewIndexes(ctx)
		if err!=nil{
			return
		}
		oc.tracker.SetNewIndexes(newIndexes)
		return
	}
	return
}

func (oc *OCRContractTransmitter) GetLatestNewIndexes(ctx context.Context) (newIndexes []int,err error) {
	opts := bind.CallOpts{Context: ctx, Pending: false}
	result, err := oc.contractCaller.GetIndexes(&opts)
	if err != nil {
		return newIndexes, errors.Wrap(err, "error getting GetIndexes")
	}
	for _,index:=range result{
		index64:=index.Int64()
		newIndexes=append(newIndexes,int(index64))
	}
	return newIndexes, nil
}