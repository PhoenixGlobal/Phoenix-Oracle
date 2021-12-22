package fluxmonitor

import (
	"math/big"

	"PhoenixOracle/core/assets"
)

const MinFundedRounds int64 = 3

type PaymentChecker struct {

	MinContractPayment *assets.Phb

	MinJobPayment *assets.Phb
}

func NewPaymentChecker(minContractPayment, minJobPayment *assets.Phb) *PaymentChecker {
	return &PaymentChecker{
		MinContractPayment: minContractPayment,
		MinJobPayment:      minJobPayment,
	}
}

func (c *PaymentChecker) SufficientFunds(availableFunds *big.Int, paymentAmount *big.Int, oracleCount uint8) bool {
	min := big.NewInt(int64(oracleCount))
	min = min.Mul(min, big.NewInt(MinFundedRounds))
	min = min.Mul(min, paymentAmount)

	return availableFunds.Cmp(min) >= 0
}

func (c *PaymentChecker) SufficientPayment(payment *big.Int) bool {
	aboveOrEqMinGlobalPayment := payment.Cmp(c.MinContractPayment.ToInt()) >= 0
	aboveOrEqMinJobPayment := true
	if c.MinJobPayment != nil {
		aboveOrEqMinJobPayment = payment.Cmp(c.MinJobPayment.ToInt()) >= 0
	}
	return aboveOrEqMinGlobalPayment && aboveOrEqMinJobPayment
}
