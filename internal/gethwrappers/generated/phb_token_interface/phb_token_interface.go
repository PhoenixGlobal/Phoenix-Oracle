// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package phb_token_interface

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"PhoenixOracle/internal/gethwrappers/generated"
)

var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

var PhbTokenMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"transferAndCall\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseApproval\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseApproval\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"remaining\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"}]",
	Bin: "0x6060604052341561000f57600080fd5b5b600160a060020a03331660009081526001602052604090206b033b2e3c9fd0803ce800000090555b5b610c51806100486000396000f300606060405236156100b75763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166306fdde0381146100bc578063095ea7b31461014757806318160ddd1461017d57806323b872dd146101a2578063313ce567146101de5780634000aea014610207578063661884631461028057806370a08231146102b657806395d89b41146102e7578063a9059cbb14610372578063d73dd623146103a8578063dd62ed3e146103de575b600080fd5b34156100c757600080fd5b6100cf610415565b60405160208082528190810183818151815260200191508051906020019080838360005b8381101561010c5780820151818401525b6020016100f3565b50505050905090810190601f1680156101395780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015257600080fd5b610169600160a060020a036004351660243561044c565b604051901515815260200160405180910390f35b341561018857600080fd5b610190610499565b60405190815260200160405180910390f35b34156101ad57600080fd5b610169600160a060020a03600435811690602435166044356104a9565b604051901515815260200160405180910390f35b34156101e957600080fd5b6101f16104f8565b60405160ff909116815260200160405180910390f35b341561021257600080fd5b61016960048035600160a060020a03169060248035919060649060443590810190830135806020601f820181900481020160405190810160405281815292919060208401838380828437509496506104fd95505050505050565b604051901515815260200160405180910390f35b341561028b57600080fd5b610169600160a060020a036004351660243561054c565b604051901515815260200160405180910390f35b34156102c157600080fd5b610190600160a060020a0360043516610648565b60405190815260200160405180910390f35b34156102f257600080fd5b6100cf610667565b60405160208082528190810183818151815260200191508051906020019080838360005b8381101561010c5780820151818401525b6020016100f3565b50505050905090810190601f1680156101395780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561037d57600080fd5b610169600160a060020a036004351660243561069e565b604051901515815260200160405180910390f35b34156103b357600080fd5b610169600160a060020a03600435166024356106eb565b604051901515815260200160405180910390f35b34156103e957600080fd5b610190600160a060020a0360043581169060243516610790565b60405190815260200160405180910390f35b60408051908101604052600f81527f436861696e4c696e6b20546f6b656e0000000000000000000000000000000000602082015281565b600082600160a060020a03811615801590610479575030600160a060020a031681600160a060020a031614155b151561048457600080fd5b61048e84846107bd565b91505b5b5092915050565b6b033b2e3c9fd0803ce800000081565b600082600160a060020a038116158015906104d6575030600160a060020a031681600160a060020a031614155b15156104e157600080fd5b6104ec85858561082a565b91505b5b509392505050565b601281565b600083600160a060020a0381161580159061052a575030600160a060020a031681600160a060020a031614155b151561053557600080fd5b6104ec85858561093c565b91505b5b509392505050565b600160a060020a033381166000908152600260209081526040808320938616835292905290812054808311156105a957600160a060020a0333811660009081526002602090815260408083209388168352929052908120556105e0565b6105b9818463ffffffff610a2316565b600160a060020a033381166000908152600260209081526040808320938916835292905220555b600160a060020a0333811660008181526002602090815260408083209489168084529490915290819020547f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925915190815260200160405180910390a3600191505b5092915050565b600160a060020a0381166000908152600160205260409020545b919050565b60408051908101604052600481527f4c494e4b00000000000000000000000000000000000000000000000000000000602082015281565b600082600160a060020a038116158015906106cb575030600160a060020a031681600160a060020a031614155b15156106d657600080fd5b61048e8484610a3a565b91505b5b5092915050565b600160a060020a033381166000908152600260209081526040808320938616835292905290812054610723908363ffffffff610afa16565b600160a060020a0333811660008181526002602090815260408083209489168084529490915290819020849055919290917f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591905190815260200160405180910390a35060015b92915050565b600160a060020a038083166000908152600260209081526040808320938516835292905220545b92915050565b600160a060020a03338116600081815260026020908152604080832094871680845294909152808220859055909291907f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9259085905190815260200160405180910390a35060015b92915050565b600160a060020a03808416600081815260026020908152604080832033909516835293815283822054928252600190529182205461086e908463ffffffff610a2316565b600160a060020a0380871660009081526001602052604080822093909355908616815220546108a3908463ffffffff610afa16565b600160a060020a0385166000908152600160205260409020556108cc818463ffffffff610a2316565b600160a060020a03808716600081815260026020908152604080832033861684529091529081902093909355908616917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9086905190815260200160405180910390a3600191505b509392505050565b60006109488484610a3a565b5083600160a060020a031633600160a060020a03167fe19260aff97b920c7df27010903aeb9c8d2be5d310a2c67824cf3f15396e4c16858560405182815260406020820181815290820183818151815260200191508051906020019080838360005b838110156109c35780820151818401525b6020016109aa565b50505050905090810190601f1680156109f05780820380516001836020036101000a031916815260200191505b50935050505060405180910390a3610a0784610b14565b15610a1757610a17848484610b23565b5b5060015b9392505050565b600082821115610a2f57fe5b508082035b92915050565b600160a060020a033316600090815260016020526040812054610a63908363ffffffff610a2316565b600160a060020a033381166000908152600160205260408082209390935590851681522054610a98908363ffffffff610afa16565b600160a060020a0380851660008181526001602052604090819020939093559133909116907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9085905190815260200160405180910390a35060015b92915050565b600082820183811015610b0957fe5b8091505b5092915050565b6000813b908111905b50919050565b82600160a060020a03811663a4c0ed363385856040518463ffffffff167c01000000000000000000000000000000000000000000000000000000000281526004018084600160a060020a0316600160a060020a0316815260200183815260200180602001828103825283818151815260200191508051906020019080838360005b83811015610bbd5780820151818401525b602001610ba4565b50505050905090810190601f168015610bea5780820380516001836020036101000a031916815260200191505b50945050505050600060405180830381600087803b1515610c0a57600080fd5b6102c65a03f11515610c1b57600080fd5b5050505b505050505600a165627a7a72305820c5f438ff94e5ddaf2058efa0019e246c636c37a622e04bb67827c7374acad8d60029",
}

