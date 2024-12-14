package ethapi

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

var ErrCreateUnauthorized = errors.New("unauthorized contract creation")

func (api *TransactionAPI) canSend(ctx context.Context, tx *types.Transaction, allowedCreators []common.Address) error {
	mapAllowedCreators := make(map[common.Address]bool)
	for _, creator := range allowedCreators {
		mapAllowedCreators[creator] = true
	}
	state, head, err := api.b.StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	if err != nil {
		return err
	}
	signer := types.MakeSigner(api.b.ChainConfig(), head.Number, head.Time)
	msg, err := core.TransactionToMessage(tx, signer, head.BaseFee)
	if err != nil {
		return err
	}
	if mapAllowedCreators[msg.From] {
		return nil
	}
	if msg.To == nil {
		return ErrCreateUnauthorized
	}

	var tracerError error
	t := &tracing.Hooks{
		OnOpcode: func(pc uint64, opcode byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			op := vm.OpCode(opcode)
			isCreate := op == vm.CREATE || op == vm.CREATE2
			if isCreate && !mapAllowedCreators[scope.Address()] && !mapAllowedCreators[scope.Caller()] {
				tracerError = ErrCreateUnauthorized
			}
		},
	}
	vmctx := core.NewEVMBlockContext(head, NewChainContext(ctx, api.b), nil, api.b.ChainConfig(), state)
	vmenv := vm.NewEVM(vmctx, vm.TxContext{GasPrice: msg.GasPrice, BlobFeeCap: msg.BlobGasFeeCap}, state, api.b.ChainConfig(), vm.Config{Tracer: t, NoBaseFee: true})

	var usedGas uint64
	_, err = core.ApplyTransactionWithEVM(msg, api.b.ChainConfig(), new(core.GasPool).AddGas(msg.GasLimit), state, vmctx.BlockNumber, common.Hash{}, tx, &usedGas, vmenv)
	if err != nil {
		return err
	}

	return tracerError
}
