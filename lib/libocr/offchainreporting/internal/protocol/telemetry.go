package protocol

import "PhoenixOracle/lib/libocr/offchainreporting/types"

type TelemetrySender interface {
	RoundStarted(
		configDigest types.ConfigDigest,
		epoch uint32,
		round uint8,
		leader types.OracleID,
	)
}
