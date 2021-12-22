package knockingtls

import (
	"fmt"
	"net"
	"time"

	"PhoenixOracle/lib/libocr/offchainreporting/types"
	"golang.org/x/time/rate"
)

type RateLimitedConn struct {
	net.Conn
	bandwidthLimiter    *rate.Limiter
	logger              types.Logger
	rateLimitingEnabled bool
}

var _ net.Conn = (*RateLimitedConn)(nil)

func NewRateLimitedConn(conn net.Conn, bandwidthLimiter *rate.Limiter, logger types.Logger) *RateLimitedConn {
	return &RateLimitedConn{
		conn,
		bandwidthLimiter,
		logger,
		false,
	}
}

// EnableRateLimiting is not thread-safe!
func (r *RateLimitedConn) EnableRateLimiting() {
	r.rateLimitingEnabled = true
}

func (r *RateLimitedConn) Read(b []byte) (n int, err error) {
	n, err = r.Conn.Read(b)
	if !r.rateLimitingEnabled {
		return n, err
	}
	nBytesAllowed := r.bandwidthLimiter.AllowN(time.Now(), n)
	if nBytesAllowed {
		return n, err
	}
	// kill the conn: close it and emit an error
	_ = r.Conn.Close() // ignore error, there's not much we can with it here
	r.logger.Error("inbound data exceeded rate limit, connection closed", types.LogFields{
		"tokenBucketRefillRate": r.bandwidthLimiter.Limit(),
		"tokenBucketSize":       r.bandwidthLimiter.Burst(),
		"bytesRead":             n,
		"readError":             err, // This error may not be null, we're adding it here to not miss it.
	})
	return 0, fmt.Errorf("inbound data exceeded rate limit, connection closed")
}
