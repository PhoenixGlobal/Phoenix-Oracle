package log

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type (
	Broadcast interface {
		DecodedLog() interface{}
		RawLog() types.Log
		String() string
		LatestBlockNumber() uint64
		LatestBlockHash() common.Hash
		JobID() int32
	}

	broadcast struct {
		latestBlockNumber uint64
		latestBlockHash   common.Hash
		decodedLog        interface{}
		rawLog            types.Log
		jobID             int32
	}
)

func (b *broadcast) DecodedLog() interface{} {
	return b.decodedLog
}

func (b *broadcast) LatestBlockNumber() uint64 {
	return b.latestBlockNumber
}

func (b *broadcast) LatestBlockHash() common.Hash {
	return b.latestBlockHash
}

func (b *broadcast) RawLog() types.Log {
	return b.rawLog
}

func (b *broadcast) SetDecodedLog(newLog interface{}) {
	b.decodedLog = newLog
}

func (b *broadcast) JobID() int32 {
	return b.jobID
}

func (b *broadcast) String() string {
	return fmt.Sprintf("Broadcast(JobID:%v,LogAddress:%v,Topics(%d):%v)", b.jobID, b.rawLog.Address, len(b.rawLog.Topics), b.rawLog.Topics)
}

func NewLogBroadcast(rawLog types.Log, decodedLog interface{}) Broadcast {
	return &broadcast{
		latestBlockNumber: 0,
		latestBlockHash:   common.Hash{},
		decodedLog:        decodedLog,
		rawLog:            rawLog,
		jobID:             0,
	}
}
