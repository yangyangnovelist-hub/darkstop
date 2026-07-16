// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package darkstop

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// MockUSDT0MetaData contains all meta data concerning the MockUSDT0 contract.
var MockUSDT0MetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"mint\",\"inputs\":[{\"name\":\"_to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"_to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"_from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x60a060405234603257600e6040565b60146036565b610d8861004782396080518181816105f00152610b0c0152610d8890f35b603c565b60405190565b5f80fd5b3360805256fe60806040526004361015610013575b6107cf565b61001d5f356100cc565b806306fdde03146100c7578063095ea7b3146100c257806318160ddd146100bd57806323b872dd146100b8578063313ce567146100b357806340c10f19146100ae57806370a08231146100a95780638da5cb5b146100a457806395d89b411461009f578063a9059cbb1461009a5763dd62ed3e0361000e57610799565b6106ff565b6106ca565b610634565b6105b9565b610504565b6104ca565b610432565b6103c3565b610330565b610247565b60e01c90565b60405190565b5f80fd5b5f80fd5b5f9103126100ea57565b6100dc565b601f801991011690565b634e487b7160e01b5f52604160045260245ffd5b90610117906100ef565b810190811067ffffffffffffffff82111761013157604052565b6100f9565b906101496101426100d2565b928361010d565b565b67ffffffffffffffff8111610169576101656020916100ef565b0190565b6100f9565b9061018061017b8361014b565b610136565b918252565b5f7f4d6f636b20555344543000000000000000000000000000000000000000000000910152565b6101b6600a61016e565b906101c360208301610185565b565b6101cd6101ac565b90565b6101d86101c5565b90565b6101e36101d0565b90565b5190565b60209181520190565b90825f9392825e0152565b61021d61022660209361022b93610214816101e6565b938480936101ea565b958691016101f3565b6100ef565b0190565b6102449160208201915f8184039101526101fe565b90565b34610277576102573660046100e0565b6102736102626101db565b61026a6100d2565b9182918261022f565b0390f35b6100d8565b60018060a01b031690565b6102909061027c565b90565b61029c81610287565b036102a357565b5f80fd5b905035906102b482610293565b565b90565b6102c2816102b6565b036102c957565b5f80fd5b905035906102da826102b9565b565b919060408382031261030457806102f8610301925f86016102a7565b936020016102cd565b90565b6100dc565b151590565b61031790610309565b9052565b919061032e905f6020850194019061030e565b565b346103615761035d61034c6103463660046102dc565b90610831565b6103546100d2565b9182918261031b565b0390f35b6100d8565b1c90565b90565b61037d9060086103829302610366565b61036a565b90565b90610390915461036d565b90565b61039e5f5f90610385565b90565b6103aa906102b6565b9052565b91906103c1905f602085019401906103a1565b565b346103f3576103d33660046100e0565b6103ef6103de610393565b6103e66100d2565b918291826103ae565b0390f35b6100d8565b909160608284031261042d5761042a610413845f85016102a7565b9361042181602086016102a7565b936040016102cd565b90565b6100dc565b346104635761045f61044e6104483660046103f8565b9161098f565b6104566100d2565b9182918261031b565b0390f35b6100d8565b90565b60ff1690565b90565b61048861048361048d92610468565b610471565b61046b565b90565b61049a6006610474565b90565b6104a5610490565b90565b6104b19061046b565b9052565b91906104c8905f602085019401906104a8565b565b346104fa576104da3660046100e0565b6104f66104e561049d565b6104ed6100d2565b918291826104b5565b0390f35b6100d8565b5f0190565b346105335761051d6105173660046102dc565b90610aff565b6105256100d2565b8061052f816104ff565b0390f35b6100d8565b906020828203126105515761054e915f016102a7565b90565b6100dc565b61056a61056561056f9261027c565b610471565b61027c565b90565b61057b90610556565b90565b61058790610572565b90565b906105949061057e565b5f5260205260405f2090565b6105b6906105b16001915f9261058a565b610385565b90565b346105e9576105e56105d46105cf366004610538565b6105a0565b6105dc6100d2565b918291826103ae565b0390f35b6100d8565b7f000000000000000000000000000000000000000000000000000000000000000090565b61061b90610287565b9052565b9190610632905f60208501940190610612565b565b34610664576106443660046100e0565b61066061064f6105ee565b6106576100d2565b9182918261061f565b0390f35b6100d8565b5f7f5553445430000000000000000000000000000000000000000000000000000000910152565b61069a600561016e565b906106a760208301610669565b565b6106b1610690565b90565b6106bc6106a9565b90565b6106c76106b4565b90565b346106fa576106da3660046100e0565b6106f66106e56106bf565b6106ed6100d2565b9182918261022f565b0390f35b6100d8565b346107305761072c61071b6107153660046102dc565b90610bd8565b6107236100d2565b9182918261031b565b0390f35b6100d8565b919060408382031261075d578061075161075a925f86016102a7565b936020016102a7565b90565b6100dc565b9061076c9061057e565b5f5260205260405f2090565b6107916107969261078c6002935f94610762565b61058a565b610385565b90565b346107ca576107c66107b56107af366004610735565b90610778565b6107bd6100d2565b918291826103ae565b0390f35b6100d8565b5f80fd5b5f90565b5f1b90565b906107e85f19916107d7565b9181191691161790565b61080661080161080b926102b6565b610471565b6102b6565b90565b90565b9061082661082161082d926107f2565b61080e565b82546107dc565b9055565b9061083a6107d3565b5061085a8161085561084e60023390610762565b859061058a565b610811565b339190916108a661089461088e7f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9259361057e565b9361057e565b9361089d6100d2565b918291826103ae565b0390a3600190565b5f1c90565b6108bf6108c4916108ae565b61036a565b90565b6108d190546108b3565b90565b5f7f696e73756666696369656e7420616c6c6f77616e636500000000000000000000910152565b61090860166020926101ea565b610911816108d4565b0190565b61092a9060208101905f8183039101526108fb565b90565b1561093457565b61093c6100d2565b62461bcd60e51b81528061095260048201610915565b0390fd5b634e487b7160e01b5f52601160045260245ffd5b61097961097f919392936102b6565b926102b6565b820391821161098a57565b610956565b916109fb9261099c6107d3565b506109bb6109b66109af60028490610762565b339061058a565b6108c7565b6109d8816109d16109cb866102b6565b916102b6565b101561092d565b806109ec6109e65f196102b6565b916102b6565b036109fe575b50919091610c73565b90565b610a0c610a2791849061096a565b610a22610a1b60028590610762565b339061058a565b610811565b5f6109f2565b5f7f6e6f74206f776e65720000000000000000000000000000000000000000000000910152565b610a6160096020926101ea565b610a6a81610a2d565b0190565b610a839060208101905f818303910152610a54565b90565b15610a8d57565b610a956100d2565b62461bcd60e51b815280610aab60048201610a6e565b0390fd5b610abe610ac4919392936102b6565b926102b6565b8201809211610acf57565b610956565b90565b610aeb610ae6610af092610ad4565b610471565b61027c565b90565b610afc90610ad7565b90565b90610b3c33610b36610b307f0000000000000000000000000000000000000000000000000000000000000000610287565b91610287565b14610a86565b610b57610b5182610b4c5f6108c7565b610aaf565b5f610811565b610b7f81610b79610b6a6001869061058a565b91610b74836108c7565b610aaf565b90610811565b610b885f610af3565b919091610bd3610bc1610bbb7fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9361057e565b9361057e565b93610bca6100d2565b918291826103ae565b0390a3565b610bee91610be46107d3565b5033919091610c73565b90565b5f7f696e73756666696369656e742062616c616e6365000000000000000000000000910152565b610c2560146020926101ea565b610c2e81610bf1565b0190565b610c479060208101905f818303910152610c18565b90565b15610c5157565b610c596100d2565b62461bcd60e51b815280610c6f60048201610c32565b0390fd5b919091610c7e6107d3565b50610caf610c96610c916001849061058a565b6108c7565b610ca8610ca2856102b6565b916102b6565b1015610c4a565b610cd782610cd1610cc26001859061058a565b91610ccc836108c7565b61096a565b90610811565b610cff82610cf9610cea6001879061058a565b91610cf4836108c7565b610aaf565b90610811565b919091610d4a610d38610d327fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9361057e565b9361057e565b93610d416100d2565b918291826103ae565b0390a360019056fea264697066735822122090488857c0130059a8c9440279aa54592ceb2ebb6379d1738e484f87c8372baa64736f6c63430008230033",
}

