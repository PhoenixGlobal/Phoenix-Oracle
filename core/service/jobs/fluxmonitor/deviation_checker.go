package fluxmonitor

import (
	"PhoenixOracle/lib/logger"
	"github.com/shopspring/decimal"
)

type DeviationThresholds struct {
	Rel float64 // |new-old|/|old| >= Rel
	Abs float64 // |new-old| >= Abs
}

type DeviationChecker struct {
	Thresholds DeviationThresholds
}

func NewDeviationChecker(rel, abs float64) *DeviationChecker {
	return &DeviationChecker{
		Thresholds: DeviationThresholds{
			Rel: rel,
			Abs: abs,
		},
	}
}

func NewZeroDeviationChecker() *DeviationChecker {
	return &DeviationChecker{
		Thresholds: DeviationThresholds{
			Rel: 0,
			Abs: 0,
		},
	}
}

func (c *DeviationChecker) OutsideDeviation(curAnswer, nextAnswer decimal.Decimal) bool {
	loggerFields := []interface{}{
		"threshold", c.Thresholds.Rel,
		"absoluteThreshold", c.Thresholds.Abs,
		"currentAnswer", curAnswer,
		"nextAnswer", nextAnswer,
	}

	if c.Thresholds.Rel == 0 && c.Thresholds.Abs == 0 {
		logger.Debugw(
			"Deviation thresholds both zero; short-circuiting deviation checker to "+
				"true, regardless of feed values", loggerFields...)
		return true
	}
	diff := curAnswer.Sub(nextAnswer).Abs()
	loggerFields = append(loggerFields, "absoluteDeviation", diff)

	if !diff.GreaterThan(decimal.NewFromFloat(c.Thresholds.Abs)) {
		logger.Debugw("Absolute deviation threshold not met", loggerFields...)
		return false
	}

	if curAnswer.IsZero() {
		if nextAnswer.IsZero() {
			logger.Debugw("Relative deviation is undefined; can't satisfy threshold", loggerFields...)
			return false
		}
		logger.Infow("Threshold met: relative deviation is ∞", loggerFields...)
		return true
	}

	// 100*|new-old|/|old|:  as a percentage
	percentage := diff.Div(curAnswer.Abs()).Mul(decimal.NewFromInt(100))

	loggerFields = append(loggerFields, "percentage", percentage)

	if percentage.LessThan(decimal.NewFromFloat(c.Thresholds.Rel)) {
		logger.Debugw("Relative deviation threshold not met", loggerFields...)
		return false
	}
	logger.Infow("Relative and absolute deviation thresholds both met", loggerFields...)
	return true
}
