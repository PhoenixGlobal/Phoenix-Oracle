package telemetry

import (
	"github.com/ethereum/go-ethereum/common"
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
)

type MonitoringEndpointGenerator interface {
	GenMonitoringEndpoint(addr common.Address) ocrtypes.MonitoringEndpoint
}