var PhbTokenABI = PhbTokenMetaData.ABI

var PhbTokenBin = PhbTokenMetaData.Bin

func DeployPhbToken(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PhbToken, error) {
	parsed, err := PhbTokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PhbTokenBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PhbToken{PhbTokenCaller: PhbTokenCaller{contract: contract}, PhbTokenTransactor: PhbTokenTransactor{contract: contract}, PhbTokenFilterer: PhbTokenFilterer{contract: contract}}, nil
}

type PhbToken struct {
	address common.Address
	abi     abi.ABI
	PhbTokenCaller
	PhbTokenTransactor
	PhbTokenFilterer
}

type PhbTokenCaller struct {
	contract *bind.BoundContract
}

type PhbTokenTransactor struct {
	contract *bind.BoundContract
}

type PhbTokenFilterer struct {
	contract *bind.BoundContract
}

type PhbTokenSession struct {
	Contract     *PhbToken
	CallOpts     bind.CallOpts
	TransactOpts bind.TransactOpts
}

type PhbTokenCallerSession struct {
	Contract *PhbTokenCaller
	CallOpts bind.CallOpts
}

type PhbTokenTransactorSession struct {
	Contract     *PhbTokenTransactor
	TransactOpts bind.TransactOpts
}

type PhbTokenRaw struct {
	Contract *PhbToken
}

