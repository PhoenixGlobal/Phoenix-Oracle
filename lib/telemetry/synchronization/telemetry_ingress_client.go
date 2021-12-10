package synchronization

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"

	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/service"
	"PhoenixOracle/lib/logger"
	telemPb "PhoenixOracle/lib/telemetry/synchronization/telem"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/wsrpc"
	"github.com/smartcontractkit/wsrpc/examples/simple/keys"
)

const SendIngressBufferSize = 100

type TelemetryIngressClient interface {
	service.Service
	Start() error
	Close() error
	Send(TelemPayload)
	Unsafe_SetTelemClient(telemPb.TelemClient) bool
}

type NoopTelemetryIngressClient struct{}

func (NoopTelemetryIngressClient) Start() error                                   { return nil }
func (NoopTelemetryIngressClient) Close() error                                   { return nil }
func (NoopTelemetryIngressClient) Send(TelemPayload)                              {}
func (NoopTelemetryIngressClient) Healthy() error                                 { return nil }
func (NoopTelemetryIngressClient) Ready() error                                   { return nil }
func (NoopTelemetryIngressClient) Unsafe_SetTelemClient(telemPb.TelemClient) bool { return true }

type telemetryIngressClient struct {
	utils.StartStopOnce
	url             *url.URL
	ks              keystore.CSA
	serverPubKeyHex string

	telemClient telemPb.TelemClient
	logging     bool

	wgDone           sync.WaitGroup
	chDone           chan struct{}
	dropMessageCount uint32
	chTelemetry      chan TelemPayload
}

type TelemPayload struct {
	Ctx             context.Context
	Telemetry       []byte
	ContractAddress common.Address
}

func NewTelemetryIngressClient(url *url.URL, serverPubKeyHex string, ks keystore.CSA, logging bool) TelemetryIngressClient {
	return &telemetryIngressClient{
		url:             url,
		ks:              ks,
		serverPubKeyHex: serverPubKeyHex,
		logging:         logging,
		chTelemetry:     make(chan TelemPayload, SendIngressBufferSize),
		chDone:          make(chan struct{}),
	}
}

func (tc *telemetryIngressClient) Start() error {
	return tc.StartOnce("TelemetryIngressClient", func() error {
		privkey, err := tc.getCSAPrivateKey()
		if err != nil {
			return err
		}

		tc.connect(privkey)

		return nil
	})
}

func (tc *telemetryIngressClient) Close() error {
	return tc.StopOnce("TelemetryIngressClient", func() error {
		close(tc.chDone)
		tc.wgDone.Wait()
		return nil
	})
}

func (tc *telemetryIngressClient) connect(clientPrivKey []byte) {
	tc.wgDone.Add(1)

	go func() {
		defer tc.wgDone.Done()

		serverPubKey := keys.FromHex(tc.serverPubKeyHex)

		conn, err := wsrpc.Dial(tc.url.String(), wsrpc.WithTransportCreds(clientPrivKey, serverPubKey))
		if err != nil {
			logger.Errorf("Error connecting to telemetry ingress server: %v", err)
			return
		}
		defer conn.Close()

		tc.telemClient = telemPb.NewTelemClient(conn)

		tc.handleTelemetry()

		<-tc.chDone

	}()
}

func (tc *telemetryIngressClient) handleTelemetry() {
	go func() {
		for {
			select {
			case p := <-tc.chTelemetry:
				telemReq := &telemPb.TelemRequest{Telemetry: p.Telemetry, Address: p.ContractAddress.String()}
				_, err := tc.telemClient.Telem(p.Ctx, telemReq)
				if err != nil {
					logger.Errorf("Could not send telemetry: %v", err)
					continue
				}
				if tc.logging {
					logger.Debugw("successfully sent telemetry to ingress server", "contractAddress", p.ContractAddress.String(), "telemetry", p.Telemetry)
				}
			case <-tc.chDone:
				return
			}
		}
	}()
}

func (tc *telemetryIngressClient) logBufferFullWithExpBackoff(payload TelemPayload) {
	count := atomic.AddUint32(&tc.dropMessageCount, 1)
	if count > 0 && (count%100 == 0 || count&(count-1) == 0) {
		logger.Warnw("telemetry ingress client buffer full, dropping message", "telemetry", payload.Telemetry, "droppedCount", count)
	}
}

func (tc *telemetryIngressClient) getCSAPrivateKey() (privkey []byte, err error) {
	// Fetch the client's public key
	keys, err := tc.ks.GetAll()
	if err != nil {
		return privkey, err
	}
	if len(keys) < 1 {
		return privkey, errors.New("CSA key does not exist")
	}

	return keys[0].Raw(), nil
}

func (tc *telemetryIngressClient) Send(payload TelemPayload) {
	select {
	case tc.chTelemetry <- payload:
		atomic.StoreUint32(&tc.dropMessageCount, 0)
	case <-payload.Ctx.Done():
		return
	default:
		tc.logBufferFullWithExpBackoff(payload)
	}
}

func (tc *telemetryIngressClient) Unsafe_SetTelemClient(client telemPb.TelemClient) bool {
	if tc.telemClient == nil {
		return false
	}

	tc.telemClient = client
	return true
}