// MockUSDT0ABI is the input ABI used to generate the binding from.
// Deprecated: Use MockUSDT0MetaData.ABI instead.
var MockUSDT0ABI = MockUSDT0MetaData.ABI

// MockUSDT0Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MockUSDT0MetaData.Bin instead.
var MockUSDT0Bin = MockUSDT0MetaData.Bin

// DeployMockUSDT0 deploys a new Ethereum contract, binding an instance of MockUSDT0 to it.
func DeployMockUSDT0(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MockUSDT0, error) {
	parsed, err := MockUSDT0MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MockUSDT0Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MockUSDT0{MockUSDT0Caller: MockUSDT0Caller{contract: contract}, MockUSDT0Transactor: MockUSDT0Transactor{contract: contract}, MockUSDT0Filterer: MockUSDT0Filterer{contract: contract}}, nil
}

// MockUSDT0 is an auto generated Go binding around an Ethereum contract.
type MockUSDT0 struct {
	MockUSDT0Caller     // Read-only binding to the contract
	MockUSDT0Transactor // Write-only binding to the contract
	MockUSDT0Filterer   // Log filterer for contract events
}

// MockUSDT0Caller is an auto generated read-only Go binding around an Ethereum contract.
type MockUSDT0Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockUSDT0Transactor is an auto generated write-only Go binding around an Ethereum contract.
type MockUSDT0Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockUSDT0Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MockUSDT0Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockUSDT0Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MockUSDT0Session struct {
	Contract     *MockUSDT0        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MockUSDT0CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MockUSDT0CallerSession struct {
	Contract *MockUSDT0Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// MockUSDT0TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MockUSDT0TransactorSession struct {
	Contract     *MockUSDT0Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// MockUSDT0Raw is an auto generated low-level Go binding around an Ethereum contract.
type MockUSDT0Raw struct {
	Contract *MockUSDT0 // Generic contract binding to access the raw methods on
}

// MockUSDT0CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MockUSDT0CallerRaw struct {
	Contract *MockUSDT0Caller // Generic read-only contract binding to access the raw methods on
}

// MockUSDT0TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MockUSDT0TransactorRaw struct {
	Contract *MockUSDT0Transactor // Generic write-only contract binding to access the raw methods on
}