type PhbTokenCallerRaw struct {
	Contract *PhbTokenCaller
}

type PhbTokenTransactorRaw struct {
	Contract *PhbTokenTransactor
}

func NewPhbToken(address common.Address, backend bind.ContractBackend) (*PhbToken, error) {
	abi, err := abi.JSON(strings.NewReader(PhbTokenABI))
	if err != nil {
		return nil, err
	}
	contract, err := bindPhbToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PhbToken{address: address, abi: abi, PhbTokenCaller: PhbTokenCaller{contract: contract}, PhbTokenTransactor: PhbTokenTransactor{contract: contract}, PhbTokenFilterer: PhbTokenFilterer{contract: contract}}, nil
}

func NewPhbTokenCaller(address common.Address, caller bind.ContractCaller) (*PhbTokenCaller, error) {
	contract, err := bindPhbToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PhbTokenCaller{contract: contract}, nil
}

func NewPhbTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*PhbTokenTransactor, error) {
	contract, err := bindPhbToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PhbTokenTransactor{contract: contract}, nil
}

func NewPhbTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*PhbTokenFilterer, error) {
	contract, err := bindPhbToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PhbTokenFilterer{contract: contract}, nil
}

func bindPhbToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PhbTokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

func (_PhbToken *PhbTokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PhbToken.Contract.PhbTokenCaller.contract.Call(opts, result, method, params...)
}

func (_PhbToken *PhbTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PhbToken.Contract.PhbTokenTransactor.contract.Transfer(opts)
}

func (_PhbToken *PhbTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PhbToken.Contract.PhbTokenTransactor.contract.Transact(opts, method, params...)
}

func (_PhbToken *PhbTokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PhbToken.Contract.contract.Call(opts, result, method, params...)
}

func (_PhbToken *PhbTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PhbToken.Contract.contract.Transfer(opts)
}

func (_PhbToken *PhbTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PhbToken.Contract.contract.Transact(opts, method, params...)
}

