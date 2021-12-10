package fluxmonitor

import (
	"math/big"

	"gorm.io/gorm"

	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/internal/gethwrappers/generated/flux_aggregator_wrapper"
	"github.com/pkg/errors"
)

var FluxAggregatorABI = ethereum.MustGetABI(flux_aggregator_wrapper.FluxAggregatorABI)

type ContractSubmitter interface {
	Submit(db *gorm.DB, roundID *big.Int, submission *big.Int) error
}

type FluxAggregatorContractSubmitter struct {
	flux_aggregator_wrapper.FluxAggregatorInterface
	orm      ORM
	keyStore KeyStoreInterface
	gasLimit uint64
}

func NewFluxAggregatorContractSubmitter(
	contract flux_aggregator_wrapper.FluxAggregatorInterface,
	orm ORM,
	keyStore KeyStoreInterface,
	gasLimit uint64,
) *FluxAggregatorContractSubmitter {
	return &FluxAggregatorContractSubmitter{
		FluxAggregatorInterface: contract,
		orm:                     orm,
		keyStore:                keyStore,
		gasLimit:                gasLimit,
	}
}

func (c *FluxAggregatorContractSubmitter) Submit(db *gorm.DB, roundID *big.Int, submission *big.Int) error {
	fromAddress, err := c.keyStore.GetRoundRobinAddress()
	if err != nil {
		return err
	}

	payload, err := FluxAggregatorABI.Pack("submit", roundID, submission)
	if err != nil {
		return errors.Wrap(err, "abi.Pack failed")
	}

	return errors.Wrap(
		c.orm.CreateEthTransaction(db, fromAddress, c.Address(), payload, c.gasLimit),
		"failed to send Eth transaction",
	)
}
