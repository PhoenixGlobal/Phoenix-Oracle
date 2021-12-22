package fluxmonitor

import (
	"PhoenixOracle/lib/null"
	"github.com/ethereum/go-ethereum/common"
)

type FluxMonitorRoundStatsV2 struct {
	ID              uint64         `gorm:"primary key;not null;auto_increment"`
	PipelineRunID   null.Int64     `gorm:"default:null"`
	Aggregator      common.Address `gorm:"not null"`
	RoundID         uint32         `gorm:"not null"`
	NumNewRoundLogs uint64         `gorm:"not null;default 0"`
	NumSubmissions  uint64         `gorm:"not null;default 0"`
}