func (_PhbToken *PhbTokenCaller) Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PhbToken.contract.Call(opts, &out, "allowance", _owner, _spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

func (_PhbToken *PhbTokenSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _PhbToken.Contract.Allowance(&_PhbToken.CallOpts, _owner, _spender)
}

func (_PhbToken *PhbTokenCallerSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _PhbToken.Contract.Allowance(&_PhbToken.CallOpts, _owner, _spender)
}

func (_PhbToken *PhbTokenCaller) BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PhbToken.contract.Call(opts, &out, "balanceOf", _owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

func (_PhbToken *PhbTokenSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _PhbToken.Contract.BalanceOf(&_PhbToken.CallOpts, _owner)
}

func (_PhbToken *PhbTokenCallerSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _PhbToken.Contract.BalanceOf(&_PhbToken.CallOpts, _owner)
}

func (_PhbToken *PhbTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _PhbToken.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

func (_PhbToken *PhbTokenSession) Decimals() (uint8, error) {
	return _PhbToken.Contract.Decimals(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenCallerSession) Decimals() (uint8, error) {
	return _PhbToken.Contract.Decimals(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _PhbToken.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

func (_PhbToken *PhbTokenSession) Name() (string, error) {
	return _PhbToken.Contract.Name(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenCallerSession) Name() (string, error) {
	return _PhbToken.Contract.Name(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _PhbToken.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

func (_PhbToken *PhbTokenSession) Symbol() (string, error) {
	return _PhbToken.Contract.Symbol(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenCallerSession) Symbol() (string, error) {
	return _PhbToken.Contract.Symbol(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PhbToken.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

func (_PhbToken *PhbTokenSession) TotalSupply() (*big.Int, error) {
	return _PhbToken.Contract.TotalSupply(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenCallerSession) TotalSupply() (*big.Int, error) {
	return _PhbToken.Contract.TotalSupply(&_PhbToken.CallOpts)
}

func (_PhbToken *PhbTokenTransactor) Approve(opts *bind.TransactOpts, _spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.contract.Transact(opts, "approve", _spender, _value)
}

func (_PhbToken *PhbTokenSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.Approve(&_PhbToken.TransactOpts, _spender, _value)
}

func (_PhbToken *PhbTokenTransactorSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.Approve(&_PhbToken.TransactOpts, _spender, _value)
}

func (_PhbToken *PhbTokenTransactor) DecreaseApproval(opts *bind.TransactOpts, _spender common.Address, _subtractedValue *big.Int) (*types.Transaction, error) {
	return _PhbToken.contract.Transact(opts, "decreaseApproval", _spender, _subtractedValue)
}

func (_PhbToken *PhbTokenSession) DecreaseApproval(_spender common.Address, _subtractedValue *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.DecreaseApproval(&_PhbToken.TransactOpts, _spender, _subtractedValue)
}

func (_PhbToken *PhbTokenTransactorSession) DecreaseApproval(_spender common.Address, _subtractedValue *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.DecreaseApproval(&_PhbToken.TransactOpts, _spender, _subtractedValue)
}

func (_PhbToken *PhbTokenTransactor) IncreaseApproval(opts *bind.TransactOpts, _spender common.Address, _addedValue *big.Int) (*types.Transaction, error) {
	return _PhbToken.contract.Transact(opts, "increaseApproval", _spender, _addedValue)
}

func (_PhbToken *PhbTokenSession) IncreaseApproval(_spender common.Address, _addedValue *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.IncreaseApproval(&_PhbToken.TransactOpts, _spender, _addedValue)
}

func (_PhbToken *PhbTokenTransactorSession) IncreaseApproval(_spender common.Address, _addedValue *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.IncreaseApproval(&_PhbToken.TransactOpts, _spender, _addedValue)
}

func (_PhbToken *PhbTokenTransactor) Transfer(opts *bind.TransactOpts, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.contract.Transact(opts, "transfer", _to, _value)
}

func (_PhbToken *PhbTokenSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.Transfer(&_PhbToken.TransactOpts, _to, _value)
}

func (_PhbToken *PhbTokenTransactorSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.Transfer(&_PhbToken.TransactOpts, _to, _value)
}

func (_PhbToken *PhbTokenTransactor) TransferAndCall(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _data []byte) (*types.Transaction, error) {
	return _PhbToken.contract.Transact(opts, "transferAndCall", _to, _value, _data)
}

func (_PhbToken *PhbTokenSession) TransferAndCall(_to common.Address, _value *big.Int, _data []byte) (*types.Transaction, error) {
	return _PhbToken.Contract.TransferAndCall(&_PhbToken.TransactOpts, _to, _value, _data)
}

func (_PhbToken *PhbTokenTransactorSession) TransferAndCall(_to common.Address, _value *big.Int, _data []byte) (*types.Transaction, error) {
	return _PhbToken.Contract.TransferAndCall(&_PhbToken.TransactOpts, _to, _value, _data)
}

func (_PhbToken *PhbTokenTransactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.contract.Transact(opts, "transferFrom", _from, _to, _value)
}

func (_PhbToken *PhbTokenSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.TransferFrom(&_PhbToken.TransactOpts, _from, _to, _value)
}

func (_PhbToken *PhbTokenTransactorSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _PhbToken.Contract.TransferFrom(&_PhbToken.TransactOpts, _from, _to, _value)
}

type PhbTokenApprovalIterator struct {
	Event *PhbTokenApproval

	contract *bind.BoundContract
	event    string

	logs chan types.Log
	sub  ethereum.Subscription
	done bool
	fail error
}

func (it *PhbTokenApprovalIterator) Next() bool {

	if it.fail != nil {
		return false
	}

	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PhbTokenApproval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}

	select {
	case log := <-it.logs:
		it.Event = new(PhbTokenApproval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

func (it *PhbTokenApprovalIterator) Error() error {
	return it.fail
}

func (it *PhbTokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

type PhbTokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log
}

func (_PhbToken *PhbTokenFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*PhbTokenApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _PhbToken.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &PhbTokenApprovalIterator{contract: _PhbToken.contract, event: "Approval", logs: logs, sub: sub}, nil
}

func (_PhbToken *PhbTokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *PhbTokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _PhbToken.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:

				event := new(PhbTokenApproval)
				if err := _PhbToken.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

func (_PhbToken *PhbTokenFilterer) ParseApproval(log types.Log) (*PhbTokenApproval, error) {
	event := new(PhbTokenApproval)
	if err := _PhbToken.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

type PhbTokenTransferIterator struct {
	Event *PhbTokenTransfer

	contract *bind.BoundContract
	event    string

	logs chan types.Log
	sub  ethereum.Subscription
	done bool
	fail error
}

func (it *PhbTokenTransferIterator) Next() bool {

	if it.fail != nil {
		return false
	}

	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PhbTokenTransfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}

	select {
	case log := <-it.logs:
		it.Event = new(PhbTokenTransfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

func (it *PhbTokenTransferIterator) Error() error {
	return it.fail
}

func (it *PhbTokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

type PhbTokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Data  []byte
	Raw   types.Log
}

func (_PhbToken *PhbTokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*PhbTokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _PhbToken.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &PhbTokenTransferIterator{contract: _PhbToken.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

func (_PhbToken *PhbTokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *PhbTokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _PhbToken.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:

				event := new(PhbTokenTransfer)
				if err := _PhbToken.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

func (_PhbToken *PhbTokenFilterer) ParseTransfer(log types.Log) (*PhbTokenTransfer, error) {
	event := new(PhbTokenTransfer)
	if err := _PhbToken.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

func (_PhbToken *PhbToken) ParseLog(log types.Log) (generated.AbigenLog, error) {
	switch log.Topics[0] {
	case _PhbToken.abi.Events["Approval"].ID:
		return _PhbToken.ParseApproval(log)
	case _PhbToken.abi.Events["Transfer"].ID:
		return _PhbToken.ParseTransfer(log)

	default:
		return nil, fmt.Errorf("abigen wrapper received unknown log topic: %v", log.Topics[0])
	}
}

func (PhbTokenApproval) Topic() common.Hash {
	return common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
}

func (PhbTokenTransfer) Topic() common.Hash {
	return common.HexToHash("0xe19260aff97b920c7df27010903aeb9c8d2be5d310a2c67824cf3f15396e4c16")
}

func (_PhbToken *PhbToken) Address() common.Address {
	return _PhbToken.address
}

type PhbTokenInterface interface {
	Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error)

	BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error)

	Decimals(opts *bind.CallOpts) (uint8, error)

	Name(opts *bind.CallOpts) (string, error)

	Symbol(opts *bind.CallOpts) (string, error)

	TotalSupply(opts *bind.CallOpts) (*big.Int, error)

	Approve(opts *bind.TransactOpts, _spender common.Address, _value *big.Int) (*types.Transaction, error)

	DecreaseApproval(opts *bind.TransactOpts, _spender common.Address, _subtractedValue *big.Int) (*types.Transaction, error)

	IncreaseApproval(opts *bind.TransactOpts, _spender common.Address, _addedValue *big.Int) (*types.Transaction, error)

	Transfer(opts *bind.TransactOpts, _to common.Address, _value *big.Int) (*types.Transaction, error)

	TransferAndCall(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _data []byte) (*types.Transaction, error)

	TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error)

	FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*PhbTokenApprovalIterator, error)

	WatchApproval(opts *bind.WatchOpts, sink chan<- *PhbTokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error)

	ParseApproval(log types.Log) (*PhbTokenApproval, error)

	FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*PhbTokenTransferIterator, error)

	WatchTransfer(opts *bind.WatchOpts, sink chan<- *PhbTokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error)

	ParseTransfer(log types.Log) (*PhbTokenTransfer, error)

	ParseLog(log types.Log) (generated.AbigenLog, error)

	Address() common.Address
}
