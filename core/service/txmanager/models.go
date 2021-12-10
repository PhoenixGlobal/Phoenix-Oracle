package txmanager

import (
	"bytes"
	"fmt"
	"time"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"gopkg.in/guregu/null.v4"
	"gorm.io/datatypes"
)

type EthTxMeta struct {
	JobID         int32
	RequestID     common.Hash
	RequestTxHash common.Hash
}

func (EthTxMeta) GormDataType() string {
	return "json"
}

type EthTxState string
type EthTxAttemptState string

const (
	EthTxUnstarted               = EthTxState("unstarted")
	EthTxInProgress              = EthTxState("in_progress")
	EthTxFatalError              = EthTxState("fatal_error")
	EthTxUnconfirmed             = EthTxState("unconfirmed")
	EthTxConfirmed               = EthTxState("confirmed")
	EthTxConfirmedMissingReceipt = EthTxState("confirmed_missing_receipt")

	EthTxAttemptInProgress      = EthTxAttemptState("in_progress")
	EthTxAttemptInsufficientEth = EthTxAttemptState("insufficient_eth")
	EthTxAttemptBroadcast       = EthTxAttemptState("broadcast")
)

type EthTx struct {
	ID             int64
	Nonce          *int64
	FromAddress    common.Address
	ToAddress      common.Address
	EncodedPayload []byte
	Value          assets.Eth
	GasLimit uint64
	Error    null.String
	BroadcastAt   *time.Time
	CreatedAt     time.Time
	State         EthTxState
	EthTxAttempts []EthTxAttempt `gorm:"->"`
	Meta    datatypes.JSON
	Subject uuid.NullUUID
}

func (e EthTx) GetError() error {
	if e.Error.Valid {
		return errors.New(e.Error.String)
	}
	return nil
}

func (e EthTx) GetID() string {
	return fmt.Sprintf("%d", e.ID)
}

type EthTxAttempt struct {
	ID       int64
	EthTxID  int64
	EthTx    EthTx `gorm:"foreignkey:EthTxID;->"`
	GasPrice utils.Big
	// ChainSpecificGasLimit on the EthTxAttempt is always the same as the on-chain encoded value for gas limit
	ChainSpecificGasLimit   uint64
	SignedRawTx             []byte
	Hash                    common.Hash
	CreatedAt               time.Time
	BroadcastBeforeBlockNum *int64
	State                   EthTxAttemptState
	EthReceipts             []EthReceipt `gorm:"foreignKey:TxHash;references:Hash;association_foreignkey:Hash;->"`
}

func (a EthTxAttempt) GetSignedTx() (*types.Transaction, error) {
	s := rlp.NewStream(bytes.NewReader(a.SignedRawTx), 0)
	signedTx := new(types.Transaction)
	if err := signedTx.DecodeRLP(s); err != nil {
		logger.Error("could not decode RLP")
		return nil, err
	}
	return signedTx, nil
}

type EthReceipt struct {
	ID               int64
	TxHash           common.Hash
	BlockHash        common.Hash
	BlockNumber      int64
	TransactionIndex uint
	Receipt          []byte
	CreatedAt        time.Time
}
