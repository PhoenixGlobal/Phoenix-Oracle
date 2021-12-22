package telemetry

import (
	"context"

	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/telemetry/synchronization"
	"github.com/ethereum/go-ethereum/common"
)

type IngressAgentWrapper struct {
	telemetryIngressClient synchronization.TelemetryIngressClient
}

func NewIngressAgentWrapper(telemetryIngressClient synchronization.TelemetryIngressClient) *IngressAgentWrapper {
	return &IngressAgentWrapper{telemetryIngressClient}
}

func (t *IngressAgentWrapper) GenMonitoringEndpoint(addr common.Address) ocrtypes.MonitoringEndpoint {
	return NewIngressAgent(t.telemetryIngressClient, addr)
}

type IngressAgent struct {
	telemetryIngressClient synchronization.TelemetryIngressClient
	contractAddress        common.Address
}

func NewIngressAgent(telemetryIngressClient synchronization.TelemetryIngressClient, contractAddress common.Address) *IngressAgent {
	return &IngressAgent{
		telemetryIngressClient,
		contractAddress,
	}
}

func (t *IngressAgent) SendLog(telemetry []byte) {
	payload := synchronization.TelemPayload{
		Ctx:             context.Background(),
		Telemetry:       telemetry,
		ContractAddress: t.contractAddress,
	}
	t.telemetryIngressClient.Send(payload)
}
