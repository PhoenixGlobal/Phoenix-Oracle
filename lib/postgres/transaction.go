package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/sqlx"
	"gorm.io/gorm"
)

const (
	LockTimeout = 15 * time.Second

	IdleInTxSessionTimeout = 1 * time.Hour
)

var (
	ErrNoDeadlineSet = errors.New("no deadline set")
)

func GormTransactionWithoutContext(db *gorm.DB, fc func(tx *gorm.DB) error, txOptss ...sql.TxOptions) (err error) {
	var txOpts sql.TxOptions
	if len(txOptss) > 0 {
		txOpts = txOptss[0]
	} else {
		txOpts = DefaultSqlTxOptions
	}
	return db.Transaction(func(tx *gorm.DB) error {
		err = tx.Exec(fmt.Sprintf(`SET LOCAL lock_timeout = %v; SET LOCAL idle_in_transaction_session_timeout = %v;`, LockTimeout.Milliseconds(), IdleInTxSessionTimeout.Milliseconds())).Error
		if err != nil {
			return errors.Wrap(err, "error setting transaction timeouts")
		}
		return fc(tx)
	}, &txOpts)
}

func GormTransaction(ctx context.Context, db *gorm.DB, fc func(tx *gorm.DB) error, txOptss ...sql.TxOptions) (err error) {
	var txOpts sql.TxOptions
	if len(txOptss) > 0 {
		txOpts = txOptss[0]
	} else {
		txOpts = DefaultSqlTxOptions
	}
	if _, set := ctx.Deadline(); !set {
		return ErrNoDeadlineSet
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err = tx.Exec(fmt.Sprintf(`SET LOCAL lock_timeout = %v; SET LOCAL idle_in_transaction_session_timeout = %v;`, LockTimeout.Milliseconds(), IdleInTxSessionTimeout.Milliseconds())).Error
		if err != nil {
			return errors.Wrap(err, "error setting transaction timeouts")
		}
		return fc(tx)
	}, &txOpts)
}

func GormTransactionWithDefaultContext(db *gorm.DB, fc func(tx *gorm.DB) error, txOptss ...sql.TxOptions) error {
	var txOpts sql.TxOptions
	if len(txOptss) > 0 {
		txOpts = txOptss[0]
	} else {
		txOpts = DefaultSqlTxOptions
	}
	ctx, cancel := DefaultQueryCtx()
	defer cancel()
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Exec(fmt.Sprintf(`SET LOCAL lock_timeout = %v; SET LOCAL idle_in_transaction_session_timeout = %v;`, LockTimeout.Milliseconds(), IdleInTxSessionTimeout.Milliseconds())).Error
		if err != nil {
			return errors.Wrap(err, "error setting transaction timeouts")
		}
		return fc(tx)
	}, &txOpts)
	return err
}

func DBWithDefaultContext(db *gorm.DB, fc func(db *gorm.DB) error) error {
	ctx, cancel := DefaultQueryCtx()
	defer cancel()
	return fc(db.WithContext(ctx))
}

func SqlTransaction(ctx context.Context, rdb *sql.DB, fc func(tx *sqlx.Tx) error, txOpts ...sql.TxOptions) (err error) {
	opts := &DefaultSqlTxOptions
	if len(txOpts) > 0 {
		opts = &txOpts[0]
	}
	db := WrapDbWithSqlx(rdb)

	tx, err := db.BeginTxx(ctx, opts)
	panicked := false

	defer func() {
		if panicked || err != nil {
			if perr := tx.Rollback(); perr != nil {
				panic(perr)
			}
		}
	}()

	_, err = tx.Exec(fmt.Sprintf(`SET LOCAL lock_timeout = %v; SET LOCAL idle_in_transaction_session_timeout = %v;`, LockTimeout.Milliseconds(), IdleInTxSessionTimeout.Milliseconds()))
	if err != nil {
		return errors.Wrap(err, "error setting transaction timeouts")
	}

	panicked = true
	err = fc(tx)
	panicked = false

	if err == nil {
		err = errors.WithStack(tx.Commit())
	}

	return
}
