package fluxmonitor

import (
	"math/big"

	"github.com/shopspring/decimal"
)

type SubmissionChecker struct {
	Min decimal.Decimal
	Max decimal.Decimal
}

func NewSubmissionChecker(min *big.Int, max *big.Int) *SubmissionChecker {
	return &SubmissionChecker{
		Min: decimal.NewFromBigInt(min, 0),
		Max: decimal.NewFromBigInt(max, 0),
	}
}

func (c *SubmissionChecker) IsValid(answer decimal.Decimal) bool {
	return answer.GreaterThanOrEqual(c.Min) && answer.LessThanOrEqual(c.Max)
}
