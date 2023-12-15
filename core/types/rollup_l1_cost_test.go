package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestRollupGasData(t *testing.T) {
	for i := 0; i < 100; i++ {
		time := uint64(1)
		cfg := &params.ChainConfig{
			Optimism:     params.OptimismTestConfig.Optimism,
			RegolithTime: &time,
		}
		basefee := big.NewInt(1)
		overhead := big.NewInt(1)
		scalar := big.NewInt(1_000_000)

		costFunc0 := newL1CostFunc(cfg, basefee, overhead, scalar, 0)
		costFunc1 := newL1CostFunc(cfg, basefee, overhead, scalar, 1)

		emptyTx = NewTransaction(
			0,
			common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			big.NewInt(0), 0, big.NewInt(0),
			nil,
		)
		c0, _ := costFunc0(emptyTx.L1CostData())
		c1, _ := costFunc1(emptyTx.L1CostData())
		require.Equal(t, c0, big.NewInt(1569))
		require.Equal(t, c1, big.NewInt(481))
	}
}
