package offchainreporting

import (
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/logger"
	"go.uber.org/zap"
)

var _ ocrtypes.Logger = &ocrLogger{}

type ocrLogger struct {
	internal  *logger.Logger
	trace     bool
	saveError func(string)
}

func NewLogger(l *logger.Logger, trace bool, saveError func(string)) ocrtypes.Logger {
	internal := logger.CreateLogger(l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar())
	return &ocrLogger{
		internal:  internal,
		trace:     trace,
		saveError: saveError,
	}
}

func (ol *ocrLogger) Trace(msg string, fields ocrtypes.LogFields) {
	if ol.trace {
		ol.internal.Debugw(msg, toKeysAndValues(fields)...)
	}
}

func (ol *ocrLogger) Debug(msg string, fields ocrtypes.LogFields) {
	ol.internal.Debugw(msg, toKeysAndValues(fields)...)
}

func (ol *ocrLogger) Info(msg string, fields ocrtypes.LogFields) {
	ol.internal.Infow(msg, toKeysAndValues(fields)...)
}

func (ol *ocrLogger) Warn(msg string, fields ocrtypes.LogFields) {
	ol.internal.Warnw(msg, toKeysAndValues(fields)...)
}

func (ol *ocrLogger) Error(msg string, fields ocrtypes.LogFields) {
	ol.saveError(msg)
	ol.internal.Errorw(msg, toKeysAndValues(fields)...)
}

func toKeysAndValues(fields ocrtypes.LogFields) []interface{} {
	out := []interface{}{}
	for key, val := range fields {
		out = append(out, key, val)
	}
	return out
}
