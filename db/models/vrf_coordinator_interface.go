package models

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/internal/gethwrappers/generated/solidity_vrf_coordinator_interface"
	"PhoenixOracle/util"
)

type RawRandomnessRequestLog solidity_vrf_coordinator_interface.VRFCoordinatorRandomnessRequest

type RandomnessRequestLog struct {
	KeyHash   common.Hash
	Seed      *big.Int // uint256
	JobID     common.Hash
	Sender    common.Address
	Fee       *assets.Phb // uint256
	RequestID common.Hash
	Raw       RawRandomnessRequestLog
}

var dummyCoordinator, _ = solidity_vrf_coordinator_interface.NewVRFCoordinator(
	common.Address{}, nil)

func toGethLog(log types.Log) types.Log {
	return types.Log{
		Address:     log.Address,
		Topics:      log.Topics,
		Data:        []byte(log.Data),
		BlockNumber: log.BlockNumber,
		TxHash:      log.TxHash,
		TxIndex:     log.TxIndex,
		BlockHash:   log.BlockHash,
		Index:       log.Index,
		Removed:     log.Removed,
	}
}

func ParseRandomnessRequestLog(log types.Log) (*RandomnessRequestLog, error) {
	rawLog, err := dummyCoordinator.ParseRandomnessRequest(toGethLog(log))
	if err != nil {
		return nil, errors.Wrapf(err,
			"while parsing %x as RandomnessRequestLog", log.Data)
	}
	return RawRandomnessRequestLogToRandomnessRequestLog(
		(*RawRandomnessRequestLog)(rawLog)), nil
}

func (l *RandomnessRequestLog) RawData() ([]byte, error) {
	return randomnessRequestRawDataArgs().Pack(l.KeyHash,
		l.Seed, l.Sender, (*big.Int)(l.Fee), l.RequestID)
}

func (l *RandomnessRequestLog) Equal(ol RandomnessRequestLog) bool {
	return l.KeyHash == ol.KeyHash &&
		equal(l.Seed, ol.Seed) &&
		l.JobID == ol.JobID &&
		l.Sender == ol.Sender &&
		l.Fee.Cmp(ol.Fee) == 0 &&
		l.RequestID == ol.RequestID
}

func (l *RandomnessRequestLog) ComputedRequestID() common.Hash {
	soliditySeed, err := utils.Uint256ToBytes(l.Seed)
	if err != nil {
		panic(errors.Wrapf(err, "vrf seed out of bounds in %#+v", l))
	}
	return utils.MustHash(string(append(l.KeyHash[:], soliditySeed...)))
}

func RawRandomnessRequestLogToRandomnessRequestLog(
	l *RawRandomnessRequestLog) *RandomnessRequestLog {
	return &RandomnessRequestLog{
		KeyHash:   l.KeyHash,
		Seed:      l.Seed,
		JobID:     l.JobID,
		Sender:    l.Sender,
		Fee:       (*assets.Phb)(l.Fee),
		RequestID: l.RequestID,
		Raw:       *l,
	}
}

func equal(left, right *big.Int) bool { return left.Cmp(right) == 0 }
