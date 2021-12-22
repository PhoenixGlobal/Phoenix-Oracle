// Package shim contains implementations of internal types in terms of the external types
package shim

import (
	"sync"

	"github.com/pkg/errors"
	"PhoenixOracle/lib/libocr/offchainreporting/internal/protocol"
	"PhoenixOracle/lib/libocr/offchainreporting/internal/serialization"
	"PhoenixOracle/lib/libocr/offchainreporting/internal/serialization/protobuf"
	"PhoenixOracle/lib/libocr/offchainreporting/loghelper"
	"PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/libocr/subprocesses"
)

type SerializingEndpoint struct {
	chTelemetry  chan<- *protobuf.TelemetryWrapper
	configDigest types.ConfigDigest
	endpoint     types.BinaryNetworkEndpoint
	logger       types.Logger
	mutex        sync.Mutex
	subprocesses subprocesses.Subprocesses
	started      bool
	closed       bool
	closedChOut  bool
	chCancel     chan struct{}
	chOut        chan protocol.MessageWithSender
	taper        loghelper.LogarithmicTaper
}

var _ protocol.NetworkEndpoint = (*SerializingEndpoint)(nil)

func NewSerializingEndpoint(
	chTelemetry chan<- *protobuf.TelemetryWrapper,
	configDigest types.ConfigDigest,
	endpoint types.BinaryNetworkEndpoint,
	logger types.Logger,
) *SerializingEndpoint {
	return &SerializingEndpoint{
		chTelemetry,
		configDigest,
		endpoint,
		logger,
		sync.Mutex{},
		subprocesses.Subprocesses{},
		false,
		false,
		false,
		make(chan struct{}),
		make(chan protocol.MessageWithSender),
		loghelper.LogarithmicTaper{},
	}
}

func (n *SerializingEndpoint) sendTelemetry(t *protobuf.TelemetryWrapper) {
	select {
	case n.chTelemetry <- t:
		n.taper.Reset(func(oldCount uint64) {
			n.logger.Info("SerializingEndpoint: stopped dropping telemetry", types.LogFields{
				"droppedCount": oldCount,
			})
		})
	default:
		n.taper.Trigger(func(newCount uint64) {
			n.logger.Warn("SerializingEndpoint: dropping telemetry", types.LogFields{
				"droppedCount": newCount,
			})
		})
	}
}

func (n *SerializingEndpoint) serialize(msg protocol.Message) ([]byte, *protobuf.MessageWrapper) {
	sMsg, pbm, err := serialization.Serialize(msg)
	if err != nil {
		n.logger.Error("SerializingEndpoint: Failed to serialize", types.LogFields{
			"message": msg,
		})
		return nil, nil
	}
	return sMsg, pbm
}

// Start starts the SerializingEndpoint. It will also start the underlying endpoint.
func (n *SerializingEndpoint) Start() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.started {
		panic("Cannot start already started SerializingEndpoint")
	}
	n.started = true

	if err := n.endpoint.Start(); err != nil {
		return errors.Wrap(err, "while starting SerializingEndpoint")
	}

	n.subprocesses.Go(func() {
		chRaw := n.endpoint.Receive()
		for {
			select {
			case raw, ok := <-chRaw:
				if !ok {
					n.mutex.Lock()
					defer n.mutex.Unlock()
					n.closedChOut = true
					close(n.chOut)
					return
				}

				m, pbm, err := serialization.Deserialize(raw.Msg)
				if err != nil {
					n.logger.Error("SerializingEndpoint: Failed to deserialize", types.LogFields{
						"message": raw,
					})
					n.sendTelemetry(&protobuf.TelemetryWrapper{
						Wrapped: &protobuf.TelemetryWrapper_AssertionViolation{&protobuf.TelemetryAssertionViolation{
							Violation: &protobuf.TelemetryAssertionViolation_InvalidSerialization{&protobuf.TelemetryAssertionViolationInvalidSerialization{
								ConfigDigest:  n.configDigest[:],
								SerializedMsg: raw.Msg,
								Sender:        uint32(raw.Sender),
							}},
						}},
					})
					break
				}

				n.sendTelemetry(&protobuf.TelemetryWrapper{
					Wrapped: &protobuf.TelemetryWrapper_MessageReceived{&protobuf.TelemetryMessageReceived{
						ConfigDigest: n.configDigest[:],
						Msg:          pbm,
						Sender:       uint32(raw.Sender),
					}},
				})

				select {
				case n.chOut <- protocol.MessageWithSender{m, raw.Sender}:
				case <-n.chCancel:
					return
				}
			case <-n.chCancel:
				return
			}
		}
	})

	return nil
}

// Close closes the SerializingEndpoint. It will also close the underlying endpoint.
func (n *SerializingEndpoint) Close() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.started && !n.closed {
		n.closed = true
		close(n.chCancel)
		n.subprocesses.Wait()

		if !n.closedChOut {
			n.closedChOut = true
			close(n.chOut)
		}

		return n.endpoint.Close()
	}

	return nil
}

func (n *SerializingEndpoint) SendTo(msg protocol.Message, to types.OracleID) {
	sMsg, pbm := n.serialize(msg)
	if sMsg != nil {
		n.endpoint.SendTo(sMsg, to)
		n.sendTelemetry(&protobuf.TelemetryWrapper{
			Wrapped: &protobuf.TelemetryWrapper_MessageSent{&protobuf.TelemetryMessageSent{
				ConfigDigest:  n.configDigest[:],
				Msg:           pbm,
				SerializedMsg: sMsg,
				Receiver:      uint32(to),
			}},
		})
	}
}

func (n *SerializingEndpoint) Broadcast(msg protocol.Message) {
	sMsg, pbm := n.serialize(msg)
	if sMsg != nil {
		n.endpoint.Broadcast(sMsg)
		n.sendTelemetry(&protobuf.TelemetryWrapper{
			Wrapped: &protobuf.TelemetryWrapper_MessageBroadcast{&protobuf.TelemetryMessageBroadcast{
				ConfigDigest:  n.configDigest[:],
				Msg:           pbm,
				SerializedMsg: sMsg,
			}},
		})
	}
}

func (n *SerializingEndpoint) Receive() <-chan protocol.MessageWithSender {
	return n.chOut
}