// NewMockUSDT0 creates a new instance of MockUSDT0, bound to a specific deployed contract.
func NewMockUSDT0(address common.Address, backend bind.ContractBackend) (*MockUSDT0, error) {
	contract, err := bindMockUSDT0(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MockUSDT0{MockUSDT0Caller: MockUSDT0Caller{contract: contract}, MockUSDT0Transactor: MockUSDT0Transactor{contract: contract}, MockUSDT0Filterer: MockUSDT0Filterer{contract: contract}}, nil
}

// NewMockUSDT0Caller creates a new read-only instance of MockUSDT0, bound to a specific deployed contract.
func NewMockUSDT0Caller(address common.Address, caller bind.ContractCaller) (*MockUSDT0Caller, error) {
	contract, err := bindMockUSDT0(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MockUSDT0Caller{contract: contract}, nil
}

// NewMockUSDT0Transactor creates a new write-only instance of MockUSDT0, bound to a specific deployed contract.
func NewMockUSDT0Transactor(address common.Address, transactor bind.ContractTransactor) (*MockUSDT0Transactor, error) {
	contract, err := bindMockUSDT0(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MockUSDT0Transactor{contract: contract}, nil
}

// NewMockUSDT0Filterer creates a new log filterer instance of MockUSDT0, bound to a specific deployed contract.
func NewMockUSDT0Filterer(address common.Address, filterer bind.ContractFilterer) (*MockUSDT0Filterer, error) {
	contract, err := bindMockUSDT0(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MockUSDT0Filterer{contract: contract}, nil
}

// bindMockUSDT0 binds a generic wrapper to an already deployed contract.
func bindMockUSDT0(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MockUSDT0MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockUSDT0 *MockUSDT0Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockUSDT0.Contract.MockUSDT0Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockUSDT0 *MockUSDT0Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockUSDT0.Contract.MockUSDT0Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockUSDT0 *MockUSDT0Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockUSDT0.Contract.MockUSDT0Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockUSDT0 *MockUSDT0CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockUSDT0.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockUSDT0 *MockUSDT0TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockUSDT0.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockUSDT0 *MockUSDT0TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockUSDT0.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_MockUSDT0 *MockUSDT0Caller) Allowance(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MockUSDT0.contract.Call(opts, &out, "allowance", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_MockUSDT0 *MockUSDT0Session) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _MockUSDT0.Contract.Allowance(&_MockUSDT0.CallOpts, arg0, arg1)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_MockUSDT0 *MockUSDT0CallerSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _MockUSDT0.Contract.Allowance(&_MockUSDT0.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_MockUSDT0 *MockUSDT0Caller) BalanceOf(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MockUSDT0.contract.Call(opts, &out, "balanceOf", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_MockUSDT0 *MockUSDT0Session) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _MockUSDT0.Contract.BalanceOf(&_MockUSDT0.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_MockUSDT0 *MockUSDT0CallerSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _MockUSDT0.Contract.BalanceOf(&_MockUSDT0.CallOpts, arg0)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MockUSDT0 *MockUSDT0Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _MockUSDT0.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MockUSDT0 *MockUSDT0Session) Decimals() (uint8, error) {
	return _MockUSDT0.Contract.Decimals(&_MockUSDT0.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MockUSDT0 *MockUSDT0CallerSession) Decimals() (uint8, error) {
	return _MockUSDT0.Contract.Decimals(&_MockUSDT0.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MockUSDT0 *MockUSDT0Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MockUSDT0.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MockUSDT0 *MockUSDT0Session) Name() (string, error) {
	return _MockUSDT0.Contract.Name(&_MockUSDT0.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MockUSDT0 *MockUSDT0CallerSession) Name() (string, error) {
	return _MockUSDT0.Contract.Name(&_MockUSDT0.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_MockUSDT0 *MockUSDT0Caller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MockUSDT0.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_MockUSDT0 *MockUSDT0Session) Owner() (common.Address, error) {
	return _MockUSDT0.Contract.Owner(&_MockUSDT0.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_MockUSDT0 *MockUSDT0CallerSession) Owner() (common.Address, error) {
	return _MockUSDT0.Contract.Owner(&_MockUSDT0.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MockUSDT0 *MockUSDT0Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MockUSDT0.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MockUSDT0 *MockUSDT0Session) Symbol() (string, error) {
	return _MockUSDT0.Contract.Symbol(&_MockUSDT0.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MockUSDT0 *MockUSDT0CallerSession) Symbol() (string, error) {
	return _MockUSDT0.Contract.Symbol(&_MockUSDT0.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MockUSDT0 *MockUSDT0Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockUSDT0.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MockUSDT0 *MockUSDT0Session) TotalSupply() (*big.Int, error) {
	return _MockUSDT0.Contract.TotalSupply(&_MockUSDT0.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MockUSDT0 *MockUSDT0CallerSession) TotalSupply() (*big.Int, error) {
	return _MockUSDT0.Contract.TotalSupply(&_MockUSDT0.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0Transactor) Approve(opts *bind.TransactOpts, _spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.contract.Transact(opts, "approve", _spender, _amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0Session) Approve(_spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.Approve(&_MockUSDT0.TransactOpts, _spender, _amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0TransactorSession) Approve(_spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.Approve(&_MockUSDT0.TransactOpts, _spender, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_MockUSDT0 *MockUSDT0Transactor) Mint(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.contract.Transact(opts, "mint", _to, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_MockUSDT0 *MockUSDT0Session) Mint(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.Mint(&_MockUSDT0.TransactOpts, _to, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_MockUSDT0 *MockUSDT0TransactorSession) Mint(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.Mint(&_MockUSDT0.TransactOpts, _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0Transactor) Transfer(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.contract.Transact(opts, "transfer", _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0Session) Transfer(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.Transfer(&_MockUSDT0.TransactOpts, _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0TransactorSession) Transfer(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.Transfer(&_MockUSDT0.TransactOpts, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0Transactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.contract.Transact(opts, "transferFrom", _from, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0Session) TransferFrom(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.TransferFrom(&_MockUSDT0.TransactOpts, _from, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool)
func (_MockUSDT0 *MockUSDT0TransactorSession) TransferFrom(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MockUSDT0.Contract.TransferFrom(&_MockUSDT0.TransactOpts, _from, _to, _amount)
}

// MockUSDT0ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the MockUSDT0 contract.
type MockUSDT0ApprovalIterator struct {
	Event *MockUSDT0Approval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MockUSDT0ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockUSDT0Approval)
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
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MockUSDT0Approval)
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

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MockUSDT0ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockUSDT0ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockUSDT0Approval represents a Approval event raised by the MockUSDT0 contract.
type MockUSDT0Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_MockUSDT0 *MockUSDT0Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*MockUSDT0ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _MockUSDT0.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &MockUSDT0ApprovalIterator{contract: _MockUSDT0.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_MockUSDT0 *MockUSDT0Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *MockUSDT0Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _MockUSDT0.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockUSDT0Approval)
				if err := _MockUSDT0.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_MockUSDT0 *MockUSDT0Filterer) ParseApproval(log types.Log) (*MockUSDT0Approval, error) {
	event := new(MockUSDT0Approval)
	if err := _MockUSDT0.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MockUSDT0TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the MockUSDT0 contract.
type MockUSDT0TransferIterator struct {
	Event *MockUSDT0Transfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MockUSDT0TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockUSDT0Transfer)
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
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MockUSDT0Transfer)
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

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MockUSDT0TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockUSDT0TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockUSDT0Transfer represents a Transfer event raised by the MockUSDT0 contract.
type MockUSDT0Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_MockUSDT0 *MockUSDT0Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*MockUSDT0TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MockUSDT0.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &MockUSDT0TransferIterator{contract: _MockUSDT0.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_MockUSDT0 *MockUSDT0Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *MockUSDT0Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MockUSDT0.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockUSDT0Transfer)
				if err := _MockUSDT0.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_MockUSDT0 *MockUSDT0Filterer) ParseTransfer(log types.Log) (*MockUSDT0Transfer, error) {
	event := new(MockUSDT0Transfer)
	if err := _MockUSDT0.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
