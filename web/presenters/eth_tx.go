package presenters

import (
	"strconv"

	"PhoenixOracle/core/service/txmanager"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type EthTxResource struct {
	JAID
	State    string          `json:"state"`
	Data     hexutil.Bytes   `json:"data"`
	From     *common.Address `json:"from"`
	GasLimit string          `json:"gasLimit"`
	GasPrice string          `json:"gasPrice"`
	Hash     common.Hash     `json:"hash"`
	Hex      string          `json:"rawHex"`
	Nonce    string          `json:"nonce"`
	SentAt   string          `json:"sentAt"`
	To       *common.Address `json:"to"`
	Value    string          `json:"value"`
}

func (EthTxResource) GetName() string {
	return "transactions"
}

func NewEthTxResource(tx txmanager.EthTx) EthTxResource {
	return EthTxResource{
		Data:     hexutil.Bytes(tx.EncodedPayload),
		From:     &tx.FromAddress,
		GasLimit: strconv.FormatUint(tx.GasLimit, 10),
		State:    string(tx.State),
		To:       &tx.ToAddress,
		Value:    tx.Value.String(),
	}
}

func NewEthTxResourceFromAttempt(txa txmanager.EthTxAttempt) EthTxResource {
	tx := txa.EthTx

	r := NewEthTxResource(tx)
	r.JAID = NewJAID(txa.Hash.Hex())
	r.GasPrice = txa.GasPrice.String()
	r.Hash = txa.Hash
	r.Hex = hexutil.Encode(txa.SignedRawTx)

	if tx.Nonce != nil {
		r.Nonce = strconv.FormatUint(uint64(*tx.Nonce), 10)
	}
	if txa.BroadcastBeforeBlockNum != nil {
		r.SentAt = strconv.FormatUint(uint64(*txa.BroadcastBeforeBlockNum), 10)
	}
	return r
}
