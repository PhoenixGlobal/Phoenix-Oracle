package models

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type Tx struct {
	ID       uint64 `storm:"id,increment,index"`
	From     common.Address
	To       common.Address
	Data     []byte
	Nonce    uint64
	Value    *big.Int
	GasLimit uint64
	TxAttempt
}

func (self *Tx) EthTx(gasPrice *big.Int) *types.Transaction {
	return types.NewTransaction(
		self.Nonce,
		self.To,
		self.Value,
		self.GasLimit,
		gasPrice,
		self.Data,
	)
}

type TxAttempt struct {
	Hash      common.Hash `storm:"id,index,unique"`
	TxID      uint64 `storm:"index"`
	GasPrice  *big.Int
	Confirmed bool
	Hex       string
	SentAt    uint64
}

