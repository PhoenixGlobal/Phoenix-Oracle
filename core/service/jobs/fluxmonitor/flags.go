package fluxmonitor

import (
	"reflect"

	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/internal/gethwrappers/generated"
	"PhoenixOracle/internal/gethwrappers/generated/flags_wrapper"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Flags interface {
	ContractExists() bool
	IsLowered(contractAddr common.Address) (bool, error)
	Address() common.Address
	ParseLog(log types.Log) (generated.AbigenLog, error)
}

type ContractFlags struct {
	flags_wrapper.FlagsInterface
}

func NewFlags(addrHex string, ethClient ethereum.Client) (Flags, error) {
	flags := &ContractFlags{}

	if addrHex == "" {
		return flags, nil
	}

	contractAddr := common.HexToAddress(addrHex)
	contract, err := flags_wrapper.NewFlags(contractAddr, ethClient)
	if err != nil {
		return flags, err
	}

	if contract != nil && !reflect.ValueOf(contract).IsNil() {
		flags.FlagsInterface = contract
	}

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
		return true, err
	}

	return !flags[0] || !flags[1], nil
}
