package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const canCreateGasLimit = uint64(1000000)

// `bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)`
var proxyImplementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

// Predeploy address (may or may not have an implementation behind it)
var letPredeploy = common.HexToAddress("0x42000000000000000000000000000000000001e7")

// 4-byte signature of `canCreate(address contractCaller)` + 12-byte padding
var canCreatePrefix = common.FromHex("0x7804a5dc000000000000000000000000")

func CanCreate(evm *EVM, caller ContractRef) error {
	implementation := evm.StateDB.GetState(letPredeploy, proxyImplementationSlot)
	if implementation == (common.Hash{}) {
		return nil
	}
	var contractCaller common.Address
	if contract, ok := caller.(*Contract); ok {
		contractCaller = contract.Caller()
	}
	data := append(canCreatePrefix, contractCaller[:]...)
	_, _, err := evm.Call(caller, letPredeploy, data, canCreateGasLimit, new(uint256.Int))
	return err
}
