package txmanager

import (
	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"
)

type TxStrategy interface {
	Subject() uuid.NullUUID
	PruneQueue(tx *gorm.DB) (n int64, err error)
}

var _ TxStrategy = SendEveryStrategy{}

func NewQueueingTxStrategy(subject uuid.UUID, queueSize uint32) (strategy TxStrategy) {
	if queueSize > 0 {
		strategy = NewDropOldestStrategy(subject, queueSize)
	} else {
		strategy = SendEveryStrategy{}
	}
	return
}

type SendEveryStrategy struct{}

func (SendEveryStrategy) Subject() uuid.NullUUID             { return uuid.NullUUID{} }
func (SendEveryStrategy) PruneQueue(*gorm.DB) (int64, error) { return 0, nil }

var _ TxStrategy = DropOldestStrategy{}

type DropOldestStrategy struct {
	subject   uuid.UUID
	queueSize uint32
}

func NewDropOldestStrategy(subject uuid.UUID, queueSize uint32) DropOldestStrategy {
	return DropOldestStrategy{subject, queueSize}
}

func (s DropOldestStrategy) Subject() uuid.NullUUID {
	return uuid.NullUUID{UUID: s.subject, Valid: true}
}

func (s DropOldestStrategy) PruneQueue(tx *gorm.DB) (n int64, err error) {
	res := tx.Exec(`
DELETE FROM eth_txes
WHERE state = 'unstarted' AND subject = ? AND
id < (
	SELECT min(id) FROM (
		SELECT id
		FROM eth_txes
		WHERE state = 'unstarted' AND subject = ?
		ORDER BY id DESC
		LIMIT ?
	) numbers
)`, s.subject, s.subject, s.queueSize)
	return res.RowsAffected, res.Error
}
