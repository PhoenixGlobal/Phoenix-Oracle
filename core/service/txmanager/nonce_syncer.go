package txmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"PhoenixOracle/core/keystore/keys/ethkey"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/postgres"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gorm.io/gorm"
)

type (
	NonceSyncer struct {
		db        *gorm.DB
		ethClient ethereum.Client
	}
	NSinserttx struct {
		Etx     EthTx
		Attempt EthTxAttempt
	}
)

func NewNonceSyncer(db *gorm.DB, ethClient ethereum.Client) *NonceSyncer {
	return &NonceSyncer{
		db,
		ethClient,
	}
}

func (s NonceSyncer) SyncAll(ctx context.Context, keys []ethkey.KeyV2) (merr error) {
	var wg sync.WaitGroup
	var errMu sync.Mutex

	wg.Add(len(keys))
	for _, key := range keys {
		go func(k ethkey.KeyV2) {
			defer wg.Done()
			if err := s.fastForwardNonceIfNecessary(ctx, k.Address.Address()); err != nil {
				errMu.Lock()
				defer errMu.Unlock()
				merr = multierr.Combine(merr, err)
			}
		}(key)
	}

	wg.Wait()

	return errors.Wrap(merr, "NonceSyncer#fastForwardNoncesIfNecessary failed")
}

func (s NonceSyncer) fastForwardNonceIfNecessary(ctx context.Context, address common.Address) error {
	chainNonce, err := s.pendingNonceFromEthClient(ctx, address)
	if err != nil {
		return errors.Wrap(err, "GetNextNonce failed to loadInitialNonceFromEthClient")
	}
	if chainNonce == 0 {
		return nil
	}

	selectCtx, cancel := postgres.DefaultQueryCtx()
	defer cancel()
	keyNextNonce, err := GetNextNonce(s.db.WithContext(selectCtx), address)
	if err != nil {
		return err
	}

	localNonce := keyNextNonce
	hasInProgressTransaction, err := s.hasInProgressTransaction(address)
	if err != nil {
		return errors.Wrapf(err, "failed to query for in_progress transaction for address %s", address.Hex())
	} else if hasInProgressTransaction {
		localNonce++
	}
	if chainNonce <= uint64(localNonce) {
		return nil
	}
	logger.Warnw(fmt.Sprintf("NonceSyncer: address %s has been used before, either by an external wallet or a different Phoenix node. "+
		"Local nonce is %v but the on-chain nonce for this account was %v. "+
		"It's possible that this node was restored from a backup. If so, transactions sent by the previous node will NOT be re-org protected and in rare cases may need to be manually bumped/resubmitted. "+
		"Please note that using the phoenix keys with an external wallet is NOT SUPPORTED and can lead to missed or stuck transactions. ",
		address.Hex(), localNonce, chainNonce),
		"address", address.Hex(), "keyNextNonce", keyNextNonce, "localNonce", localNonce, "chainNonce", chainNonce)

	// Need to remember to decrement the chain nonce by one to account for in_progress transaction
	newNextNonce := chainNonce
	if hasInProgressTransaction {
		newNextNonce--
	}
	return postgres.DBWithDefaultContext(s.db, func(db *gorm.DB) error {
		res := db.Exec(`UPDATE eth_key_states SET next_nonce = ?, updated_at = ? WHERE address = ? AND next_nonce = ?`, newNextNonce, time.Now(), address, keyNextNonce)
		if res.Error != nil {
			return errors.Wrap(res.Error, "NonceSyncer#fastForwardNonceIfNecessary failed to update keys.next_nonce")
		}
		if res.RowsAffected == 0 {
			return errors.Errorf("NonceSyncer#fastForwardNonceIfNecessary optimistic lock failure fastforwarding nonce %v to %v for key %s", localNonce, chainNonce, address.Hex())
		}
		return nil
	})
}

func (s NonceSyncer) pendingNonceFromEthClient(ctx context.Context, account common.Address) (nextNonce uint64, err error) {
	ctx, cancel := ethereum.DefaultQueryCtx(ctx)
	defer cancel()
	nextNonce, err = s.ethClient.PendingNonceAt(ctx, account)
	return nextNonce, errors.WithStack(err)
}

func (s NonceSyncer) hasInProgressTransaction(account common.Address) (exists bool, err error) {
	err = postgres.DBWithDefaultContext(s.db, func(db *gorm.DB) error {
		return db.Raw(`SELECT EXISTS(SELECT 1 FROM eth_txes WHERE state = 'in_progress' AND from_address = ?)`, account).Scan(&exists).Error
	})
	return
}
