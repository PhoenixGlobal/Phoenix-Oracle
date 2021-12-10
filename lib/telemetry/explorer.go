package telemetry

import (
	"context"

	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/telemetry/synchronization"
	"github.com/ethereum/go-ethereum/common"
)

type ExplorerAgent struct {
	explorerClient synchronization.ExplorerClient
}

func NewExplorerAgent(explorerClient synchronization.ExplorerClient) *ExplorerAgent {
	return &ExplorerAgent{explorerClient}
}

func (t *ExplorerAgent) SendLog(log []byte) {
	t.explorerClient.Send(context.Background(), log, synchronization.ExplorerBinaryMessage)
}

func (t *ExplorerAgent) GenMonitoringEndpoint(addr common.Address) ocrtypes.MonitoringEndpoint {
	return t
}
