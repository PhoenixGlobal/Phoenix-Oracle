package gracefulpanic

import (
	"runtime/debug"

	"PhoenixOracle/lib/logger"
)

func WrapRecover(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			logger.Default.Errorw("goroutine panicked", "panic", err, "stacktrace", string(debug.Stack()))
		}
	}()
	fn()
}
