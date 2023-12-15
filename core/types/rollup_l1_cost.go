// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type L1CostData struct {
	zeroes, ones uint64
}

func NewL1CostData(data []byte) (out L1CostData) {
	for _, byte := range data {
		if byte == 0 {
			out.zeroes++
		} else {
			out.ones++
		}
	}
	return out
}

type StateGetter interface {
	GetState(common.Address, common.Hash) common.Hash
}

// L1CostFunc is used in the state transition to determine the L1 data fee charged to the sender of
// the transaction.
type L1CostFunc func(tx L1CostData) *big.Int

// l1CostFunc is an internal version of L1CostFunc that also returns the gasUsed for use
// in receipts.
type l1CostFunc func(tx L1CostData) (fee, gasUsed *big.Int)

var (
	L1BaseFeeSlot = common.BigToHash(big.NewInt(1))
	OverheadSlot  = common.BigToHash(big.NewInt(5))
	ScalarSlot    = common.BigToHash(big.NewInt(6))
)

var L1BlockAddr = common.HexToAddress("0x4200000000000000000000000000000000000015")

// NewL1CostFunc returns a function used for calculating L1 fee cost.  This depends on the oracles
// because gas costs can change over time, and depends on blockTime since the specific function
// used to compute the fee can differ between hardforks.
func NewL1CostFunc(config *params.ChainConfig, statedb StateGetter, blockTime uint64) L1CostFunc {
	l1BaseFee := statedb.GetState(L1BlockAddr, L1BaseFeeSlot).Big()
	overhead := statedb.GetState(L1BlockAddr, OverheadSlot).Big()
	scalar := statedb.GetState(L1BlockAddr, ScalarSlot).Big()
	f := newL1CostFunc(config, l1BaseFee, overhead, scalar, blockTime)
	return func(l1CostData L1CostData) *big.Int {
		fee, _ := f(l1CostData)
		return fee
	}
}

func newL1CostFunc(config *params.ChainConfig, l1BaseFee, overhead, scalar *big.Int, blockTime uint64) l1CostFunc {
	isRegolith := config.IsRegolith(blockTime)
	return func(l1CostData L1CostData) (fee, gasUsed *big.Int) {
		if config.Optimism == nil {
			return nil, nil
		}
		gas := l1CostData.zeroes * params.TxDataZeroGas
		if isRegolith {
			gas += l1CostData.ones * params.TxDataNonZeroGasEIP2028
		} else {
			gas += (l1CostData.ones + 68) * params.TxDataNonZeroGasEIP2028
		}
		l1GasUsed := new(big.Int).SetUint64(gas)
		l1GasUsed = l1GasUsed.Add(l1GasUsed, overhead)
		l1Cost := new(big.Int).Set(l1GasUsed)
		l1Cost.Mul(l1GasUsed, l1BaseFee).Mul(l1Cost, scalar).Div(l1Cost, big.NewInt(1_000_000))
		return l1Cost, l1GasUsed
	}
}
