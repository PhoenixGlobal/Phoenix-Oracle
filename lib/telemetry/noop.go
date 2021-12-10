package telemetry

import (
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"github.com/ethereum/go-ethereum/common"
)

type NoopAgent struct {
}

func (t *NoopAgent) SendLog(log []byte) {
}

func (t *NoopAgent) GenMonitoringEndpoint(addr common.Address) ocrtypes.MonitoringEndpoint {
	return t
}
