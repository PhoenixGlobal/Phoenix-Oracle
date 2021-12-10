package postgres

import (
	"PhoenixOracle/db/models"
	"gorm.io/gorm"
)

const BatchSize uint = 1000

type BatchFunc func(offset, limit uint) (count uint, err error)

func Batch(cb BatchFunc) error {
	offset := uint(0)
	limit := BatchSize

	for {
		count, err := cb(offset, limit)
		if err != nil {
			return err
		}

		if count < limit {
			return nil
		}

		offset += limit
	}
}

func Sessions(db *gorm.DB, offset, limit int) ([]models.Session, error) {
	var sessions []models.Session
	err := db.
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error
	return sessions, err
}
