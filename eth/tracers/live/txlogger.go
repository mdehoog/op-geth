package live

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
)

func init() {
	tracers.LiveDirectory.Register("txlogger", newTxLoggerTracer)
}

type txloggerConfig struct {
	MaxAge int `json:"maxAge"` // Any transactions older than MaxAge are skipped (if 0, all transactions are logged)
}

type txlogger struct {
	config  txloggerConfig
	tx      *types.Transaction
	from    common.Address
	start   time.Time
	reads   uint64
	writes  uint64
	calls   uint64
	logs    uint64
	creates uint64
	faults  uint64
}

func newTxLoggerTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config txloggerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %v", err)
		}
	}

	t := &txlogger{
		config: config,
	}
	return &tracing.Hooks{
		OnTxStart: t.OnTxStart,
		OnTxEnd:   t.OnTxEnd,
		OnOpcode:  t.OnOpcode,
		OnFault:   t.OnFault,
	}, nil
}

func (t *txlogger) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if t.tx == nil {
		return
	}
	switch vm.OpCode(op) {
	case vm.SLOAD, vm.BALANCE, vm.EXTCODEHASH, vm.EXTCODESIZE, vm.EXTCODECOPY:
		t.reads++
	case vm.SSTORE:
		t.writes++
	case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
		t.logs++
	case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
		t.calls++
	case vm.CREATE, vm.CREATE2:
		t.creates++
	}
}

func (t *txlogger) OnFault(pc uint64, op byte, gas, cost uint64, _ tracing.OpContext, depth int, err error) {
	t.faults++
}

func (t *txlogger) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.start = time.Now()
	if t.start.Unix()-int64(vm.Time) > int64(t.config.MaxAge) {
		// transaction is too old, skip
		t.tx = nil
		return
	}
	t.tx = tx
	t.from = from
	t.reads, t.writes, t.calls, t.logs, t.creates, t.faults = 0, 0, 0, 0, 0, 0
}

func (t *txlogger) OnTxEnd(receipt *types.Receipt, err error) {
	if t.tx == nil {
		return
	}
	duration := time.Since(t.start)
	to := ""
	if t.tx.To() != nil {
		to = t.tx.To().Hex()
	}

	// efficiency is defined as gas used per nanosecond
	efficiency := float64(receipt.GasUsed) / float64(duration.Nanoseconds())

	log.Info(
		"OnTxEnd",
		"hash", t.tx.Hash().Hex(),
		"from", t.from.Hex(),
		"to", to,
		"value", t.tx.Value().String(),
		"size", len(t.tx.Data()),
		"nonce", t.tx.Nonce(),
		"gas", receipt.GasUsed,
		"price", t.tx.GasPrice().String(),
		"duration", duration.Nanoseconds(),
		"efficiency", efficiency,
		"reads", t.reads,
		"writes", t.writes,
		"calls", t.calls,
		"logs", t.logs,
		"creates", t.creates,
		"faults", t.faults,
		"error", err,
	)
}
