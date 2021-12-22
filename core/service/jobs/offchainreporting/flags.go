package offchainreporting

import (
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/internal/gethwrappers/generated/flags_wrapper"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type ContractFlags struct {
	flags_wrapper.FlagsInterface
}

func NewFlags(addrHex string, ethClient ethereum.Client) (*ContractFlags, error) {
	flags := &ContractFlags{}

	if addrHex == "" {
		return flags, nil
	}

	contractAddr := common.HexToAddress(addrHex)
	contract, err := flags_wrapper.NewFlags(contractAddr, ethClient)
	if err != nil {
		return flags, errors.Wrap(err, "Failed to create flags wrapper")
	}
	flags.FlagsInterface = contract
	return flags, nil
}

func (f *ContractFlags) Contract() flags_wrapper.FlagsInterface {
	return f.FlagsInterface
}

func (f *ContractFlags) ContractExists() bool {
	return f.FlagsInterface != nil
}

func (f *ContractFlags) IsLowered(contractAddr common.Address) (bool, error) {
	if !f.ContractExists() {
		return true, nil
	}

	flags, err := f.GetFlags(nil,
		[]common.Address{utils.ZeroAddress, contractAddr},
	)
	if err != nil {
		return true, errors.Wrap(err, "Failed to call GetFlags in the contract")
	}

	return !flags[0] || !flags[1], nil
}
